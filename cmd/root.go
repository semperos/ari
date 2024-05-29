package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"codeberg.org/anaseto/goal"
	gos "codeberg.org/anaseto/goal/os"
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
			fmt.Fprintln(os.Stdout, value.Sprint(ctx.goalContext, false))
		case modeSqlEvalReadOnly:
			goalDict, err := modeSqlRunQuery(&ctx, line, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to run SQL query %q\nDatabase Error:%s\n", line, err)
			}
			fmt.Printf("%v\n", goalDict)
		case modeSqlEvalReadWrite:
			goalDict, err := modeSqlRunQuery(&ctx, line, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to run SQL query %q\nDatabase Error:%s\n", line, err)
			}
			fmt.Printf("%v\n", goalDict)
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
	ks := goal.NewAS([]string{"lastInsertId", "rowsAffected"})
	vs := goal.NewAI([]int64{lastInsertId, rowsAffected})
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
	goalNameRe := regexp.MustCompile("[a-zA-Z]+")
	return func(v [][]rune, line, col int) (msg string, completions editline.Completions) {
		candidatesPerCategory := map[string][]acEntry{}
		word, start, end := computil.FindWord(v, line, col)
		// N.B. Matching system commands must come first.
		if matchesSystemCommand(word) {
			acSystemCommandCandidates(strings.ToLower(word), candidatesPerCategory)
		} else {
			s := goalNameRe.FindStringIndex(word)
			word := word[s[0]:s[1]]
			start = s[0] // Preserve non-word prefix
			end = s[1]   // Preserve non-word suffix
			msg = fmt.Sprintf("Matching %v", word)
			lword := strings.ToLower(word)
			goalGlobals := ctx.GlobalNames(nil)
			category := "Global"
			for _, goalGlobal := range goalGlobals {
				if strings.HasPrefix(strings.ToLower(goalGlobal), lword) {
					candidatesPerCategory[category] = append(candidatesPerCategory[category], acEntry{goalGlobal, "A Goal global"})
				}
			}
			goalKeywords := goalKeywords(ctx)
			category = "Keyword"
			for _, goalKeyword := range goalKeywords {
				if strings.HasPrefix(strings.ToLower(goalKeyword), lword) {
					candidatesPerCategory[category] = append(candidatesPerCategory[category], acEntry{goalKeyword, "A Goal keyword"})
				}
			}
			category = "Syntax"
			for name, chstr := range goalNonAsciis {
				if strings.HasPrefix(strings.ToLower(name), lword) {
					candidatesPerCategory[category] = append(candidatesPerCategory[category], acEntry{chstr, "Goal syntax"})
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
	return e.description
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
		keys := make([]string, len(acModeSystemCommands))
		for k := range acModeSystemCommands {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, mode := range keys {
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

func goalRegisterVariadics(ctx *goal.Context) {
	// From Goal itself
	ctx.RegisterMonad("chdir", gos.VFChdir)
	ctx.RegisterMonad("close", gos.VFClose)
	ctx.RegisterMonad("flush", gos.VFFlush)
	ctx.RegisterMonad("mkdir", gos.VFMkdir)
	ctx.RegisterMonad("remove", gos.VFRemove)
	ctx.RegisterMonad("stat", gos.VFStat)
	ctx.RegisterDyad("env", gos.VFEnv)
	ctx.RegisterDyad("import", gos.VFImport)
	ctx.RegisterDyad("open", gos.VFOpen)
	ctx.RegisterDyad("print", gos.VFPrint)
	ctx.RegisterDyad("read", gos.VFRead)
	ctx.RegisterDyad("rename", gos.VFRename)
	ctx.RegisterDyad("run", gos.VFRun)
	ctx.RegisterDyad("say", gos.VFSay)
	ctx.RegisterDyad("shell", gos.VFShell)
	ctx.AssignGlobal("STDOUT", gos.NewFileWriter(os.Stdout))
	ctx.AssignGlobal("STDERR", gos.NewFileWriter(os.Stderr))
	ctx.AssignGlobal("STDIN", gos.NewFileReader(os.Stdin))

	// Ari
	ctx.RegisterDyad("sql.q", VFSqlQ)
}

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
