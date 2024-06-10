package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"codeberg.org/anaseto/goal"
	gos "codeberg.org/anaseto/goal/os"
	resty "github.com/go-resty/resty/v2"
	"github.com/knz/bubbline"
	"github.com/knz/bubbline/complete"
	"github.com/knz/bubbline/computil"
	"github.com/knz/bubbline/editline"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// For now, we only support DuckDB
const dbDriver = "duckdb"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ari",
	Short: "ari - Array relational interactive environment",
	Long: `ari is an interactive environment for array + relational programming.

It embeds the Goal array programming language, with extensions for
working with SQL and HTTP APIs.`,
	Run: func(cmd *cobra.Command, args []string) {
		replMain()
	},
}

type evalMode int

const (
	modeGoalEval = iota
	modeSqlEvalReadOnly
	modeSqlEvalReadWrite
)

const (
	modeGoalPrompt             = "goal> "
	modeGoalNextPrompt         = "    > "
	modeSqlReadOnlyPrompt      = "sql> "
	modeSqlReadOnlyNextPrompt  = "   > "
	modeSqlReadWritePrompt     = "sql!> "
	modeSqlReadWriteNextPrompt = "    > "
)

type AriContext struct {
	editor      *bubbline.Editor
	evalMode    evalMode
	goalContext *goal.Context
	sqlDatabase *SqlDatabase
}

// See `replMain` for full initialization of this Context.
//
// This is defined at the top level so that functions which have fixed signatures (e.g., those which are registered as backing definitions for Goal functions) can access it. Functions should still accept an AriContext directly where possible for better testing.
var globalAriContext = AriContext{}

func replMain() {
	editorInitialize(&globalAriContext)
	modeGoalInitialize(&globalAriContext)
	modeGoalSetReplDefaults(&globalAriContext)

	// REPL starts here
	for {
		line, err := globalAriContext.editor.GetLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			if errors.Is(err, bubbline.ErrInterrupted) {
				// Entered Ctrl+C to cancel input.
				fmt.Println("^C")
			} else {
				fmt.Println("error:", err)
			}
			continue
		}
		// Add to REPL history, even if not a legal expression (thus before we try to evaluate)
		err = globalAriContext.editor.AddHistory(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write REPL history, error: %v\n", err)
		}

		// Support system commands
		if matchesSystemCommand(line) {
			cmdAndArgs := strings.Split(line, " ")
			systemCommand := cmdAndArgs[0]
			switch systemCommand {
			case ")goal":
				modeSqlClose(&globalAriContext) // NB: Include in every non-"sql" case
				modeGoalSetReplDefaults(&globalAriContext)
			case ")sql": // TODO Accept data source name as argument
				var mode evalMode
				mode = modeSqlEvalReadOnly
				modeSqlInitialize(&globalAriContext, mode)
				modeSqlSetReplDefaults(&globalAriContext, mode)
			case ")sql!": // TODO Accept data source name as argument
				var mode evalMode
				mode = modeSqlEvalReadWrite
				modeSqlInitialize(&globalAriContext, mode)
				modeSqlSetReplDefaults(&globalAriContext, mode)
			default:
				fmt.Fprintf(os.Stderr, "Unsupported system command '%v'\n", systemCommand)
			}
			continue
		}

		// Future: Consider user commands with ]

		switch globalAriContext.evalMode {
		case modeGoalEval:
			value, err := globalAriContext.goalContext.Eval(line)
			if err != nil {
				fmt.Fprint(os.Stderr, err)
			}
			// Suppress printing values for variable assignments
			if !globalAriContext.goalContext.AssignedLast() {
				fmt.Fprintln(os.Stdout, value.Sprint(globalAriContext.goalContext, false))
			}

		case modeSqlEvalReadOnly:
			_, err := modeSqlRunQuery(&globalAriContext, line, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to run SQL query %q\nDatabase Error:%s\n", line, err)
			} else {
				_, err := globalAriContext.goalContext.Eval(`fmt.tbl[sql.t;*#'sql.t;#sql.t;"%.1f"]`)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to print SQL query results via Goal evaluation: %v\n", err)
				}
			}
		case modeSqlEvalReadWrite:
			_, err := modeSqlRunExec(&globalAriContext, line, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to run SQL query %q\nDatabase Error:%s\n", line, err)
			} else {
				_, err := globalAriContext.goalContext.Eval(`fmt.tbl[sql.t;*#'sql.t;#sql.t;"%.1f"]`)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to print SQL exec results via Goal evaluation: %v\n", err)
				}
			}
		}
	}
}

func editorInitialize(ariContext *AriContext) {
	editor := bubbline.New()
	editor.Placeholder = ""
	editor.Reflow = func(x bool, y string, _ int) (bool, string, string) {
		return editline.DefaultReflow(x, y, 80)
	}
	historyFile := viper.GetString("history")
	if err := editor.LoadHistory(historyFile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load history, error: %v\n", err)
	}
	editor.SetAutoSaveHistory(historyFile, true)
	editor.SetDebugEnabled(true)
	editor.SetExternalEditorEnabled(true, "goal")
	ariContext.editor = editor
}

func modeGoalInitialize(ariContext *AriContext) {
	goalContext := goal.NewContext()
	goalContext.Log = os.Stderr
	goalRegisterVariadics(goalContext)
	_ = goalLoadExtendedPreamble(goalContext)
	ariContext.goalContext = goalContext
}

// Caller is expected to close everything
func modeSqlInitialize(ariContext *AriContext, evalMode evalMode) {
	dataSourceName := viper.GetString("database")
	if evalMode == modeSqlEvalReadOnly && len(dataSourceName) != 0 {
		// In-memory doesn't support read_only access
		dataSourceName = dataSourceName + "?access_mode=read_only"
	}
	db, err := sql.Open(dbDriver, dataSourceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database data source %v, error: %v\n", dataSourceName, err)
	}
	// NB: No `defer db.Close()`, caller must close.
	ariContext.sqlDatabase = &SqlDatabase{db, true}
}

func modeSqlClose(ariContext *AriContext) {
	if ariContext.sqlDatabase != nil {
		ariContext.sqlDatabase.db.Close()
		ariContext.sqlDatabase.isOpen = false
	}
}

func sqlRunQuery(sqlDatabase *SqlDatabase, sqlQuery string, args []any) (goal.V, error) {
	var rows *sql.Rows
	var err error
	if len(args) > 0 {
		rows, err = sqlDatabase.db.QueryContext(context.Background(), sqlQuery, args...)
	} else {
		rows, err = sqlDatabase.db.QueryContext(context.Background(), sqlQuery)
	}
	if err != nil {
		return goal.V{}, err
	}
	defer rows.Close()
	colNames, err := rows.Columns()
	if err != nil {
		return goal.V{}, err
	}
	// NB: For future type introspection, see below.
	// colTypes, err := rows.ColumnTypes()
	// if err != nil {
	// 	panic(err)
	// }
	cols := make([]interface{}, len(colNames))
	colPtrs := make([]interface{}, len(colNames))
	rowValues := make([][]goal.V, len(cols))
	for i := 0; i < len(colNames); i++ {
		colPtrs[i] = &cols[i]
	}
	for rows.Next() {
		err = rows.Scan(colPtrs...)
		if err != nil {
			return goal.V{}, err
		}
		for i, col := range cols {
			// fmt.Printf("SQL %v // Go %v\n", colTypes[i].DatabaseTypeName(), reflect.TypeOf(col))
			if col == nil {
				rowValues[i] = append(rowValues[i], goal.NewF(math.NaN()))
			} else {
				switch col := col.(type) {
				case bool:
					if col {
						rowValues[i] = append(rowValues[i], goal.NewF(math.Inf(1)))
					} else {
						rowValues[i] = append(rowValues[i], goal.NewF(math.Inf(-1)))
					}
				case float32:
					rowValues[i] = append(rowValues[i], goal.NewF(float64(col)))
				case float64:
					rowValues[i] = append(rowValues[i], goal.NewF(col))
				case int32:
					rowValues[i] = append(rowValues[i], goal.NewI(int64(col)))
				case int64:
					rowValues[i] = append(rowValues[i], goal.NewI(col))
				case string:
					rowValues[i] = append(rowValues[i], goal.NewS(col))
				case time.Time:
					// From DuckDB docs: https://duckdb.org/docs/sql/data_types/timestamp
					// "DuckDB represents instants as the number of microseconds (µs) since 1970-01-01 00:00:00+00"
					rowValues[i] = append(rowValues[i], goal.NewI(col.UnixMicro()))
				default:
					fmt.Printf("Go Type %v\n", reflect.TypeOf(col))
					rowValues[i] = append(rowValues[i], goal.NewS(fmt.Sprintf("%v", col)))
				}
			}
		}
	}
	// NB: For future type introspection, see above.
	// fmt.Printf("COLS %v\n", colNames)
	// fmt.Printf("ROWS %v\n", rowValues)
	dictVs := make([]goal.V, len(rowValues))
	for i, slc := range rowValues {
		// NB: Goal's underlying NewArray function specializes for us.
		dictVs[i] = goal.NewAV(slc)
	}
	goalD := goal.NewD(goal.NewAS(colNames), goal.NewAV(dictVs))
	return goalD, nil
}

func modeSqlRunQuery(ariContext *AriContext, sqlQuery string, args []any) (goal.V, error) {
	if ariContext.sqlDatabase == nil || !ariContext.sqlDatabase.isOpen {
		modeSqlInitialize(ariContext, modeSqlEvalReadOnly)
	}
	goalD, err := sqlRunQuery(ariContext.sqlDatabase, sqlQuery, args)
	if err != nil {
		return goal.V{}, err
	}
	// Last result table as sql.t in Goal, to support switching eval mdoes:
	ariContext.goalContext.AssignGlobal("sql.t", goalD)
	return goalD, nil
}

func modeSqlRunExec(ariContext *AriContext, sqlQuery string, args []any) (goal.V, error) {
	if ariContext.sqlDatabase == nil || !ariContext.sqlDatabase.isOpen {
		modeSqlInitialize(ariContext, modeSqlEvalReadWrite)
	}
	result, err := ariContext.sqlDatabase.db.Exec(sqlQuery, args...)
	if err != nil {
		return goal.V{}, err
	}
	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return goal.V{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return goal.V{}, err
	}
	// Consider: Formatting this as a table for consistency with )sql, but it's really a simpler dict
	ks := goal.NewAS([]string{"lastInsertId", "rowsAffected"})
	// vs := goal.NewAI([]int64{lastInsertId, rowsAffected})
	vs := goal.NewAV([]goal.V{goal.NewAI([]int64{lastInsertId}), goal.NewAI([]int64{rowsAffected})})
	goalD := goal.NewD(ks, vs)
	ariContext.goalContext.AssignGlobal("sql.t", goalD)
	return goalD, nil
}

// Copied from Goal's implementation
func panicType(op, sym string, x goal.V) goal.V {
	return goal.Panicf("%s : bad type %q in %s", op, x.Type(), sym)
}

type SqlDatabase struct {
	db     *sql.DB
	isOpen bool
}

// Append implements goal.BV.
func (sqlDatabase *SqlDatabase) Append(ctx *goal.Context, dst []byte, compact bool) []byte {
	return append(dst, fmt.Sprintf("<%v %#v>", sqlDatabase.Type(), sqlDatabase.db)...)
}

// LessT implements goal.BV.
func (sqlDatabase *SqlDatabase) LessT(y goal.BV) bool {
	// Goal falls back to ordering by type name,
	// and there is no other reasonable way to order
	// these HttpClient structs.
	return sqlDatabase.Type() < y.Type()
}

// Matches implements goal.BV.
func (sqlDatabase *SqlDatabase) Matches(y goal.BV) bool {
	switch yv := y.(type) {
	case *SqlDatabase:
		return sqlDatabase.db == yv.db
	default:
		return false
	}
}

// Type implements goal.BV.
func (sqlDatabase *SqlDatabase) Type() string {
	return "ari.SqlDatabase"
}

// Implements sql.open
func VFSqlOpen(goalContext *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	dataSourceName, ok := x.BV().(goal.S)
	switch len(args) {
	case 1:
		if !ok {
			return panicType("sql.open s", "s", x)
		}
		dsn := string(dataSourceName)
		if len(dsn) == 0 {
			// Empty means use the value from config/CLI args
			dsn = viper.GetString("database")
		}
		db, err := sql.Open(dbDriver, dsn)
		if err != nil {
			return goal.NewPanicError(err)
		}
		sqlDb := SqlDatabase{db, true}
		return goal.NewV(&sqlDb)
	default:
		return goal.Panicf("sql.open : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements sql.q
func VFSqlQ(goalContext *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	sqlQuery, ok := x.BV().(goal.S)
	switch len(args) {
	case 1:
		// Uses the database configured at the Ari level
		if !ok {
			return panicType("sql.q s", "s", x)
		}
		queryResult, err := modeSqlRunQuery(&globalAriContext, string(sqlQuery), nil)
		if err != nil {
			return goal.Errorf("%v", err)
		}
		return queryResult
	case 2:
		// Explicit database as first argument
		sqlDatabase, ok := x.BV().(*SqlDatabase)
		if !ok {
			return panicType("ari.SqlDatabase sql.q s", "ari.SqlDatabase", x)
		}
		y := args[0]
		sqlQuery, ok := y.BV().(goal.S)
		if !ok {
			return panicType("ari.SqlDatabase sql.q s", "s", y)
		}
		queryResult, err := sqlRunQuery(sqlDatabase, string(sqlQuery), nil)
		if err != nil {
			return goal.Errorf("%v", err)
		}
		return queryResult
	default:
		return goal.Panicf("sql.q : too many arguments (%d), expects 1 or 2 arguments", len(args))
	}
}

// HTTP via go-resty

type HttpClient struct {
	client *resty.Client
}

// LessT implements goal.BV.
func (httpClient *HttpClient) LessT(y goal.BV) bool {
	// Goal falls back to ordering by type name,
	// and there is no other reasonable way to order
	// these HttpClient structs.
	return httpClient.Type() < y.Type()
}

// Matches implements goal.BV.
func (httpClient *HttpClient) Matches(y goal.BV) bool {
	switch yv := y.(type) {
	case *HttpClient:
		return httpClient.client == yv.client
	default:
		return false
	}
}

// Type implements goal.BV.
func (httpClient *HttpClient) Type() string {
	return "ari.HttpClient"
}

// Append implements goal.BV
func (httpClient *HttpClient) Append(goalContext *goal.Context, dst []byte, compact bool) []byte {
	// Go prints nil as `<nil>` so following suit.
	return append(dst, fmt.Sprintf("<%v %#v>", httpClient.Type(), httpClient.client)...)
}

func newHttpClient(optionsD *goal.D) (*HttpClient, error) {
	// [DONE] BaseURL               string
	// [DONE] QueryParam            url.Values //  type Values map[string][]string
	// [DONE] FormData              url.Values
	// [DONE] PathParams            map[string]string
	// [DONE] RawPathParams         map[string]string
	// [DONE] Header                http.Header // Use Add methods; accept dictionary of either single strings or []string
	// [DONE] HeaderAuthorizationKey string
	// [DONE] UserInfo              *User // Struct of Username, Password string
	// [DONE] Token                 string
	// [DONE] AuthScheme            string
	// Cookies               []*http.Cookie // Medium-sized struct
	// Error                 reflect.Type
	// [DONE] Debug                 bool
	// [DONE] DisableWarn           bool
	// [DONE] AllowGetMethodPayload bool
	// [DONE] RetryCount            int
	// [DONE] RetryWaitTime         time.Duration // Pick canonical unit (millis/micros) int64
	// [DONE] RetryMaxWaitTime      time.Duration int64
	// RetryConditions       []RetryConditionFunc // Research: How tough is it to invoke a Goal lambda from Go land?
	// RetryHooks            []OnRetryFunc
	// RetryAfter            RetryAfterFunc
	// [DONE] RetryResetReaders     bool
	// JSONMarshal           func(v interface{}) ([]byte, error)
	// JSONUnmarshal         func(data []byte, v interface{}) error
	// XMLMarshal            func(v interface{}) ([]byte, error)
	// XMLUnmarshal          func(data []byte, v interface{}) error
	restyClient := resty.New()
	if optionsD.Len() == 0 {
		return &HttpClient{resty.New()}, nil
	} else {
		ka := optionsD.KeyArray()
		va := optionsD.ValueArray()
		switch kas := ka.(type) {
		case *goal.AS:
			for i, k := range kas.Slice {
				value := va.At(i)
				switch k {
				case "AllowGetMethodPayload":
					if value.IsTrue() {
						restyClient.AllowGetMethodPayload = true
					} else if value.IsFalse() {
						restyClient.AllowGetMethodPayload = false
					} else {
						return nil, fmt.Errorf("http.client expects \"AllowGetMethodPayload\" to be 0 or 1 (falsey/truthy), but received: %v\n", value)
					}
				case "AuthScheme":
					switch goalV := value.BV().(type) {
					case goal.S:
						restyClient.AuthScheme = string(goalV)
					default:
						return nil, fmt.Errorf("http.client expects \"AuthScheme\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "BaseUrl":
					switch goalV := value.BV().(type) {
					case goal.S:
						restyClient.BaseURL = string(goalV)
					default:
						return nil, fmt.Errorf("http.client expects \"BaseUrl\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "Debug":
					if value.IsTrue() {
						restyClient.Debug = true
					} else if value.IsFalse() {
						restyClient.Debug = false
					} else {
						return nil, fmt.Errorf("http.client expects \"Debug\" to be 0 or 1, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "DisableWarn":
					if value.IsTrue() {
						restyClient.DisableWarn = true
					} else if value.IsFalse() {
						restyClient.DisableWarn = false
					} else {
						return nil, fmt.Errorf("http.client expects \"DisableWarn\" to be 0 or 1 (falsey/truthy), but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "FormData":
					switch goalV := value.BV().(type) {
					case (*goal.D):
						formDataKeys := goalV.KeyArray()
						formDataValues := goalV.ValueArray()
						switch fdks := formDataKeys.(type) {
						case (*goal.AS):
							urlValues := make(url.Values, fdks.Len())
							for hvi := 0; hvi < formDataValues.Len(); hvi++ {
								for i, hk := range fdks.Slice {
									formDataValue := formDataValues.At(i)
									switch hv := formDataValue.BV().(type) {
									case (goal.S):
										urlValues.Add(hk, string(hv))
									case (*goal.AS):
										for _, w := range hv.Slice {
											urlValues.Add(hk, w)
										}
									default:
										return nil, fmt.Errorf("http.client expects \"FormData\" to be a dictionary with values that are strings or lists of strings, but received a %v: %v\n", reflect.TypeOf(hv), hv)
									}
								}
							}
							restyClient.FormData = urlValues
						default:
							return nil, fmt.Errorf("http.client expects \"FormData\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(fdks), fdks)
						}
					default:
						return nil, fmt.Errorf("http.client expects \"FormData\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "Header":
					switch goalV := value.BV().(type) {
					case (*goal.D):
						headerKeys := goalV.KeyArray()
						headerValues := goalV.ValueArray()
						switch hks := headerKeys.(type) {
						case (*goal.AS):
							hd := make(http.Header, hks.Len())
							for hvi := 0; hvi < headerValues.Len(); hvi++ {
								for i, hk := range hks.Slice {
									headerValue := headerValues.At(i)
									switch hv := headerValue.BV().(type) {
									case (goal.S):
										hd.Add(hk, string(hv))
									case (*goal.AS):
										for _, w := range hv.Slice {
											hd.Add(hk, w)
										}
									default:
										return nil, fmt.Errorf("http.client expects \"Header\" to be a dictionary with values that are strings or lists of strings, but received a %v: %v\n", reflect.TypeOf(hv), hv)
									}
								}
							}
							restyClient.Header = hd
						default:
							return nil, fmt.Errorf("http.client expects \"Header\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(hks), hks)
						}
					default:
						return nil, fmt.Errorf("http.client expects \"Header\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "HeaderAuthorizationKey":
					switch goalV := value.BV().(type) {
					case goal.S:
						restyClient.HeaderAuthorizationKey = string(goalV)
					default:
						return nil, fmt.Errorf("http.client expects \"HeaderAuthorizationKey\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "PathParams":
					switch goalV := value.BV().(type) {
					case *goal.D:
						pathParams, err := stringMapFromGoalDict(goalV)
						if err != nil {
							return nil, err
						}
						restyClient.PathParams = pathParams
					default:
						return nil, fmt.Errorf("http.client expects \"PathParams\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "QueryParam":
					switch goalV := value.BV().(type) {
					case (*goal.D):
						queryParamKeys := goalV.KeyArray()
						queryParamValues := goalV.ValueArray()
						switch qpks := queryParamKeys.(type) {
						case (*goal.AS):
							urlValues := make(url.Values, qpks.Len())
							for qpvi := 0; qpvi < queryParamValues.Len(); qpvi++ {
								for i, hk := range qpks.Slice {
									queryParamValue := queryParamValues.At(i)
									switch hv := queryParamValue.BV().(type) {
									case (goal.S):
										urlValues.Add(hk, string(hv))
									case (*goal.AS):
										for _, w := range hv.Slice {
											urlValues.Add(hk, w)
										}
									default:
										return nil, fmt.Errorf("http.client expects \"QueryParam\" to be a dictionary with values that are strings or lists of strings, but received a %v: %v\n", reflect.TypeOf(hv), hv)
									}
								}
							}
							restyClient.QueryParam = urlValues
						default:
							return nil, fmt.Errorf("http.client expects \"QueryParam\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(qpks), qpks)
						}
					default:
						return nil, fmt.Errorf("http.client expects \"QueryParam\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "RawPathParams":
					switch goalV := value.BV().(type) {
					case *goal.D:
						pathParams, err := stringMapFromGoalDict(goalV)
						if err != nil {
							return nil, err
						}
						restyClient.RawPathParams = pathParams
					default:
						return nil, fmt.Errorf("http.client expects \"RawPathParams\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "RetryCount":
					if value.IsI() {
						restyClient.RetryCount = int(value.I())
					} else {
						return nil, fmt.Errorf("http.client expects \"RetryCount\" to be an integer, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "RetryMaxWaitTimeMilli":
					if value.IsI() {
						restyClient.RetryMaxWaitTime = time.Duration(value.I()) * time.Millisecond
					} else {
						return nil, fmt.Errorf("http.client expects \"RetryMaxWaitTimeMilli\" to be an integer, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "RetryResetReaders":
					if value.IsTrue() {
						restyClient.RetryResetReaders = true
					} else if value.IsFalse() {
						restyClient.RetryResetReaders = false
					} else {
						return nil, fmt.Errorf("http.client expects \"RetryResetReaders\" to be 0 or 1 (falsey/truthy), but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "RetryWaitTimeMilli":
					if value.IsI() {
						restyClient.RetryWaitTime = time.Duration(value.I()) * time.Millisecond
					} else {
						return nil, fmt.Errorf("http.client expects \"RetryWaitTimeMilli\" to be an integer, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "Token":
					switch goalV := value.BV().(type) {
					case goal.S:
						restyClient.Token = string(goalV)
					default:
						return nil, fmt.Errorf("http.client expects \"Token\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				case "UserInfo":
					switch goalV := value.BV().(type) {
					case (*goal.D):
						userInfoKeys := goalV.KeyArray()
						userInfoValues := goalV.ValueArray()
						switch uiks := userInfoKeys.(type) {
						case (*goal.AS):
							switch uivs := userInfoValues.(type) {
							case (*goal.AS):
								userInfo := resty.User{}
								for i, uik := range uiks.Slice {
									switch uik {
									case "Username":
										userInfo.Username = uivs.Slice[i]
									case "Password":
										userInfo.Password = uivs.Slice[i]
									default:
										return nil, fmt.Errorf("Unsupported \"UserInfo\" key: %v\n", uik)
									}
								}
								restyClient.UserInfo = &userInfo
							default:
								return nil, fmt.Errorf("http.client expects \"UserInfo\" to be a dictionary with string values, but received a %v: %v\n", reflect.TypeOf(uivs), uivs)
							}
						default:
							return nil, fmt.Errorf("http.client expects \"UserInfo\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(uiks), uiks)
						}
					default:
						return nil, fmt.Errorf("http.client expects \"UserInfo\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
					}
				default:
					return nil, fmt.Errorf("Unsupported ari.HttpClient option: %v\n", k)
				}
			}
		default:
			return nil, fmt.Errorf("http.client expects a Goal dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(va), va)
		}
	}
	return &HttpClient{client: restyClient}, nil
}

func stringMapFromGoalDict(d *goal.D) (map[string]string, error) {
	ka := d.KeyArray()
	va := d.ValueArray()
	m := make(map[string]string, ka.Len())
	switch kas := ka.(type) {
	case *goal.AS:
		switch vas := va.(type) {
		case *goal.AS:
			vasSlice := vas.Slice
			for i, k := range kas.Slice {
				m[k] = vasSlice[i]
			}
		default:
			return nil, fmt.Errorf("[Developer Error] stringMapFromGoalDict expects a Goal dict with string keys and string values, but received values: %v\n", va)
		}
	default:
		return nil, fmt.Errorf("[Developer Error] stringMapFromGoalDict expects a Goal dict with string keys and string values, but received keys: %v\n", ka)
	}
	return m, nil
}

func VFHttpClient(goalContext *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	clientOptions, ok := x.BV().(*goal.D)
	switch len(args) {
	case 1:
		if !ok {
			return panicType("http.client d", "d", x)
		}
		hc, err := newHttpClient(clientOptions)
		if err != nil {
			return goal.NewPanicError(err)
		}
		return goal.NewV(hc)
	default:
		return goal.NewPanic("http.client : too many arguments")
	}
}

func VFHTTPMaker(method string) func(goalContext *goal.Context, args []goal.V) goal.V {
	methodLower := strings.ToLower(method)
	methodUpper := strings.ToUpper(method)
	return func(goalContext *goal.Context, args []goal.V) goal.V {
		x := args[len(args)-1]
		switch len(args) {
		case 1:
			url, ok := x.BV().(goal.S)
			if !ok {
				return panicType(fmt.Sprintf("http.%s s", methodLower), "s", x)
			}
			httpClient, err := newHttpClient(&goal.D{})
			if err != nil {
				return goal.NewPanicError(err)
			}
			req := httpClient.client.R()
			resp, err := req.Execute(methodUpper, string(url))
			if err != nil {
				fmt.Fprintf(os.Stderr, "HTTP error: %v\n", err)
				// Continue
				// error response built below, with
				// "ok" as false
			}
			// Construct goal.V values for return dict
			return goalDictFromResponse(resp)
		case 2:
			var httpClient *HttpClient
			switch clientOpts := x.BV().(type) {
			case *HttpClient:
				httpClient = clientOpts
			case *goal.D:
				var err error
				httpClient, err = newHttpClient(clientOpts)
				if err != nil {
					return goal.NewPanicError(err)
				}
			default:
				return goal.NewPanic(fmt.Sprintf("client http.%s url : client must be a dict or HttpClient instance, but received a %v: %v", methodLower, reflect.TypeOf(clientOpts), clientOpts))
			}
			y := args[0]
			urlS, ok := y.BV().(goal.S)
			if !ok {
				return panicType(fmt.Sprintf("HttpClient http.%s url", methodLower), "url", y)
			}
			req := httpClient.client.R()
			resp, err := req.Execute(methodUpper, string(urlS))
			if err != nil {
				fmt.Fprintf(os.Stderr, "HTTP error: %v\n", err)
				// Continue
				// error response built below, with
				// "ok" as false
			}
			return goalDictFromResponse(resp)
		case 3:
			var httpClient *HttpClient
			switch clientOpts := x.BV().(type) {
			case *HttpClient:
				httpClient = clientOpts
			case *goal.D:
				var err error
				httpClient, err = newHttpClient(clientOpts)
				if err != nil {
					return goal.NewPanicError(err)
				}
			default:
				return goal.NewPanic(fmt.Sprintf("client http.%s url optionsDict : client must be a dict or HttpClient instance, but received a %v: %v", methodLower, reflect.TypeOf(clientOpts), clientOpts))
			}
			y := args[1]
			urlS, ok := y.BV().(goal.S)
			if !ok {
				return panicType(fmt.Sprintf("HttpClient http.%s url", methodLower), "url", y)
			}
			z := args[0]
			optionsD, ok := z.BV().(*goal.D)
			if !ok {
				return panicType(fmt.Sprintf("http.%s[HttpClient;url;optionsDict]", methodLower), "optionsDict", z)
			}
			req := httpClient.client.R()
			req, err := augmentRequestWithOptions(req, optionsD)
			if err != nil {
				return goal.NewPanicError(err)
			}
			resp, err := req.Execute(methodUpper, string(urlS))
			if err != nil {
				fmt.Fprintf(os.Stderr, "HTTP error: %v\n", err)
				// Continue
				// error response built below, with
				// "ok" as false
			}
			return goalDictFromResponse(resp)
		default:
			return goal.Panicf("http.%s : too many arguments (%d), expects 1, 2, or 3 arguments", methodLower, len(args))
		}
	}
}

func goalDictFromResponse(resp *resty.Response) goal.V {
	statusS := goal.NewS(resp.Status())
	headers := resp.Header()
	headerKeysSlice := make([]string, 0)
	headerValuesSlice := make([]goal.V, 0)
	for k, vs := range headers {
		headerKeysSlice = append(headerKeysSlice, k)
		valuesAS := goal.NewAS(vs)
		headerValuesSlice = append(headerValuesSlice, valuesAS)
	}
	headerD := goal.NewD(goal.NewAS(headerKeysSlice), goal.NewAV(headerValuesSlice))
	bodyS := goal.NewS(resp.String())
	var isOk goal.V
	if resp.IsSuccess() {
		isOk = goal.NewI(1)
	} else {
		isOk = goal.NewI(0)
	}
	ks := goal.NewAS([]string{"status", "headers", "string", "ok"})
	vs := goal.NewAV([]goal.V{statusS, headerD, bodyS, isOk})
	// TODO Consider whether this should be a pointer.
	return goal.NewD(ks, vs)
}

func augmentRequestWithOptions(req *resty.Request, optionsD *goal.D) (*resty.Request, error) {
	optionsKeys := optionsD.KeyArray()
	optionsValues := optionsD.ValueArray()
	switch kas := optionsKeys.(type) {
	case (*goal.AS):
		for i, k := range kas.Slice {
			value := optionsValues.At(i)
			switch k {
			case "QueryParam":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					queryParamKeys := goalV.KeyArray()
					queryParamValues := goalV.ValueArray()
					switch qpks := queryParamKeys.(type) {
					case (*goal.AS):
						urlValues := make(url.Values, qpks.Len())
						for qpvi := 0; qpvi < queryParamValues.Len(); qpvi++ {
							for i, hk := range qpks.Slice {
								queryParamValue := queryParamValues.At(i)
								switch hv := queryParamValue.BV().(type) {
								case (goal.S):
									urlValues.Add(hk, string(hv))
								case (*goal.AS):
									for _, w := range hv.Slice {
										urlValues.Add(hk, w)
									}
								default:
									return nil, fmt.Errorf("http.client expects \"QueryParam\" to be a dictionary with values that are strings or lists of strings, but received a %v: %v\n", reflect.TypeOf(hv), hv)
								}
							}
						}
						req.QueryParam = urlValues
					default:
						return nil, fmt.Errorf("http.client expects \"QueryParam\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(qpks), qpks)
					}
				default:
					return nil, fmt.Errorf("http.client expects \"QueryParam\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "FormData":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					formDataKeys := goalV.KeyArray()
					formDataValues := goalV.ValueArray()
					switch fdks := formDataKeys.(type) {
					case (*goal.AS):
						urlValues := make(url.Values, fdks.Len())
						for hvi := 0; hvi < formDataValues.Len(); hvi++ {
							for i, hk := range fdks.Slice {
								formDataValue := formDataValues.At(i)
								switch hv := formDataValue.BV().(type) {
								case (goal.S):
									urlValues.Add(hk, string(hv))
								case (*goal.AS):
									for _, w := range hv.Slice {
										urlValues.Add(hk, w)
									}
								default:
									return nil, fmt.Errorf("http.client expects \"FormData\" to be a dictionary with values that are strings or lists of strings, but received a %v: %v\n", reflect.TypeOf(hv), hv)
								}
							}
						}
						req.FormData = urlValues
					default:
						return nil, fmt.Errorf("http.client expects \"FormData\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(fdks), fdks)
					}
				default:
					return nil, fmt.Errorf("http.client expects \"FormData\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "Header":
				switch goalV := value.BV().(type) {
				case (*goal.D):
					headerKeys := goalV.KeyArray()
					headerValues := goalV.ValueArray()
					switch hks := headerKeys.(type) {
					case (*goal.AS):
						hd := make(http.Header, hks.Len())
						for hvi := 0; hvi < headerValues.Len(); hvi++ {
							for i, hk := range hks.Slice {
								headerValue := headerValues.At(i)
								switch hv := headerValue.BV().(type) {
								case (goal.S):
									hd.Add(hk, string(hv))
								case (*goal.AS):
									for _, w := range hv.Slice {
										hd.Add(hk, w)
									}
								default:
									return nil, fmt.Errorf("http.client expects \"Header\" to be a dictionary with values that are strings or lists of strings, but received a %v: %v\n", reflect.TypeOf(hv), hv)
								}
							}
						}
						req.Header = hd
					default:
						return nil, fmt.Errorf("http.client expects \"Header\" to be a dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(hks), hks)
					}
				default:
					return nil, fmt.Errorf("http.client expects \"Header\" to be a dictionary, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "Cookies":
				panic("not yet implemented")
			case "PathParams":
				switch goalV := value.BV().(type) {
				case *goal.D:
					pathParams, err := stringMapFromGoalDict(goalV)
					if err != nil {
						return nil, err
					}
					req.PathParams = pathParams
				default:
					return nil, fmt.Errorf("http.client expects \"PathParams\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "RawPathParams":
				switch goalV := value.BV().(type) {
				case *goal.D:
					pathParams, err := stringMapFromGoalDict(goalV)
					if err != nil {
						return nil, err
					}
					req.RawPathParams = pathParams
				default:
					return nil, fmt.Errorf("http.client expects \"RawPathParams\" to be a string, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			case "Debug":
				if value.IsTrue() {
					req.Debug = true
				} else if value.IsFalse() {
					req.Debug = false
				} else {
					return nil, fmt.Errorf("http.client expects \"Debug\" to be 0 or 1, but received a %v: %v\n", reflect.TypeOf(value), value)
				}
			default:
				return nil, fmt.Errorf("Unsupported resty.Request option: %v\n", k)
			}
		}
	default:
		return nil, fmt.Errorf("http.client expects a Goal dictionary with string keys, but received a %v: %v\n", reflect.TypeOf(kas), kas)
	}
	return req, nil
}

// When the REPL mode is switched to Goal, this resets the proper defaults.
func modeGoalSetReplDefaults(ariContext *AriContext) {
	ariContext.evalMode = modeGoalEval
	ariContext.editor.Prompt = modeGoalPrompt
	ariContext.editor.NextPrompt = modeGoalNextPrompt
	ariContext.editor.AutoComplete = modeGoalAutocompleteFn(ariContext.goalContext)
	ariContext.editor.CheckInputComplete = nil
	ariContext.editor.SetExternalEditorEnabled(true, "goal")
}

// When the REPL mode is switched to SQL, this resets the proper defaults. Separate modes for read-only and read/write SQL evaluation.
func modeSqlSetReplDefaults(ariContext *AriContext, evalMode evalMode) {
	ariContext.evalMode = evalMode
	ariContext.editor.CheckInputComplete = modeSqlCheckInputComplete
	ariContext.editor.AutoComplete = modeSqlAutocomplete
	ariContext.editor.SetExternalEditorEnabled(true, "sql")
	switch ariContext.evalMode {
	case modeSqlEvalReadOnly:
		ariContext.editor.Prompt = modeSqlReadOnlyPrompt
		ariContext.editor.NextPrompt = modeSqlReadOnlyNextPrompt
	case modeSqlEvalReadWrite:
		ariContext.editor.Prompt = modeSqlReadWritePrompt
		ariContext.editor.NextPrompt = modeSqlReadWriteNextPrompt
	}
}

func matchesSystemCommand(s string) bool {
	return strings.HasPrefix(s, ")")
}

func modeSqlCheckInputComplete(v [][]rune, line, _ int) bool {
	if len(v) == 1 && matchesSystemCommand(string(v[0])) {
		return true
	}
	if line == len(v)-1 && // Enter on last row.
		strings.HasSuffix(string(v[len(v)-1]), ";") { // Semicolon at end of last row.
		return true
	}
	return false
}

func modeGoalAutocompleteFn(goalContext *goal.Context) func(v [][]rune, line, col int) (msg string, completions editline.Completions) {
	goalNameRe := regexp.MustCompile(`[a-zA-Z\.]+`)
	return func(v [][]rune, line, col int) (msg string, completions editline.Completions) {
		candidatesPerCategory := map[string][]acEntry{}
		word, start, end := computil.FindWord(v, line, col)
		// N.B. Matching system commands must come first.
		if matchesSystemCommand(word) {
			acSystemCommandCandidates(strings.ToLower(word), candidatesPerCategory)
		} else {
			locs := goalNameRe.FindStringIndex(word)
			if locs != nil {
				word = word[locs[0]:locs[1]]
				start = locs[0] // Preserve non-word prefix
				end = locs[1]   // Preserve non-word suffix
			}
			// msg = fmt.Sprintf("Matching %v", word)
			lword := strings.ToLower(word)
			goalGlobals := goalContext.GlobalNames(nil)
			category := "Global"
			for _, goalGlobal := range goalGlobals {
				if strings.HasPrefix(strings.ToLower(goalGlobal), lword) {
					var help string
					if val, ok := goalGlobalsHelp[goalGlobal]; ok {
						help = val
					} else {
						help = "A Goal global binding"
					}
					candidatesPerCategory[category] = append(candidatesPerCategory[category], acEntry{goalGlobal, help})
				}
			}
			goalKeywords := goalKeywords(goalContext)
			category = "Keyword"
			for _, goalKeyword := range goalKeywords {
				if strings.HasPrefix(strings.ToLower(goalKeyword), lword) {
					var help string
					if val, ok := goalKeywordsHelp[goalKeyword]; ok {
						help = val
					} else {
						help = "A Goal keyword"
					}
					candidatesPerCategory[category] = append(candidatesPerCategory[category], acEntry{goalKeyword, help})
				}
			}
			category = "Syntax"
			syntaxSet := make(map[string]bool, 0)
			for name, chstr := range goalSyntax {
				if strings.HasPrefix(strings.ToLower(name), lword) {
					if _, ok := syntaxSet[chstr]; !ok {
						syntaxSet[chstr] = true
						var help string
						if val, ok := goalSyntaxHelp[chstr]; ok {
							help = val
						} else {
							help = "Goal syntax"
						}
						candidatesPerCategory[category] = append(candidatesPerCategory[category], acEntry{chstr, help})
					}
				}
			}
		}
		// msg = fmt.Sprintf("Type is %v")
		completions = &multiComplete{
			Values:     complete.MapValues(candidatesPerCategory, nil),
			moveRight:  end - col,
			deleteLeft: end - start,
		}
		return msg, completions
	}
}

type acEntry struct {
	name, description string
}

func (e acEntry) Title() string {
	return e.name
}

func (e acEntry) Description() string {
	return "\n" + e.description
}

func modeSqlAutocomplete(v [][]rune, line, col int) (msg string, completions editline.Completions) {
	word, wstart, wend := computil.FindWord(v, line, col)
	// msg = fmt.Sprintf("Matching '%v'", word)
	candidatesPerCategory := map[string][]acEntry{}
	lword := strings.ToLower(word)
	// N.B. Matching system commands must come first.
	acSystemCommandCandidates(lword, candidatesPerCategory)
	for _, sqlWord := range acSqlKeywords {
		if strings.HasPrefix(strings.ToLower(sqlWord), lword) {
			candidatesPerCategory["sql"] = append(candidatesPerCategory["sql"], acEntry{sqlWord, "SQL help"})
		}
	}
	completions = &multiComplete{
		Values:     complete.MapValues(candidatesPerCategory, nil),
		moveRight:  wend - col,
		deleteLeft: wend - wstart,
	}
	return msg, completions
}

func acSystemCommandCandidates(lword string, candidatesPerCategory map[string][]acEntry) {
	if matchesSystemCommand(lword) {
		for _, mode := range acModeSystemCommandsKeys {
			if len(lword) == 0 || strings.HasPrefix(strings.ToLower(mode), lword) {
				candidatesPerCategory["mode"] = append(candidatesPerCategory["mode"], acEntry{mode, acModeSystemCommands[mode]})
			}
		}
	}
}

type multiComplete struct {
	complete.Values
	moveRight, deleteLeft int
}

func (m *multiComplete) Candidate(e complete.Entry) editline.Candidate {
	return candidate{e.Title(), m.moveRight, m.deleteLeft}
}

type candidate struct {
	repl                  string
	moveRight, deleteLeft int
}

func (m candidate) Replacement() string { return m.repl }
func (m candidate) MoveRight() int      { return m.moveRight }
func (m candidate) DeleteLeft() int     { return m.deleteLeft }

var acModeSystemCommands = map[string]string{
	")goal": "Goal array language mode", ")sql": "Read-only SQL mode (querying)", ")sql!": "Read/write SQL mode",
}

var acModeSystemCommandsKeys = func() []string {
	keys := make([]string, len(acModeSystemCommands))
	for k := range acModeSystemCommands {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}()

func goalRegisterVariadics(goalContext *goal.Context) {
	// From Goal itself
	gos.Import(goalContext, "")
	// Ari
	goalContext.RegisterDyad("http.client", VFHttpClient)
	goalContext.RegisterDyad("http.get", VFHTTPMaker("GET"))
	goalContext.RegisterDyad("http.post", VFHTTPMaker("POST"))
	goalContext.RegisterDyad("http.put", VFHTTPMaker("PUT"))
	goalContext.RegisterDyad("http.delete", VFHTTPMaker("DELETE"))
	goalContext.RegisterDyad("http.patch", VFHTTPMaker("PATCH"))
	goalContext.RegisterDyad("http.head", VFHTTPMaker("HEAD"))
	goalContext.RegisterDyad("http.options", VFHTTPMaker("OPTIONS"))
	goalContext.RegisterDyad("sql.open", VFSqlOpen)
	goalContext.RegisterDyad("sql.q", VFSqlQ)
}

// CLI (Cobra, Viper)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	cfgDir := path.Join(home, ".config", "ari")

	defaultHistFile := path.Join(cfgDir, "ari-history.txt")
	defaultCfgFile := path.Join(cfgDir, "ari-config.yaml")

	// Config file has processing in initConfig outside of viper lifecycle, so it's a separate variable.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", defaultCfgFile, "ari configuration")

	// Everything else should go through viper for consistency.
	pFlags := rootCmd.PersistentFlags()

	flagNameHistory := "history"
	flagNameDatabase := "database"

	pFlags.String(flagNameHistory, defaultHistFile, "history of REPL entries")
	pFlags.String(flagNameDatabase, "", "DuckDB database (default: in-memory)")

	err = viper.BindPFlag(flagNameHistory, pFlags.Lookup(flagNameHistory))
	cobra.CheckErr(err)
	err = viper.BindPFlag(flagNameDatabase, pFlags.Lookup(flagNameDatabase))
	cobra.CheckErr(err)

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		cfgDir := path.Join(home, ".config", "ari")
		err = os.MkdirAll(cfgDir, 0o755)
		cobra.CheckErr(err)

		viper.AddConfigPath(cfgDir)
		viper.SetConfigName("ari-config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "[INFO] Using config file:", viper.ConfigFileUsed())
	}
}
