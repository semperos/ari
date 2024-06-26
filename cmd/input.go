package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"codeberg.org/anaseto/goal"
	"github.com/knz/bubbline"
	"github.com/knz/bubbline/complete"
	"github.com/knz/bubbline/computil"
	"github.com/knz/bubbline/editline"
	"github.com/semperos/ari"
	"github.com/spf13/viper"
)

type AutoCompleter struct {
	ariContext         *ari.Context
	sqlKeywords        []string
	goalKeywordsHelp   map[string]string
	goalKeywordsKeys   []string
	goalSyntaxAliases  map[string]string
	goalSyntaxHelp     map[string]string
	goalSyntaxKeys     []string
	systemCommandsHelp map[string]string
	systemCommandsKeys []string
}

func matchesSystemCommand(s string) bool {
	return strings.HasPrefix(s, ")")
}

// CheckInputComplete handler for SQL.
//
// This detects whether a SQL statement is complete from the CLI editor's perspective,
// it does not provide autocomplete functionality.
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

// Autocompletion

//nolint:lll
func (autoCompleter *AutoCompleter) systemCommandsAutoComplete() func(v [][]rune, line, col int) (string, editline.Completions) {
	// Cache system commands, few though they be.
	if autoCompleter.systemCommandsKeys == nil {
		autoCompleter.cacheSystemCommands()
	}
	return func(v [][]rune, line, col int) (string, editline.Completions) {
		perCategory := map[string][]acEntry{}
		word, start, end := computil.FindWord(v, line, col)
		lword := strings.ToLower(word)
		if matchesSystemCommand(lword) {
			for _, mode := range autoCompleter.systemCommandsKeys {
				if len(lword) == 0 || strings.HasPrefix(strings.ToLower(mode), lword) {
					perCategory["mode"] = append(perCategory["mode"], acEntry{mode, autoCompleter.systemCommandsHelp[mode]})
				}
			}
		}

		completions := &multiComplete{
			Values:     complete.MapValues(perCategory, nil),
			moveRight:  end - col,
			deleteLeft: end - start,
		}
		msg := ""
		return msg, completions
	}
}

//nolint:lll
func (autoCompleter *AutoCompleter) goalAutoCompleteFn() func(v [][]rune, line, col int) (string, editline.Completions) {
	goalContext := autoCompleter.ariContext.GoalContext
	goalNameRe := regexp.MustCompile(`[a-zA-Z\.]+`)
	// Cache Goal syntax autocompletion keys.
	if autoCompleter.goalSyntaxKeys == nil {
		autoCompleter.cacheGoalSyntax()
	}
	return func(v [][]rune, line, col int) (string, editline.Completions) {
		perCategory := map[string][]acEntry{}
		word, start, end := computil.FindWord(v, line, col)
		// N.B. Matching system commands must come first.
		if matchesSystemCommand(word) {
			return autoCompleter.systemCommandsAutoComplete()(v, line, col)
		}
		locs := goalNameRe.FindStringIndex(word)
		if locs != nil {
			word = word[locs[0]:locs[1]]
			start = locs[0] // Preserve non-word prefix
			end = locs[1]   // Preserve non-word suffix
		}
		msg := fmt.Sprintf("Matching %v", word)
		lword := strings.ToLower(word)
		autoCompleteGoalGlobals(goalContext, lword, perCategory)
		autoCompleteGoalKeywords(autoCompleter, lword, perCategory)
		autoCompleteGoalSyntax(autoCompleter, lword, perCategory)
		// msg = fmt.Sprintf("Type is %v")
		completions := &multiComplete{
			Values:     complete.MapValues(perCategory, nil),
			moveRight:  end - col,
			deleteLeft: end - start,
		}
		// msg := ""
		return msg, completions
	}
}

func autoCompleteGoalSyntax(autoCompleter *AutoCompleter, lword string, perCategory map[string][]acEntry) {
	category := "Syntax"
	syntaxSet := make(map[string]bool, 0)
	for _, name := range autoCompleter.goalSyntaxKeys {
		chstr := autoCompleter.goalSyntaxAliases[name]
		if strings.HasPrefix(strings.ToLower(name), lword) {
			if _, ok := syntaxSet[chstr]; !ok {
				syntaxSet[chstr] = true
				var help string
				if val, ok := autoCompleter.goalSyntaxHelp[chstr]; ok {
					help = val
				} else {
					help = "Goal syntax"
				}
				perCategory[category] = append(perCategory[category], acEntry{chstr, help})
			}
		}
	}
}

func autoCompleteGoalKeywords(autoCompleter *AutoCompleter, lword string, perCategory map[string][]acEntry) {
	// Keywords can be user-defined via Go, but they are all present once Goal is initialized.
	if autoCompleter.goalKeywordsKeys == nil {
		autoCompleter.cacheGoalKeywords(autoCompleter.ariContext.GoalContext)
	}
	category := "Keyword"
	for _, goalKeyword := range autoCompleter.goalKeywordsKeys {
		if strings.HasPrefix(strings.ToLower(goalKeyword), lword) {
			var help string
			if val, ok := autoCompleter.goalKeywordsHelp[goalKeyword]; ok {
				help = val
			} else {
				help = "A Goal keyword"
			}
			perCategory[category] = append(perCategory[category], acEntry{goalKeyword, help})
		}
	}
}

func autoCompleteGoalGlobals(goalContext *goal.Context, lword string, perCategory map[string][]acEntry) {
	// Globals cannot be cached; this is what assignment in Goal creates.
	goalGlobals := goalContext.GlobalNames(nil)
	goalGlobalsHelp := ari.GoalGlobalsHelp()
	category := "Global"
	for _, goalGlobal := range goalGlobals {
		if strings.HasPrefix(strings.ToLower(goalGlobal), lword) {
			var help string
			if val, ok := goalGlobalsHelp[goalGlobal]; ok {
				help = val
			} else {
				help = "A Goal global binding"
			}
			perCategory[category] = append(perCategory[category], acEntry{goalGlobal, help})
		}
	}
}

func (autoCompleter *AutoCompleter) sqlAutoCompleteFn() func(v [][]rune, line, col int) (string, editline.Completions) {
	// Cache sorted slice of SQL keywords
	if autoCompleter.sqlKeywords == nil {
		autoCompleter.cacheSQL()
	}
	return func(v [][]rune, line, col int) (string, editline.Completions) {
		word, wstart, wend := computil.FindWord(v, line, col)
		// msg = fmt.Sprintf("Matching '%v'", word)
		perCategory := map[string][]acEntry{}
		lword := strings.ToLower(word)
		// N.B. Matching system commands must come first.
		if matchesSystemCommand(word) {
			return autoCompleter.systemCommandsAutoComplete()(v, line, col)
		}
		for _, sqlWord := range autoCompleter.sqlKeywords {
			if strings.HasPrefix(strings.ToLower(sqlWord), lword) {
				perCategory["sql"] = append(perCategory["sql"], acEntry{sqlWord, "A SQL keyword"})
			}
		}
		completions := &multiComplete{
			Values:     complete.MapValues(perCategory, nil),
			moveRight:  wend - col,
			deleteLeft: wend - wstart,
		}
		msg := ""
		return msg, completions
	}
}

func (autoCompleter *AutoCompleter) cacheGoalKeywords(goalContext *goal.Context) {
	// TODO Work out abstraction for adding help to user-defined Goal keywords.
	goalKeywords := goalContext.Keywords(nil)
	goalKeywordsHelp := ari.GoalKeywordsHelp()
	sort.Strings(goalKeywords)
	autoCompleter.goalKeywordsKeys = goalKeywords
	autoCompleter.goalKeywordsHelp = goalKeywordsHelp
}

func (autoCompleter *AutoCompleter) cacheGoalSyntax() {
	goalSyntax := ari.GoalSyntax()
	keys := make([]string, 0)
	for k := range goalSyntax {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	autoCompleter.goalSyntaxKeys = keys
	autoCompleter.goalSyntaxAliases = goalSyntax
	autoCompleter.goalSyntaxHelp = ari.GoalSyntaxHelp()
}

func (autoCompleter *AutoCompleter) cacheSQL() {
	autoCompleter.sqlKeywords = ari.SQLKeywords()
}

func (autoCompleter *AutoCompleter) cacheSystemCommands() {
	m, ks := systemCommands()
	autoCompleter.systemCommandsKeys = ks
	autoCompleter.systemCommandsHelp = m
}

func systemCommands() (map[string]string, []string) {
	m := map[string]string{
		")goal": "Goal array language mode", ")sql": "Read-only SQL mode (querying)", ")sql!": "Read/write SQL mode",
	}
	// Prepare sorted keys ahead of time
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return m, keys
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

// Bubbline

const cliDefaultReflowWidth = 80

func cliEditorInitialize() *bubbline.Editor {
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
	return editor
}
