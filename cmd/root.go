package cmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"codeberg.org/anaseto/goal"
	"github.com/knz/bubbline"
	"github.com/knz/bubbline/complete"
	"github.com/knz/bubbline/computil"
	"github.com/knz/bubbline/editline"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/semperos/ari"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "ari",
	Short: "ari - Array relational interactive environment",
	Long: `ari is an interactive environment for array + relational programming.

It embeds the Goal array programming language, with extensions for
working with SQL and HTTP APIs.`,
	Run: func(_ *cobra.Command, _ []string) {
		replMain()
	},
}

type cliEvalMode int

const (
	cliModeGoalEval = iota
	cliModeSQLEvalReadOnly
	cliModeSQLEvalReadWrite
)

const (
	cliModeGoalPrompt             = "goal> "
	cliModeGoalNextPrompt         = "    > "
	cliModeSQLReadOnlyPrompt      = "sql> "
	cliModeSQLReadOnlyNextPrompt  = "   > "
	cliModeSQLReadWritePrompt     = "sql!> "
	cliModeSQLReadWriteNextPrompt = "    > "
)

type CliContext struct {
	cliEditor   *bubbline.Editor
	cliEvalMode cliEvalMode
	ariContext  *ari.Context
}

func replMain() {
	mainCliContext := CliContext{}
	cliEditorInitialize(&mainCliContext)
	err := ari.ContextInitGoal(&ari.GlobalContext)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	mainCliContext.ariContext = &ari.GlobalContext
	cliModeGoalSetReplDefaults(&mainCliContext)

	// REPL starts here
	for {
		line, err := mainCliContext.cliEditor.GetLine()
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
		err = mainCliContext.cliEditor.AddHistory(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write REPL history, error: %v\n", err)
		}

		// Support system commands
		if matchesSystemCommand(line) {
			cmdAndArgs := strings.Split(line, " ")
			systemCommand := cmdAndArgs[0]
			switch systemCommand {
			case ")goal":
				cliModeSQLClose(&mainCliContext) // NB: Include in every non-"sql" case
				cliModeGoalSetReplDefaults(&mainCliContext)
			case ")sql": // TODO Accept data source name as argument
				var mode cliEvalMode
				mode = cliModeSQLEvalReadOnly
				err := cliModeSQLInitialize(&mainCliContext, mode)
				if err != nil {
					log.Fatalf("%v", err)
				}
				cliModeSQLSetReplDefaults(&mainCliContext, mode)
			case ")sql!": // TODO Accept data source name as argument
				var mode cliEvalMode
				mode = cliModeSQLEvalReadWrite
				err := cliModeSQLInitialize(&mainCliContext, mode)
				if err != nil {
					log.Fatalf("%v", err)
				}
				cliModeSQLSetReplDefaults(&mainCliContext, mode)
			default:
				fmt.Fprintf(os.Stderr, "Unsupported system command '%v'\n", systemCommand)
			}
			continue
		}

		// Future: Consider user commands with ]

		goalContext := mainCliContext.ariContext.GoalContext

		switch mainCliContext.cliEvalMode {
		case cliModeGoalEval:
			value, err := goalContext.Eval(line)
			if err != nil {
				fmt.Fprint(os.Stderr, err)
			}
			// Suppress printing values for variable assignments
			if !mainCliContext.ariContext.GoalContext.AssignedLast() {
				fmt.Fprintln(os.Stdout, value.Sprint(goalContext, false))
			}
		case cliModeSQLEvalReadOnly:
			_, err := cliModeSQLRunQuery(&mainCliContext, line, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to run SQL query %q\nDatabase Error:%s\n", line, err)
			} else {
				_, err := goalContext.Eval(`fmt.tbl[sql.t;*#'sql.t;#sql.t;"%.1f"]`)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to print SQL query results via Goal evaluation: %v\n", err)
				}
			}
		case cliModeSQLEvalReadWrite:
			_, err := cliModeSQLRunExec(&mainCliContext, line, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to run SQL query %q\nDatabase Error:%s\n", line, err)
			} else {
				_, err := goalContext.Eval(`fmt.tbl[sql.t;*#'sql.t;#sql.t;"%.1f"]`)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to print SQL exec results via Goal evaluation: %v\n", err)
				}
			}
		}
	}
}

const cliDefaultReflowWidth = 80

func cliEditorInitialize(cliContext *CliContext) {
	editor := bubbline.New()
	editor.Placeholder = ""
	editor.Reflow = func(x bool, y string, _ int) (bool, string, string) {
		return editline.DefaultReflow(x, y, cliDefaultReflowWidth)
	}
	historyFile := viper.GetString("history")
	if err := editor.LoadHistory(historyFile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load history, error: %v\n", err)
	}
	editor.SetAutoSaveHistory(historyFile, true)
	editor.SetDebugEnabled(true)
	editor.SetExternalEditorEnabled(true, "goal")
	cliContext.cliEditor = editor
}

// Caller is expected to close everything.
func cliModeSQLInitialize(_ *CliContext, evalMode cliEvalMode) error {
	dataSourceName := viper.GetString("database")
	if evalMode == cliModeSQLEvalReadOnly && len(dataSourceName) != 0 {
		// In-memory doesn't support read_only access
		dataSourceName += "?access_mode=read_only"
	}
	err := ari.ContextInitSQL(&ari.GlobalContext, dataSourceName)
	if err != nil {
		return err
	}
	return nil
}

func cliModeSQLClose(cliContext *CliContext) {
	sqlDatabase := cliContext.ariContext.SQLDatabase
	if sqlDatabase != nil {
		sqlDatabase.DB.Close()
		sqlDatabase.IsOpen = false
	}
}

func cliModeSQLRunQuery(cliContext *CliContext, sqlQuery string, args []any) (goal.V, error) {
	sqlDatabase := cliContext.ariContext.SQLDatabase
	if sqlDatabase == nil || !sqlDatabase.IsOpen {
		err := cliModeSQLInitialize(cliContext, cliModeSQLEvalReadOnly)
		if err != nil {
			return goal.V{}, err
		}
	}
	goalD, err := ari.SQLQueryContext(sqlDatabase, sqlQuery, args)
	if err != nil {
		return goal.V{}, err
	}
	// Last result table as sql.t in Goal, to support switching eval mdoes:
	cliContext.ariContext.GoalContext.AssignGlobal("sql.t", goalD)
	return goalD, nil
}

func cliModeSQLRunExec(cliContext *CliContext, sqlQuery string, args []any) (goal.V, error) {
	sqlDatabase := cliContext.ariContext.SQLDatabase
	if sqlDatabase == nil || !sqlDatabase.IsOpen {
		err := cliModeSQLInitialize(cliContext, cliModeSQLEvalReadOnly)
		if err != nil {
			return goal.V{}, err
		}
	}
	goalD, err := ari.SQLExec(sqlDatabase, sqlQuery, args)
	if err != nil {
		return goal.V{}, err
	}
	cliContext.ariContext.GoalContext.AssignGlobal("sql.t", goalD)
	return goalD, nil
}

// When the REPL mode is switched to Goal, this resets the proper defaults.
func cliModeGoalSetReplDefaults(cliContext *CliContext) {
	cliContext.cliEvalMode = cliModeGoalEval
	cliContext.cliEditor.Prompt = cliModeGoalPrompt
	cliContext.cliEditor.NextPrompt = cliModeGoalNextPrompt
	cliContext.cliEditor.AutoComplete = modeGoalAutocompleteFn(cliContext.ariContext.GoalContext)
	cliContext.cliEditor.CheckInputComplete = nil
	cliContext.cliEditor.SetExternalEditorEnabled(true, "goal")
}

// When the REPL mode is switched to SQL, this resets the proper defaults.
func cliModeSQLSetReplDefaults(cliContext *CliContext, evalMode cliEvalMode) {
	cliContext.cliEvalMode = evalMode
	cliContext.cliEditor.CheckInputComplete = modeSQLCheckInputComplete
	cliContext.cliEditor.AutoComplete = modeSQLAutocomplete
	cliContext.cliEditor.SetExternalEditorEnabled(true, "sql")
	switch cliContext.cliEvalMode {
	case cliModeSQLEvalReadOnly:
		cliContext.cliEditor.Prompt = cliModeSQLReadOnlyPrompt
		cliContext.cliEditor.NextPrompt = cliModeSQLReadOnlyNextPrompt
	case cliModeSQLEvalReadWrite:
		cliContext.cliEditor.Prompt = cliModeSQLReadWritePrompt
		cliContext.cliEditor.NextPrompt = cliModeSQLReadWriteNextPrompt
	}
}

func matchesSystemCommand(s string) bool {
	return strings.HasPrefix(s, ")")
}

func modeSQLCheckInputComplete(v [][]rune, line, _ int) bool {
	if len(v) == 1 && matchesSystemCommand(string(v[0])) {
		return true
	}
	if line == len(v)-1 && // Enter on last row.
		strings.HasSuffix(string(v[len(v)-1]), ";") { // Semicolon at end of last row.;
		return true
	}
	return false
}

func modeGoalAutocompleteFn(goalContext *goal.Context) func(v [][]rune, line, col int) (string, editline.Completions) {
	goalNameRe := regexp.MustCompile(`[a-zA-Z\.]+`)
	return func(v [][]rune, line, col int) (string, editline.Completions) {
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
					if val, ok := ari.GoalGlobalsHelp[goalGlobal]; ok {
						help = val
					} else {
						help = "A Goal global binding"
					}
					candidatesPerCategory[category] = append(candidatesPerCategory[category], acEntry{goalGlobal, help})
				}
			}
			goalKeywords := ari.GoalKeywords(goalContext)
			category = "Keyword"
			for _, goalKeyword := range goalKeywords {
				if strings.HasPrefix(strings.ToLower(goalKeyword), lword) {
					var help string
					if val, ok := ari.GoalKeywordsHelp[goalKeyword]; ok {
						help = val
					} else {
						help = "A Goal keyword"
					}
					candidatesPerCategory[category] = append(candidatesPerCategory[category], acEntry{goalKeyword, help})
				}
			}
			category = "Syntax"
			syntaxSet := make(map[string]bool, 0)
			for name, chstr := range ari.GoalSyntax {
				if strings.HasPrefix(strings.ToLower(name), lword) {
					if _, ok := syntaxSet[chstr]; !ok {
						syntaxSet[chstr] = true
						var help string
						if val, ok := ari.GoalSyntaxHelp[chstr]; ok {
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
		completions := &multiComplete{
			Values:     complete.MapValues(candidatesPerCategory, nil),
			moveRight:  end - col,
			deleteLeft: end - start,
		}
		msg := ""
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

func modeSQLAutocomplete(v [][]rune, line, col int) (string, editline.Completions) {
	word, wstart, wend := computil.FindWord(v, line, col)
	// msg = fmt.Sprintf("Matching '%v'", word)
	candidatesPerCategory := map[string][]acEntry{}
	lword := strings.ToLower(word)
	// N.B. Matching system commands must come first.
	acSystemCommandCandidates(lword, candidatesPerCategory)
	for _, sqlWord := range acSQLKeywords {
		if strings.HasPrefix(strings.ToLower(sqlWord), lword) {
			candidatesPerCategory["sql"] = append(candidatesPerCategory["sql"], acEntry{sqlWord, "SQL help"})
		}
	}
	completions := &multiComplete{
		Values:     complete.MapValues(candidatesPerCategory, nil),
		moveRight:  wend - col,
		deleteLeft: wend - wstart,
	}
	msg := ""
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
	keys := make([]string, 0, len(acModeSystemCommands))
	for k := range acModeSystemCommands {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}()

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
