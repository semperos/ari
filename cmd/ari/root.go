package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"unsafe"

	"codeberg.org/anaseto/goal"
	"github.com/knz/bubbline"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/semperos/ari"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands.
// var

type cliMode int

const (
	cliModeGoal = iota
	cliModeSQLReadOnly
	cliModeSQLReadWrite
)

const (
	cliModeGoalPrompt             = "goal> "
	cliModeGoalNextPrompt         = "    > "
	cliModeSQLReadOnlyPrompt      = "sql> "
	cliModeSQLReadOnlyNextPrompt  = "   > "
	cliModeSQLReadWritePrompt     = "sql!> "
	cliModeSQLReadWriteNextPrompt = "    > "
)

type CliSystem struct {
	cliEditor     *bubbline.Editor
	cliMode       cliMode
	autoCompleter *AutoCompleter
	ariContext    *ari.Context
	debug         bool
	programName   string
}

func (cliSystem *CliSystem) switchModePre(cliMode cliMode) {
	if cliMode != cliModeSQLReadOnly && cliMode != cliModeSQLReadWrite {
		cliSystem.sqlClose()
	}
}

func (cliSystem *CliSystem) switchMode(cliMode cliMode) {
	switch cliMode {
	case cliModeGoal:
		cliSystem.switchModeToGoal()
	case cliModeSQLReadOnly:
		cliSystem.switchModeToSQLReadOnly()
	case cliModeSQLReadWrite:
		cliSystem.switchModeToSQLReadWrite()
	}
	cliSystem.switchModePre(cliMode)
}

func (cliSystem *CliSystem) switchModeToGoal() {
	cliSystem.cliMode = cliModeGoal
	cliSystem.cliEditor.Prompt = cliModeGoalPrompt
	cliSystem.cliEditor.NextPrompt = cliModeGoalNextPrompt
	cliSystem.cliEditor.AutoComplete = cliSystem.autoCompleter.goalAutoCompleteFn()
	// TODO This should be at parity with Goal's own REPL, see Goal's cmd.go
	cliSystem.cliEditor.CheckInputComplete = nil
	cliSystem.cliEditor.SetExternalEditorEnabled(true, "goal")
}

func (cliSystem *CliSystem) switchModeToSQLReadOnly() {
	err := cliSystem.ariContext.SQLDatabase.Open(cliSystem.ariContext.SQLDatabase.DataSource)
	if err != nil {
		log.Fatalf("%v", err)
	}
	cliSystem.cliMode = cliModeSQLReadOnly
	cliSystem.cliEditor.CheckInputComplete = modeSQLCheckInputComplete
	cliSystem.cliEditor.AutoComplete = cliSystem.autoCompleter.sqlAutoCompleteFn()
	cliSystem.cliEditor.SetExternalEditorEnabled(true, "sql")
	cliSystem.cliEditor.Prompt = cliModeSQLReadOnlyPrompt
	cliSystem.cliEditor.NextPrompt = cliModeSQLReadOnlyNextPrompt
}

func (cliSystem *CliSystem) switchModeToSQLReadWrite() {
	err := cliSystem.ariContext.SQLDatabase.Open(cliSystem.ariContext.SQLDatabase.DataSource)
	if err != nil {
		log.Fatalf("%v", err)
	}
	cliSystem.cliMode = cliModeSQLReadOnly
	cliSystem.cliEditor.CheckInputComplete = modeSQLCheckInputComplete
	cliSystem.cliEditor.AutoComplete = cliSystem.autoCompleter.sqlAutoCompleteFn()
	cliSystem.cliEditor.SetExternalEditorEnabled(true, "sql")
	cliSystem.cliEditor.Prompt = cliModeSQLReadWritePrompt
	cliSystem.cliEditor.NextPrompt = cliModeSQLReadWriteNextPrompt
}

func replMain() {
	dataSourceName := viper.GetString("database")
	ariContext, err := ari.NewContext(dataSourceName)
	if err != nil {
		log.Fatalf("%v", err)
	}
	cliEditor := cliEditorInitialize()
	autoCompleter := &AutoCompleter{ariContext: ariContext}
	mainCliSystem := CliSystem{
		ariContext:    ariContext,
		autoCompleter: autoCompleter,
		cliEditor:     cliEditor,
		cliMode:       cliModeGoal,
		debug:         viper.GetBool("debug"),
		programName:   os.Args[0],
	}

	mainCliSystem.switchModeToGoal()

	goalFilesToLoad := viper.GetStringSlice("load")
	for _, f := range goalFilesToLoad {
		err := runScript(&mainCliSystem, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load file %q with error: %v", f, err)
			os.Exit(1)
		}
	}

	// REPL starts here
	for {
		line, err := cliEditor.GetLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if errors.Is(err, bubbline.ErrInterrupted) {
				// Entered Ctrl+C to cancel input.
				fmt.Fprintln(os.Stdout, "^C")
			} else {
				fmt.Fprintln(os.Stderr, "error:", err)
			}
			continue
		}
		// Add to REPL history, even if not a legal expression (thus before we try to evaluate)
		err = cliEditor.AddHistory(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write REPL history with error: %v\n", err)
			// NB: Not exiting if history file fails to load, just printing.
		}

		// Support system commands
		if matchesSystemCommand(line) {
			mainCliSystem.replEvalSystemCommand(line, dataSourceName)
			continue
		}

		// Future: Consider user commands with ]

		switch mainCliSystem.cliMode {
		case cliModeGoal:
			mainCliSystem.replEvalGoal(line)
		case cliModeSQLReadOnly:
			mainCliSystem.replEvalSQLReadOnly(line)
		case cliModeSQLReadWrite:
			mainCliSystem.replEvalSQLReadWrite(line)
		}
	}
}

func (cliSystem *CliSystem) replEvalGoal(line string) {
	goalContext := cliSystem.ariContext.GoalContext
	value, err := goalContext.Eval(line)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
	}

	if !goalContext.AssignedLast() {
		fmt.Fprintln(os.Stdout, value.Sprint(goalContext, false))
	}
}

func (cliSystem *CliSystem) replEvalSQLReadOnly(line string) {
	_, err := cliSystem.sqlQuery(line, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run SQL query %q\nDatabase Error:%s\n", line, err)
	} else {
		_, err := cliSystem.ariContext.GoalContext.Eval(`fmt.tbl[sql.t;*#'sql.t;#sql.t;"%.1f"]`)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to print SQL query results via Goal evaluation: %v\n", err)
		}
	}
}

func (cliSystem *CliSystem) replEvalSQLReadWrite(line string) {
	_, err := cliSystem.sqlExec(line, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run SQL query %q\nDatabase Error:%s\n", line, err)
	} else {
		_, err := cliSystem.ariContext.GoalContext.Eval(`fmt.tbl[sql.t;*#'sql.t;#sql.t;"%.1f"]`)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to print SQL exec results via Goal evaluation: %v\n", err)
		}
	}
}

func (cliSystem *CliSystem) replEvalSystemCommand(line string, dataSourceName string) {
	cmdAndArgs := strings.Split(line, " ")
	systemCommand := cmdAndArgs[0]
	switch systemCommand {
	// TODO )help
	case ")goal":
		cliSystem.switchMode(cliModeGoal)
	case ")sql":
		cliSystem.switchMode(cliModeSQLReadOnly)
	case ")sql!":
		var mode cliMode
		mode = cliModeSQLReadWrite
		err := cliSystem.sqlInitialize(dataSourceName, mode)
		if err != nil {
			log.Fatalf("%v", err)
		}
		cliSystem.switchMode(cliModeSQLReadWrite)
	default:
		fmt.Fprintf(os.Stderr, "Unsupported system command '%v'\n", systemCommand)
	}
}

// Caller is expected to close everything.
func (cliSystem *CliSystem) sqlInitialize(dataSourceName string, cliMode cliMode) error {
	if cliMode == cliModeSQLReadOnly && len(dataSourceName) != 0 {
		// In-memory doesn't support read_only access
		dataSourceName += "?access_mode=read_only"
	}
	sqlDatabase, err := ari.NewSQLDatabase(dataSourceName)
	if err != nil {
		return err
	}
	cliSystem.ariContext.SQLDatabase = sqlDatabase
	return nil
}

func (cliSystem *CliSystem) sqlClose() error {
	sqlDatabase := cliSystem.ariContext.SQLDatabase
	if sqlDatabase != nil {
		err := sqlDatabase.DB.Close()
		if err != nil {
			return err
		}
		sqlDatabase.IsOpen = false
	}
	return nil
}

func (cliSystem *CliSystem) sqlQuery(sqlQuery string, args []any) (goal.V, error) {
	sqlDatabase := cliSystem.ariContext.SQLDatabase
	if sqlDatabase == nil || !sqlDatabase.IsOpen {
		err := cliSystem.sqlInitialize(viper.GetString("database"), cliModeSQLReadOnly)
		if err != nil {
			return goal.V{}, err
		}
	}
	goalD, err := ari.SQLQueryContext(sqlDatabase, sqlQuery, args)
	if err != nil {
		return goal.V{}, err
	}
	// Last result table as sql.t in Goal, to support switching eval mdoes:
	cliSystem.ariContext.GoalContext.AssignGlobal("sql.t", goalD)
	return goalD, nil
}

func (cliSystem *CliSystem) sqlExec(sqlQuery string, args []any) (goal.V, error) {
	sqlDatabase := cliSystem.ariContext.SQLDatabase
	if sqlDatabase == nil || !sqlDatabase.IsOpen {
		err := cliSystem.sqlInitialize(viper.GetString("database"), cliModeSQLReadOnly)
		if err != nil {
			return goal.V{}, err
		}
	}
	goalD, err := ari.SQLExec(sqlDatabase, sqlQuery, args)
	if err != nil {
		return goal.V{}, err
	}
	cliSystem.ariContext.GoalContext.AssignGlobal("sql.t", goalD)
	return goalD, nil
}

// Adapted from Goal's implementation.
func runScript(cliSystem *CliSystem, fname string) error {
	bs, err := os.ReadFile(fname)
	if err != nil {
		return fmt.Errorf("%s: %w", cliSystem.programName, err)
	}
	// We avoid redundant copy in bytes->string conversion.
	source := unsafe.String(unsafe.SliceData(bs), len(bs))
	return runSource(cliSystem, source, fname)
}

// Adapted from Goal's implementation.
func runSource(cliSystem *CliSystem, source, loc string) error {
	goalContext := cliSystem.ariContext.GoalContext
	err := goalContext.Compile(source, loc, "")
	if err != nil {
		if cliSystem.debug {
			printProgram(goalContext, cliSystem.programName)
		}
		return formatError(cliSystem.programName, err)
	}
	if cliSystem.debug {
		printProgram(goalContext, cliSystem.programName)
		return nil
	}
	r, err := goalContext.Run()
	// if m.interactive {
	// 	if err != nil {
	// 		formatREPLError(err, m.quiet)
	// 	}
	// 	if r.IsError() {
	// 		formatREPLError(errors.New(formatGoalError(goalContext, r)), m.quiet)
	// 	}
	// 	runStdin(goalContext, cfg, m)
	// 	return nil
	// }
	if err != nil {
		return formatError(cliSystem.programName, err)
	}
	if r.IsError() {
		return fmt.Errorf("%s", formatGoalError(goalContext, r))
	}
	return nil
}

// printProgram prints debug information about the context and any compiled
// source.
//
// Adapted from Goal's implementation.
func printProgram(ctx *goal.Context, programName string) {
	fmt.Fprintf(os.Stderr, "%s: debug info below:\n%v", programName, ctx.String())
}

// formatError formats an error from script or command.
//
// Adapted from Goal's implementation.
func formatError(programName string, err error) error {
	//nolint:errorlint // Need the type case e for the ErrorStack
	if e, ok := err.(*goal.Panic); ok {
		return fmt.Errorf("%s: %v", programName, e.ErrorStack())
	}
	return fmt.Errorf("%s: %w", programName, err)
}

// formatGoalError formats a Goal error value returned from the program.
func formatGoalError(ctx *goal.Context, r goal.V) string {
	if e, ok := r.BV().(*goal.Error); ok {
		return e.Msg(ctx)
	}
	return "(failed to format Goal error)"
}

// CLI (Cobra, Viper)

// Return a function for cobra.OnInitialize function.
func initConfigFn(cfgFile string) func() {
	return func() {
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
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "ari",
		Short: "ari - Array relational interactive environment",
		Long: `ari is an interactive environment for array + relational programming.

It embeds the Goal array programming language, with extensions for
working with SQL and HTTP APIs.`,
		Run: func(_ *cobra.Command, _ []string) {
			replMain()
		},
	}

	var cfgFile string
	cobra.OnInitialize(initConfigFn(cfgFile))

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	cfgDir := path.Join(home, ".config", "ari")

	defaultHistFile := path.Join(cfgDir, "ari-history.txt")
	defaultCfgFile := path.Join(cfgDir, "ari-config.yaml")

	// TODO Write default ari-config.yaml file if none present
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
	rootCmd.Flags().BoolP("debug", "d", false, "Enable debugging (detailed printing)")
	rootCmd.Flags().StringP("execute", "e", "", "String of Goal code to execute, last result not printed automatically")
	rootCmd.Flags().StringArrayP("load", "l", nil, "Goal source files to load on startup")
	err = viper.BindPFlag("load", rootCmd.Flags().Lookup("load"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to bind 'load' CLI flag: %v\n", err)
		os.Exit(1)
	}
	rootCmd.Flags().BoolP("version", "v", false, "Print version info and exit")

	// NB: MUST be last in this method.
	err = rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
