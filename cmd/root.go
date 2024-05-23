package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
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
	modeSqlEval
)

// TODO Support Prolog https://github.com/ichiban/prolog?tab=readme-ov-file
const (
	modeGoalPrompt     = "goal> "
	modeGoalNextPrompt = "    > "
	modeSqlPrompt      = "sql> "
	modeSqlNextPrompt  = "   > "
)

type Context struct {
	editor      *bubbline.Editor
	evalMode    evalMode
	goalContext *goal.Context
	sqlDb       *sql.DB
	sqlDbConn   *sql.Conn
}

func replMain() {
	// Goal
	goalCtx := goal.NewContext()
	goalCtx.Log = os.Stderr
	goalRegisterVariadics(goalCtx)

	// REPL interface
	editor := bubbline.New()
	editor.Placeholder = ""
	editor.Reflow = func(x bool, y string, _ int) (bool, string, string) {
		return editline.DefaultReflow(x, y, 80)
	}

	// SQL is initialized the first time )sql is invoked.
	ctx := Context{editor: editor, sqlDb: nil, sqlDbConn: nil, goalContext: goalCtx}

	modeGoalSetReplDefaults(&ctx)

	// TODO History file from configuration and configure history.
	// TODO Separate history for each language mode (maybe a good idea?)
	historyFile := ".ari.history"
	if err := editor.LoadHistory(historyFile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load history, error: %v\n", err)
	}
	editor.SetAutoSaveHistory(historyFile, true)
	editor.SetDebugEnabled(true)
	editor.SetExternalEditorEnabled(true, "goal")

	// Read-print loop starts here.
	for {
		line, err := editor.GetLine()
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
		err = editor.AddHistory(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write REPL history, error: %v\n", err)
		}

		// Support system commands
		if matchesSystemCommand(line) {
			cmdAndArgs := strings.Split(line, " ")
			systemCommand := cmdAndArgs[0][1:] // remove leading )
			switch systemCommand {
			case "goal":
				modeSqlClose(&ctx) // NB: Include in every non-"sql" case
				modeGoalSetReplDefaults(&ctx)
			case "sql":
				modeSqlInitialize(&ctx)
				modeSqlSetReplDefaults(&ctx)
			default:
				fmt.Fprintf(os.Stderr, "Unsupported system command '%v'\n", systemCommand)
			}
			continue
		}

		// Future: Support user commands with ]

		switch ctx.evalMode {
		case modeGoalEval:
			value, err := goalCtx.Eval(line)
			if err != nil {
				fmt.Fprint(os.Stderr, err)
			}
			fmt.Fprintln(os.Stdout, value.Sprint(goalCtx, false))
		case modeSqlEval:
			rows, err := ctx.sqlDb.QueryContext(context.Background(), line)
			if err != nil {
				// fmt.Fprintf(os.Stderr, "Failed to run SQL query %q\nDatabase Error:\n%s\n", line, err)
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			defer rows.Close()
			colNames, err := rows.Columns()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get SQL columns for query %q\nDatabase Error:\n%s", line, err)
				continue
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
					fmt.Fprintf(os.Stderr, "Failed to scan SQL row fro query %q\nDatabase Error:\n%s", line, err)
					break
				}
				for i, col := range cols {
					// fmt.Printf("SQL %v // Go %v\n", colTypes[i].DatabaseTypeName(), reflect.TypeOf(col))
					switch col := col.(type) {
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
						rowValues[i] = append(rowValues[i], goal.NewS(fmt.Sprintf("%v", col)))
					}
				}
				// values = append(values, goal.NewAV(vs))
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
			// Last result table as sql.t in Goal:
			ctx.goalContext.AssignGlobal("sql.t", goalD)
			fmt.Printf("%v\n", goalD)
		}
	}
}

// Caller is expected to close both ctx.sqlDb and ctx.sqlDbConn
func modeSqlInitialize(ctx *Context) {
	exampleDbFile := "/Users/dlg/dev/julia/work-product-data/work_2023.duckdb?access_mode=read_only"
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

func modeGoalSetReplDefaults(ctx *Context) {
	ctx.evalMode = modeGoalEval
	ctx.editor.Prompt = modeGoalPrompt
	ctx.editor.NextPrompt = modeGoalNextPrompt
	ctx.editor.AutoComplete = modeGoalAutocompleteFn(ctx.goalContext)
	ctx.editor.CheckInputComplete = nil // TODO Configurable
	ctx.editor.SetExternalEditorEnabled(true, "goal")
}

func modeSqlSetReplDefaults(ctx *Context) {
	ctx.evalMode = modeSqlEval
	ctx.editor.CheckInputComplete = modeSqlCheckInputComplete
	ctx.editor.Prompt = modeSqlPrompt
	ctx.editor.NextPrompt = modeSqlNextPrompt
	ctx.editor.AutoComplete = modeSqlAutocomplete
	ctx.editor.SetExternalEditorEnabled(true, "sql")
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
	goalSymbolRe := regexp.MustCompile("[a-zA-Z]+")
	return func(v [][]rune, line, col int) (msg string, completions editline.Completions) {
		origWord, _, _ := computil.FindWord(v, line, col)
		s := goalSymbolRe.FindStringIndex(origWord)
		word := origWord[s[0]:s[1]]
		wstart := s[0] // Preserve non-word prefix
		wend := s[1]   // Preserve non-word suffix
		msg = fmt.Sprintf("Matching %v", word)
		candidatesPerCategory := map[string][]string{}
		lword := strings.ToLower(word)
		// N.B. Matching system commands must come first.
		acSystemCommandCandidates(lword, candidatesPerCategory)
		goalGlobals := ctx.GlobalNames(nil)
		category := "Global"
		for _, goalGlobal := range goalGlobals {
			if strings.HasPrefix(strings.ToLower(goalGlobal), lword) {
				candidatesPerCategory[category] = append(candidatesPerCategory[category], goalGlobal)
			}
		}
		goalKeywords := goalKeywords(ctx)
		category = "Keyword"
		for _, goalKeyword := range goalKeywords {
			if strings.HasPrefix(strings.ToLower(goalKeyword), lword) {
				candidatesPerCategory[category] = append(candidatesPerCategory[category], goalKeyword)
			}
		}
		category = "Syntax"
		for name, chstr := range goalNonAsciis {
			if strings.HasPrefix(strings.ToLower(name), lword) {
				candidatesPerCategory[category] = append(candidatesPerCategory[category], chstr)
			}
		}
		// msg = fmt.Sprintf("Type is %v")
		completions = &multiComplete{
			Values:     complete.MapValues(candidatesPerCategory, nil),
			moveRight:  wend - col,
			deleteLeft: wend - wstart,
		}
		return msg, completions
	}
}

func modeSqlAutocomplete(v [][]rune, line, col int) (msg string, completions editline.Completions) {
	word, wstart, wend := computil.FindWord(v, line, col)
	// msg = fmt.Sprintf("Matching '%v'", word)
	candidatesPerCategory := map[string][]string{}
	lword := strings.ToLower(word)
	// N.B. Matching system commands must come first.
	acSystemCommandCandidates(lword, candidatesPerCategory)
	for _, sqlWord := range acSqlKeywords {
		if strings.HasPrefix(strings.ToLower(sqlWord), lword) {
			candidatesPerCategory["sql"] = append(candidatesPerCategory["sql"], sqlWord)
		}
	}
	completions = &multiComplete{
		Values:     complete.MapValues(candidatesPerCategory, nil),
		moveRight:  wend - col,
		deleteLeft: wend - wstart,
	}
	return msg, completions
}

func acSystemCommandCandidates(lword string, candidatesPerCategory map[string][]string) {
	if matchesSystemCommand(lword) {
		// Consider: Could be handled by storing )goal, )sql as the mode values
		for _, mode := range acModeSystemCommands {
			if len(lword) == 0 || strings.HasPrefix(strings.ToLower(mode), lword) {
				candidatesPerCategory["mode"] = append(candidatesPerCategory["mode"], mode)
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

// See goal.go, sql.go for other autocomplete sources
var acModeSystemCommands = []string{
	")goal", ")sql",
}

func goalRegisterVariadics(ctx *goal.Context) {
	// NB: Keep up-to-date with acGoalBuiltins
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
