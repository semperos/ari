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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ari",
	Short: "ari - Array relational interactive environment",
	Long: `ari is an interactive environment for array + relational programming.

It embeds the Goal array programming language along with extensions for accessing
DuckDB and SQLite for interacting with a local relational database.`,
	Run: func(cmd *cobra.Command, args []string) {
		// initDb()
		// goalMain()
		// cliMain()
		replMain()
	},
}

type evalMode int

const (
	modeGoalEval = iota
	modeSqlEvalReadOnly
	modeSqlEvalReadWrite
)

// TODO Support Prolog https://github.com/ichiban/prolog?tab=readme-ov-file
const (
	modeGoalPrompt             = "goal> "
	modeGoalNextPrompt         = "    > "
	modeSqlReadOnlyPrompt      = "sql> "
	modeSqlReadOnlyNextPrompt  = "   > "
	modeSqlReadWritePrompt     = "sql!> "
	modeSqlReadWriteNextPrompt = "    > "
)

type Context struct {
	editor      *bubbline.Editor
	evalMode    evalMode
	goalContext *goal.Context
	sqlDb       *sql.DB
	sqlDbConn   *sql.Conn
}

// See `replMain` for full initialization of this Context
var ctx = Context{}

func replMain() {
	editorInitialize(&ctx)
	modeGoalInitialize(&ctx)
	modeSqlInitialize(&ctx, modeSqlEvalReadOnly)
	// Goal is the default mode.
	modeGoalSetReplDefaults(&ctx)

	// Read-print loop starts here.
	for {
		line, err := ctx.editor.GetLine()
		if err != nil {
			if err == io.EOF {
				// No more input.
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
		// Add to REPL history, even if not a legal expression
		err = ctx.editor.AddHistory(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write REPL history, error: %v\n", err)
		}

		// Support system commands
		if matchesSystemCommand(line) {
			cmdAndArgs := strings.Split(line, " ")
			systemCommand := cmdAndArgs[0]
			switch systemCommand {
			case ")goal":
				modeSqlClose(&ctx) // NB: Include in every non-"sql" case
				modeGoalSetReplDefaults(&ctx)
			case ")sql":
				var mode evalMode
				mode = modeSqlEvalReadOnly
				modeSqlInitialize(&ctx, mode)
				modeSqlSetReplDefaults(&ctx, mode)
			case ")sql!":
				var mode evalMode
				mode = modeSqlEvalReadWrite
				modeSqlInitialize(&ctx, mode)
				modeSqlSetReplDefaults(&ctx, mode)
			default:
				fmt.Fprintf(os.Stderr, "Unsupported system command '%v'\n", systemCommand)
			}
			continue
		}

		// Future: Support user commands with ]

		switch ctx.evalMode {
		case modeGoalEval:
			value, err := ctx.goalContext.Eval(line)
			if err != nil {
				fmt.Fprint(os.Stderr, err)
			}
			// Suppress printing the value when assigning values to variables.
			if !ctx.goalContext.AssignedLast() {
				fmt.Fprintln(os.Stdout, value.Sprint(ctx.goalContext, false))
			}

		case modeSqlEvalReadOnly:
			_, err := modeSqlRunQuery(&ctx, line, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to run SQL query %q\nDatabase Error:%s\n", line, err)
			} else {
				_, err := ctx.goalContext.Eval(`fmt.tbl[sql.t;*#'sql.t;#sql.t;"%.1f"]`)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to print SQL query results via Goal evaluation: %v\n", err)
				}
			}
		case modeSqlEvalReadWrite:
			_, err := modeSqlRunExec(&ctx, line, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to run SQL query %q\nDatabase Error:%s\n", line, err)
			} else {
				// Consider: Making this a table for consistency, but it's better as a dict.
				_, err := ctx.goalContext.Eval(`fmt.tbl[sql.t;*#'sql.t;#sql.t;"%.1f"]`)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to print SQL exec results via Goal evaluation: %v\n", err)
				}
			}
		}
	}
}

func editorInitialize(ctx *Context) {
	editor := bubbline.New()
	editor.Placeholder = ""
	editor.Reflow = func(x bool, y string, _ int) (bool, string, string) {
		return editline.DefaultReflow(x, y, 80)
	}
	// TODO History file from configuration and configure history.
	// TODO Separate history for each language mode (maybe a good idea?)
	historyFile := ".ari.history"
	if err := editor.LoadHistory(historyFile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load history, error: %v\n", err)
	}
	editor.SetAutoSaveHistory(historyFile, true)
	editor.SetDebugEnabled(true)
	editor.SetExternalEditorEnabled(true, "goal")
	ctx.editor = editor
}

func modeGoalInitialize(ctx *Context) {
	// Goal
	goalCtx := goal.NewContext()
	goalCtx.Log = os.Stderr
	goalRegisterVariadics(goalCtx)
	// TODO Consider making this optional to load
	_ = goalLoadExtendedPreamble(goalCtx)
	ctx.goalContext = goalCtx
}

// Caller is expected to close both ctx.sqlDb and ctx.sqlDbConn
func modeSqlInitialize(ctx *Context, evalMode evalMode) {
	var exampleDbFile string
	switch evalMode {
	case modeSqlEvalReadOnly:
		exampleDbFile = "/Users/dlg/dev/julia/work-product-data/work_2023.duckdb?access_mode=read_only"
	case modeSqlEvalReadWrite:
		exampleDbFile = "/Users/dlg/dev/julia/work-product-data/work_2023.duckdb"
	}
	dbDriver := "duckdb"                         // TODO Configurable AND accept via system command args
	db, err := sql.Open(dbDriver, exampleDbFile) // TODO Configurable // TODO FIXME
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open in-memory %v database, error: %v", dbDriver, err) // TODO Include path if persistent
	}
	// defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to in-memory %v database, error: %v", dbDriver, err)
	}
	// defer conn.Close()
	ctx.sqlDb = db
	ctx.sqlDbConn = conn
}

func modeSqlClose(ctx *Context) {
	if ctx.sqlDb != nil {
		ctx.sqlDb.Close()
	}
	if ctx.sqlDbConn != nil {
		ctx.sqlDbConn.Close()
	}
}

// TODO sql.sq for "statement query"
// TODO sql.se for "statement execute"
func modeSqlRunQuery(ctx *Context, sqlQuery string, args []any) (goal.V, error) {
	var rows *sql.Rows
	var err error
	// TODO Probably unnecessary
	if len(args) > 0 {
		rows, err = ctx.sqlDb.QueryContext(context.Background(), sqlQuery, args...)
	} else {
		rows, err = ctx.sqlDb.QueryContext(context.Background(), sqlQuery)
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
	// Last result table as sql.t in Goal, to support switching eval mdoes:
	ctx.goalContext.AssignGlobal("sql.t", goalD)
	return goalD, nil
}

func modeSqlRunExec(ctx *Context, sqlQuery string, args []any) (goal.V, error) {
	result, err := ctx.sqlDb.Exec(sqlQuery, args...)
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
	ctx.goalContext.AssignGlobal("sql.t", goalD)
	return goalD, nil
}

// Copied from Goal's implementation
func panicType(op, sym string, x goal.V) goal.V {
	return goal.Panicf("%s : bad type %q in %s", op, x.Type(), sym)
}

// TODO Dyad for sql.e
// TODO Consider whether a parallel set of these fns should be `cq` and `ce` for "connection query" and "connection execute" and accept a connection as an arg
// Goal variadic function for SQL query
func VFSqlQ(goalContext *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	sqlQuery, ok := x.BV().(goal.S)
	switch len(args) {
	case 1:
		if !ok {
			return panicType("sql.q s", "s", x)
		}
		fmt.Printf("QUERY %q\n", string(sqlQuery))
		queryResult, err := modeSqlRunQuery(&ctx, string(sqlQuery), nil)
		if err != nil {
			return goal.Errorf("%v", err)
		}
		return queryResult
	// TODO Figure out how to accept a universal array as the right argument.
	case 2:
		if !ok {
			return panicType("squery sql.q sparams", "squery", x)
		}
		y := args[0]
		yv := y.BV()
		placeholderArgs := make([]interface{}, 1)
		fmt.Printf("args %q\n", args)
		switch yv := yv.(type) {
		case *goal.AB:
			for i, x := range yv.Slice {
				placeholderArgs[i] = x
			}
		case *goal.AI:
			for i, x := range yv.Slice {
				placeholderArgs[i] = x
			}
		case *goal.AV:
			for i, x := range yv.Slice {
				if x.IsI() {
					placeholderArgs[i] = x.I()
				}
			}
		case *goal.AS:
			for i, x := range yv.Slice {
				placeholderArgs[i] = x
			}
		}
		if !ok {
			return panicType("squery sql.q sparams", "sparams", y)
		}
		queryResult, err := modeSqlRunQuery(&ctx, string(sqlQuery), placeholderArgs)
		if err != nil {
			return goal.Errorf("%v", err)
		}
		return queryResult
	default:
		return goal.Panicf("sql.q : too many arguments (%d), expects 1 or 2 arguments", len(args))
	}
}

// HTTP via go-resty

// TODO Research: What would a `reflect` Goal function/lib look like, to grab things like public struct fields on the Goal side?
//
//	https://go.dev/blog/laws-of-reflection
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
func (httpClient *HttpClient) Append(ctx *goal.Context, dst []byte, compact bool) []byte {
	// Go prints nil as `<nil>` so following suit.
	return append(dst, fmt.Sprintf("<ari.HttpClient %#v>", httpClient.client)...)
}

func newHttpClient(optionsD *goal.D) (*HttpClient, error) {
	// TODO Support options that resty.Client supports
	// [DONE] BaseURL               string
	// [DONE] QueryParam            url.Values //  type Values map[string][]string
	// [DONE] FormData              url.Values
	// [DONE] PathParams            map[string]string
	// [DONE] RawPathParams         map[string]string
	// [DONE] Header                http.Header // Use Add methods; accept dictionary of either single strings or []string
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

	// // HeaderAuthorizationKey is used to set/access Request Authorization header
	// // value when `SetAuthToken` option is used.
	// HeaderAuthorizationKey string
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
	// TODO Support dictionary argument with data that is transformed into options supported by resty.Client
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
		// TODO Should probably not include prefix
		return goal.NewPanic("http.client : too many arguments")
	}
}

func VFHttpGet(goalContext *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	sqlQuery, ok := x.BV().(goal.S)
	switch len(args) {
	case 1:
		if !ok {
			return panicType("http.get s", "s", x)
		}
		hc, err := newHttpClient(&goal.D{})
		if err != nil {
			return goal.NewPanicError(err)
		}
		resp, err := hc.client.R().Get(string(sqlQuery))
		if err != nil {
			fmt.Fprintf(os.Stderr, "HTTP error: %v\n", err)
			// Continue
			// error response built below, with
			// "ok" as false
		}
		// Construct goal.V values for return dict
		statusS := goal.NewS(resp.Status())
		headers := resp.Header()
		headerKeysSlice := make([]string, 0) // TODO Figure out why len(resp.Header()) wasn't correct
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
		return goal.NewD(ks, vs)
	case 2:
		if !ok {
			return panicType("httpclient http.get url", "squery", x)
		}
		panic("unimplemented")
		y := args[0]
		yv := y.BV()
		placeholderArgs := make([]interface{}, 1)
		fmt.Printf("args %q\n", args)
		switch yv := yv.(type) {
		case *goal.AB:
			for i, x := range yv.Slice {
				placeholderArgs[i] = x
			}
		case *goal.AI:
			for i, x := range yv.Slice {
				placeholderArgs[i] = x
			}
		case *goal.AV:
			for i, x := range yv.Slice {
				if x.IsI() {
					placeholderArgs[i] = x.I()
				}
			}
		case *goal.AS:
			for i, x := range yv.Slice {
				placeholderArgs[i] = x
			}
		}
		if !ok {
			return panicType("squery sql.q sparams", "sparams", y)
		}
		queryResult, err := modeSqlRunQuery(&ctx, string(sqlQuery), placeholderArgs)
		if err != nil {
			return goal.Errorf("%v", err)
		}
		return queryResult
	default:
		return goal.Panicf("sql.q : too many arguments (%d), expects 1 or 2 arguments", len(args))
	}
}

// When the REPL mode is switched to Goal, this resets the proper defaults.
func modeGoalSetReplDefaults(ctx *Context) {
	ctx.evalMode = modeGoalEval
	ctx.editor.Prompt = modeGoalPrompt
	ctx.editor.NextPrompt = modeGoalNextPrompt
	ctx.editor.AutoComplete = modeGoalAutocompleteFn(ctx.goalContext)
	ctx.editor.CheckInputComplete = nil // TODO Configurable
	ctx.editor.SetExternalEditorEnabled(true, "goal")
}

// When the REPL mode is switched to SQL, this resets the proper defaults. Separate modes for read-only and read/write SQL evaluation.
func modeSqlSetReplDefaults(ctx *Context, evalMode evalMode) {
	ctx.evalMode = evalMode
	ctx.editor.CheckInputComplete = modeSqlCheckInputComplete
	ctx.editor.AutoComplete = modeSqlAutocomplete
	ctx.editor.SetExternalEditorEnabled(true, "sql")
	switch ctx.evalMode {
	case modeSqlEvalReadOnly:
		ctx.editor.Prompt = modeSqlReadOnlyPrompt
		ctx.editor.NextPrompt = modeSqlReadOnlyNextPrompt
	case modeSqlEvalReadWrite:
		ctx.editor.Prompt = modeSqlReadWritePrompt
		ctx.editor.NextPrompt = modeSqlReadWriteNextPrompt
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

func modeGoalAutocompleteFn(ctx *goal.Context) func(v [][]rune, line, col int) (msg string, completions editline.Completions) {
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
			goalGlobals := ctx.GlobalNames(nil)
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
			goalKeywords := goalKeywords(ctx)
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

func goalRegisterVariadics(ctx *goal.Context) {
	// From Goal itself
	gos.Import(ctx, "")
	// Ari
	ctx.RegisterDyad("http.client", VFHttpClient)
	ctx.RegisterDyad("http.get", VFHttpGet)
	ctx.RegisterDyad("sql.q", VFSqlQ)
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ari.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".ari" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".ari")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
