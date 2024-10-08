package main

import (
	"bufio"
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

// CheckInputComplete handler for Goal.
//
// This detects whether a SQL statement is complete from the CLI editor's perspective,
// it does not provide autocomplete functionality.
func modeGoalCheckInputComplete(v [][]rune, line, _ int) bool {
	if len(v) == 1 && matchesSystemCommand(string(v[0])) {
		return true
	}
	if line == len(v)-1 { // Enter on final line of input to allow inserting newlines.
		s := stringFromSliceOfRuneSlices(v)
		sc := &scanner{r: bufio.NewReader(strings.NewReader(s))}
		_, err := sc.readLine()
		// No error means it parsed completely as a Goal expression.
		return err == nil
	}
	return false
}

func stringFromSliceOfRuneSlices(v [][]rune) string {
	sb := strings.Builder{}
	for _, runeSlice := range v {
		sb.WriteString(string(runeSlice))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// scanner represents the state of a readline scanner for the Goal REPL. It
// handles multi-line expressions.
type scanner struct {
	r      *bufio.Reader
	depth  []byte // (){}[] depth stack
	state  scanState
	done   bool
	escape bool
}

type scanState int

const (
	scanNormal scanState = iota
	scanComment
	scanCommentBlock
	scanString
	scanQuote
	scanRawQuote
)

const delimchars = ":+-*%!&|=~,^#_?@/`"

// readLine reads until the first end of line that also ends a Goal expression.
//
// Adapated from Goal's implementation.
//
//nolint:cyclop,funlen,gocognit,gocyclo // Vendored code
func (sc *scanner) readLine() (string, error) {
	*sc = scanner{r: sc.r, depth: sc.depth[:0]}
	sb := strings.Builder{}
	var qr byte = '/'
	nl := true       // at newline
	cs := true       // at possible start of comment
	cbdelim := false // after possible comment block start/end delimiter
	for {
		c, err := sc.r.ReadByte()
		if err != nil {
			return sb.String(), err
		}
		switch c {
		case '\r':
			continue
		default:
			sb.WriteByte(c)
		}
		switch sc.state {
		case scanNormal:
			switch c {
			case '\n':
				if len(sc.depth) == 0 || sc.done {
					return sb.String(), nil
				}
				cs = true
			case ' ', '\t':
				cs = true
			case '"':
				sc.state = scanString
				cs = false
			case '{', '(', '[':
				sc.depth = append(sc.depth, c)
				cs = true
			case '}', ')', ']':
				if len(sc.depth) > 0 && sc.depth[len(sc.depth)-1] == opening(c) {
					sc.depth = sc.depth[:len(sc.depth)-1]
				} else {
					// error, so return on next \n
					sc.done = true
				}
				cs = false
			default:
				if strings.IndexByte(delimchars, c) != -1 {
					acc := sb.String()
					switch {
					case strings.HasSuffix(acc[:len(acc)-1], "rx"):
						qr = c
						sc.state = scanQuote
					case strings.HasSuffix(acc[:len(acc)-1], "rq"):
						qr = c
						sc.state = scanRawQuote
					case strings.HasSuffix(acc[:len(acc)-1], "qq"):
						qr = c
						sc.state = scanQuote
					default:
						if c == '/' && cs {
							sc.state = scanComment
							cbdelim = nl
						}
					}
				}
				cs = false
			}
		case scanComment:
			if c == '\n' {
				//nolint:gocritic // vendored code
				if cbdelim {
					sc.state = scanCommentBlock
				} else if len(sc.depth) == 0 || sc.done {
					return sb.String(), nil
				} else {
					cs = true
					sc.state = scanNormal
				}
			}
			cbdelim = false
		case scanCommentBlock:
			if cbdelim && c == '\n' {
				if len(sc.depth) == 0 || sc.done {
					return sb.String(), nil
				}
				cs = true
				sc.state = scanNormal
			} else {
				cbdelim = nl && c == '\\'
			}
		case scanQuote:
			switch c {
			case '\\':
				sc.escape = !sc.escape
			case qr:
				if !sc.escape {
					sc.state = scanNormal
				}
				sc.escape = false
			default:
				sc.escape = false
			}
		case scanString:
			switch c {
			case '\\':
				sc.escape = !sc.escape
			case '"':
				if !sc.escape {
					sc.state = scanNormal
				}
				sc.escape = false
			default:
				sc.escape = false
			}
		case scanRawQuote:
			if c == qr {
				//nolint:govet // vendored code
				c, err := sc.r.ReadByte()
				if err != nil {
					return sb.String(), err
				}
				if c == qr {
					sb.WriteByte(c)
				} else {
					//nolint:errcheck // Goal impl says cannot error
					sc.r.UnreadByte() // cannot error
					sc.state = scanNormal
				}
			}
		}
		nl = c == '\n'
	}
}

// opening returns matching opening delimiter for a given closing delimiter.
//
// Adapted from Goal's implementation.
func opening(r byte) byte {
	switch r {
	case ')':
		return '('
	case ']':
		return '['
	case '}':
		return '{'
	default:
		return r
	}
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
		// msg := fmt.Sprintf("Matching %v", word)
		lword := strings.ToLower(word)
		autoCompleteGoalGlobals(autoCompleter, lword, perCategory)
		autoCompleteGoalKeywords(autoCompleter, lword, perCategory)
		autoCompleteGoalSyntax(autoCompleter, lword, perCategory)
		// msg = fmt.Sprintf("Type is %v")
		completions := &multiComplete{
			Values:     complete.MapValues(perCategory, nil),
			moveRight:  end - col,
			deleteLeft: end - start,
		}
		msg := ""
		return msg, completions
	}
}

func autoCompleteGoalSyntax(autoCompleter *AutoCompleter, lword string, perCategory map[string][]acEntry) {
	helpFunc := autoCompleter.ariContext.Help.Func
	category := "Syntax"
	syntaxSet := make(map[string]bool, 0)
	for _, name := range autoCompleter.goalSyntaxKeys {
		chstr := autoCompleter.goalSyntaxAliases[name]
		if strings.HasPrefix(strings.ToLower(name), lword) {
			if _, ok := syntaxSet[chstr]; !ok {
				syntaxSet[chstr] = true
				help := helpFunc(chstr)
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
	helpFunc := autoCompleter.ariContext.Help.Func
	category := "Keyword"
	for _, goalKeyword := range autoCompleter.goalKeywordsKeys {
		if strings.HasPrefix(strings.ToLower(goalKeyword), lword) {
			help := helpFunc(goalKeyword)
			perCategory[category] = append(perCategory[category], acEntry{goalKeyword, help})
		}
	}
}

func autoCompleteGoalGlobals(autoCompleter *AutoCompleter, lword string, perCategory map[string][]acEntry) {
	goalContext := autoCompleter.ariContext.GoalContext
	helpFunc := autoCompleter.ariContext.Help.Func
	// Globals cannot be cached; this is what assignment in Goal creates.
	goalGlobals := goalContext.GlobalNames(nil)
	sort.Strings(goalGlobals)
	category := "Global"
	for _, goalGlobal := range goalGlobals {
		if strings.HasPrefix(strings.ToLower(goalGlobal), lword) {
			help := helpFunc(goalGlobal)
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
		")goal": "Goal array language mode",
		// TODO Output formats: https://duckdb.org/docs/api/cli/output_formats.html
		// In particular csv, json, markdown, latex, and one of the boxed ones
		")output.csv":         "Print results as CSV",
		")output.goal":        "Print results as Goal values (default)",
		")output.json":        "Print results as JSON",
		")output.json+pretty": "Print results as JSON with indentation",
		")output.latex":       "Print results as LaTeX",
		")output.markdown":    "Print results as Markdown",
		")output.tsv":         "Print results as TSV",
		")sql":                "Read-only SQL mode (querying)",
		")sql!":               "Read/write SQL mode",
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

func cliEditorInitialize() *bubbline.Editor {
	editor := bubbline.New()
	editor.Placeholder = ""
	historyFile := viper.GetString("history")
	if err := editor.LoadHistory(historyFile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load history, error: %v\n", err)
	}
	editor.SetAutoSaveHistory(historyFile, true)
	editor.SetDebugEnabled(false)
	editor.SetExternalEditorEnabled(true, "goal")
	return editor
}
