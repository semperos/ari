package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"
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
	cliModeGoal cliMode = iota
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
	cliSystem.detectAriPrompt()
	cliSystem.cliEditor.AutoComplete = cliSystem.autoCompleter.goalAutoCompleteFn()
	cliSystem.cliEditor.CheckInputComplete = modeGoalCheckInputComplete
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

func cliModeFromString(s string) (cliMode, error) {
	switch s {
	case "goal":
		return cliModeGoal, nil
	case "sql":
		return cliModeSQLReadOnly, nil
	case "sql!":
		return cliModeSQLReadWrite, nil
	default:
		return 0, errors.New("unsupported ari mode: " + s)
	}
}

// TODO Make this more testable by pulling out things from Cobra/Viper into a config struct
func ariMain(cmd *cobra.Command, args []string) int {
	dataSourceName := viper.GetString("database")
	ariContext, err := ari.NewContext(dataSourceName)
	cobra.CheckErr(err)
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

	// MUST COME FIRST
	// Engage detailed print on panic.
	debug, err := cmd.Flags().GetBool("debug")
	cobra.CheckErr(err)
	if debug {
		defer debugPrintStack(ariContext.GoalContext, mainCliSystem.programName)
	}

	cpuProfile, err := cmd.Flags().GetBool("cpu-profile")
	cobra.CheckErr(err)
	if cpuProfile {
		//nolint:govet // false positive?
		cpuProfileFile, err := os.Create(fmt.Sprintf("cpu-profile-%d", time.Now().UnixMilli()))
		cobra.CheckErr(err)
		err = pprof.StartCPUProfile(cpuProfileFile)
		cobra.CheckErr(err)
		defer pprof.StopCPUProfile()
	}

	// MUST PRECEDE EXECUTE/REPL
	goalFilesToLoad := viper.GetStringSlice("load")
	for _, f := range goalFilesToLoad {
		err = runScript(&mainCliSystem, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load file %q with error: %v", f, err)
			return 1
		}
	}

	hasFileArgument := len(args) > 0

	// Eval and exit
	programToExecute, err := cmd.Flags().GetString("execute")
	cobra.CheckErr(err)
	if programToExecute != "" {
		err = runCommand(&mainCliSystem, programToExecute)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to execute program:\n%q\n    with error:\n%v\n", programToExecute, err)
			return 1
		}
		// Support -e/--execute along with a file argument.
		if !hasFileArgument {
			return 0
		}
	}

	if hasFileArgument {
		f := args[0]
		err = runScript(&mainCliSystem, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to run file %q with error: %v", f, err)
			return 1
		}
		return 0
	}

	// With files loaded (which might adjust the prompt via Goal code)
	// and knowing we're not executing and exiting immediately,
	// set up the CLI REPL.
	startupCliModeString := viper.GetString("mode")
	startupCliMode, err := cliModeFromString(startupCliModeString)
	cobra.CheckErr(err)
	mainCliSystem.switchMode(startupCliMode)

	// REPL
	readEvalPrintLoop(mainCliSystem)
	return 0
}

func readEvalPrintLoop(mainCliSystem CliSystem) {
	cliEditor := mainCliSystem.cliEditor
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

		// Add line to REPL history, even if not a legal expression (thus before we try to evaluate)
		err = cliEditor.AddHistory(line)
		if err != nil {
			// NB: Not exiting if history file fails to load, just printing.
			fmt.Fprintf(os.Stderr, "Failed to write REPL history with error: %v\n", err)
		}

		// Future: Consider user commands with ]
		if matchesSystemCommand(line) {
			mainCliSystem.replEvalSystemCommand(line)
			continue
		}

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
		// NB: Goal errors built with the Goal `error` function are goal.V values,
		// whereas Goal returns a Go error for things like an undefined global,
		// so "Goal error[]":"Java Exception"::"Go error":"Java Error" here.
		// This means that the user's ari.prompt function is not honored in this
		// case, which is good by design in my opinion given the low-level nature
		// of these errors.
		fmt.Fprintln(os.Stderr, err)
	}

	if !goalContext.AssignedLast() {
		ariPrintFn := cliSystem.detectAriPrint()
		if ariPrintFn != nil {
			ariPrintFn(value)
		} else {
			fmt.Fprintln(os.Stdout, value.Sprint(goalContext, false))
		}
		// In the REPL, make it easy to get the value of the _p_revious expression
		// just evaluated. Equivalent of *1 in Lisp REPLs. Skip assignments.
		goalContext.AssignGlobal("ari.p", value)
	}

	cliSystem.detectAriPrompt()
}

// detectAriPrompt interrogates Goal globals ari.prompt and ari.nextprompt
// to determine the prompt shown at the CLI REPL.
func (cliSystem *CliSystem) detectAriPrompt() {
	goalContext := cliSystem.ariContext.GoalContext

	prompt, found := goalContext.GetGlobal("ari.prompt")
	if found {
		promptS, ok := prompt.BV().(goal.S)
		if ok {
			cliSystem.cliEditor.Prompt = string(promptS)
		} else {
			fmt.Fprintf(os.Stderr, "ari.prompt must be a string, but found %q\n", prompt)
		}
	}

	nextPrompt, found := goalContext.GetGlobal("ari.nextprompt")
	if found {
		nextPromptS, ok := nextPrompt.BV().(goal.S)
		if ok {
			cliSystem.cliEditor.NextPrompt = string(nextPromptS)
		} else {
			fmt.Fprintf(os.Stderr, "ari.nextprompt must be a string, but found %q\n", nextPrompt)
		}
	}
}

// detectAriPrint returns a function for printing values at the REPL in goal mode.
func (cliSystem *CliSystem) detectAriPrint() func(goal.V) {
	goalContext := cliSystem.ariContext.GoalContext
	printFn, found := goalContext.GetGlobal("ari.print")
	if found {
		if printFn.IsCallable() {
			return func(v goal.V) {
				printFn.ApplyAt(goalContext, v)
			}
		}
		fmt.Fprintf(os.Stderr, "Error: The ari.print value must be a callable, but encountered %q", printFn)
	}
	return nil
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

func (cliSystem *CliSystem) replEvalSystemCommand(line string) {
	cmdAndArgs := strings.Split(line, " ")
	systemCommand := cmdAndArgs[0]
	switch systemCommand {
	// IDEA )help that doesn't require quoting
	case ")goal":
		cliSystem.switchMode(cliModeGoal)
	case ")sql":
		cliSystem.switchMode(cliModeSQLReadOnly)
	case ")sql!":
		err := cliSystem.sqlInitialize(cliSystem.ariContext.SQLDatabase.DataSource, cliModeSQLReadWrite)
		if err != nil {
			log.Fatalf("%v", err)
		}
		cliSystem.switchMode(cliModeSQLReadWrite)
	default:
		fmt.Fprintf(os.Stderr, "Unsupported system command '%v'\n", systemCommand)
	}
}

// sqlInitialize opens a SQL database and expects the caller to close things.
func (cliSystem *CliSystem) sqlInitialize(dataSourceName string, cliMode cliMode) error {
	// In-memory doesn't support read_only access
	if cliMode == cliModeSQLReadOnly && len(dataSourceName) != 0 {
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
		if sqlDatabase.DB != nil {
			err := sqlDatabase.DB.Close()
			if err != nil {
				return err
			}
			sqlDatabase.IsOpen = false
		}
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
	// Last result table as sql.t in Goal, to support switching eval modes:
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

// debugPrintStack catches possible panics in debug mode, attempting to print debug
// information.
//
// Adapated from Goal's implementation.
func debugPrintStack(ctx *goal.Context, programName string) {
	if r := recover(); r != nil {
		printProgram(ctx, programName)
		log.Printf("Caught panic: %v\nStack Trace:\n", r)
		debug.PrintStack()
	}
}

// Adapted from Goal's implementation.
func runCommand(cliSystem *CliSystem, cmd string) error {
	return runSource(cliSystem, cmd, "")
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

// initConfigFn returns a function compatible with cobra.OnInitialize.
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

		viper.SetEnvPrefix("ARI")
		viper.AutomaticEnv()

		// Config file is optional, but if present and malformed, exit with error.
		err := viper.ReadInConfig()
		notFoundErr := viper.ConfigFileNotFoundError{}
		if err != nil && !errors.As(err, &notFoundErr) {
			cfgFileName := viper.ConfigFileUsed()
			fmt.Fprintf(os.Stderr, "[ERROR] Config file at '%v' could not be loaded due to error: %v\n", cfgFileName, err)
			os.Exit(1)
		}
	}
}

func main() {
	statusCode := 0
	rootCmd := &cobra.Command{
		Use:   "ari",
		Short: "ari - Array relational interactive environment",
		Long: `ari is an interactive environment for array + relational programming.

It embeds the Goal array programming language, with extensions for
working with SQL and HTTP APIs.`,
		// If 1 arg provided, treat as Goal source to run.
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Version and exit
			showVersion, err := cmd.Flags().GetBool("version")
			cobra.CheckErr(err)
			if showVersion {
				if bi, ok := debug.ReadBuildInfo(); ok {
					fmt.Fprintf(os.Stdout, "%v\n", bi)
				}
				os.Exit(0)
			}
			statusCode = ariMain(cmd, args)
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

	// Config file has processing in initConfigFn outside of viper lifecycle, so it's a separate variable.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", defaultCfgFile, "ari configuration")

	// Everything else should go through viper for consistency.
	pFlags := rootCmd.PersistentFlags()

	flagNameHistory := "history"
	flagNameDatabase := "database"

	pFlags.String(flagNameHistory, defaultHistFile, "history of REPL entries")
	pFlags.StringP(flagNameDatabase, "d", "", "DuckDB database (default: in-memory)")

	err = viper.BindPFlag(flagNameHistory, pFlags.Lookup(flagNameHistory))
	cobra.CheckErr(err)
	err = viper.BindPFlag(flagNameDatabase, pFlags.Lookup(flagNameDatabase))
	cobra.CheckErr(err)

	rootCmd.Flags().Bool("debug", false, "enable detailed debugging output on panic")
	rootCmd.Flags().Bool("cpu-profile", false, "write CPU profile to file")
	rootCmd.Flags().StringP("execute", "e", "", "string of Goal code to execute, last result not printed automatically")
	rootCmd.Flags().StringArrayP("load", "l", nil, "Goal source files to load on startup")
	err = viper.BindPFlag("load", rootCmd.Flags().Lookup("load"))
	cobra.CheckErr(err)
	rootCmd.Flags().StringP("mode", "m", "goal", "language mode at startup")
	err = viper.BindPFlag("mode", rootCmd.Flags().Lookup("mode"))
	cobra.CheckErr(err)
	rootCmd.Flags().BoolP("version", "v", false, "print version info and exit")

	// NB: MUST be last in this method.
	err = rootCmd.Execute()
	cobra.CheckErr(err)

	os.Exit(statusCode)
}
