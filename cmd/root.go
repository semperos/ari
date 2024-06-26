package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

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
		cliEditor:     cliEditor,
		cliMode:       cliModeGoal,
		autoCompleter: autoCompleter,
		ariContext:    ariContext,
	}

	mainCliSystem.switchModeToGoal()

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
			fmt.Fprintf(os.Stderr, "Failed to write REPL history, error: %v\n", err)
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

	// NB: MUST be last in this method.
	err = rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
