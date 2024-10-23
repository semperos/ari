// Ari is an environment for Array Relational Interactive programming.
//
// Ari's code base consists of two parts:
//
//   - Go library at `github.com/semperos/ari`
//   - Go CLI application at `github.com/semperos/ari/cmd/ari`
//
// ## Library ari
//
// The ari library at `github.com/semperos/ari` extends the Goal programming language
// with new functions for:
//
//   - Unit testing in Goal
//   - Date & time handling (via `time`)
//   - HTTP client (via `github.com/go-resty/resty/v2`) and server (via `net/http`)
//   - SQL client (via `database/sql`)
//
// Goal functions defined in Go must return a valid Goal value. The ari types that
// satisfy `goal.BV` represent such values in this code base. Their Goal string
// representation in Goal is not meant to be readable by the language, but instead are
// designed to expose as much information as possible to avoid the need to create other
// means by which to interrogate that state.
//
// Ari attempts to keep the names of Go functions and fields intact across the
// board, including from Goal code. For example, the options for configuring
// an HTTP client in Goal have names like `Body` and `QueryParam` to match what the
// underlying go-resty code expects. Even though this results in non-idiomatic
// names in Goal, it has been done to reduce overall cognitive load.
//
// ## CLI ari
//
// The CLI application at `github.com/semperos/ari/cmd/ari` provides a rich REPL with
// multi-line editing, history, and auto-complete functionality. This interface is ideal when
// first learning Goal, using Goal as an ad hoc calculator, or when building a REPL-
// based user experience.
//
// To drive ari's Goal REPL from an editor, you should prefer the `--raw` REPL,
// which does not have line editing, history, or auto-complete features, but which has
// better performance. You can use the Goal `ac` function with a string argument for glob matching
// or a regular expression argument for loose regex matching of all bindings in your Goal
// environment, optionally passing those results to `help` to view their help strings.
//
// Concretely, the CLI app adds the following on top of the base ari library:
//
//   - A rich REPL using `github.com/knz/bubbline` (built on `github.com/charmbracelet/bubbletea`)
//     with dedicated Goal and SQL modes, system commands starting with `)`, multiple output
//     formats, basic profiling and debugging capabilities, and ability to programmatically
//     change the REPL prompt.
//   - A `help` Goal verb that allows (re)defining help strings for globals/keywords in Goal
//   - A dependency on `github.com/marcboeker/go-duckdb` to use DuckDB as the SQL database
//   - TUI functions for basic terminal styling
//   - Common configuration options can be saved to a `$HOME/.config/ari/ari-config.yaml` file
//     for reuse, which are overridden when CLI arguments are provided.
package ari

import (
	"os"

	"codeberg.org/anaseto/goal"
	"github.com/semperos/ari/vendored/help"
)

type Help struct {
	Dictionary map[string]map[string]string
	Func       func(string) string
}

type Context struct {
	// GoalContext is needed to evaluate Goal programs and introspect the Goal execution environment.
	GoalContext *goal.Context
	// HTTPClient exposed for testing purposes.
	HTTPClient *HTTPClient
	// SQLDatabase keeps track of open database connections as well as the data source name.
	SQLDatabase *SQLDatabase
	// Help stores documentation information for identifiers.
	// The top-level keys must match a modes Name output;
	// the inner maps are a mapping from mode-specific identifiers
	// to a string that describes them and which is user-facing.
	Help Help
}

// Initialize a Goal language context with Ari's extensions.
func NewGoalContext(ariContext *Context, help Help, sqlDatabase *SQLDatabase) (*goal.Context, error) {
	goalContext := goal.NewContext()
	goalContext.Log = os.Stderr
	goalRegisterVariadics(ariContext, goalContext, help, sqlDatabase)
	err := goalLoadExtendedPreamble(goalContext)
	if err != nil {
		return nil, err
	}
	return goalContext, nil
}

// Initialize a Goal language context with Ari's extensions.
func NewUniversalGoalContext(ariContext *Context, help Help) (*goal.Context, error) {
	goalContext := goal.NewContext()
	goalContext.Log = os.Stderr
	goalRegisterUniversalVariadics(ariContext, goalContext, help)
	err := goalLoadExtendedPreamble(goalContext)
	if err != nil {
		return nil, err
	}
	return goalContext, nil
}

// Initialize SQL struct, but don't open the DB yet.
//
// Call SQLDatabase.open to open the database.
func NewSQLDatabase(dataSourceName string) (*SQLDatabase, error) {
	return &SQLDatabase{DataSource: dataSourceName, DB: nil, IsOpen: false}, nil
}

func NewHelp() map[string]map[string]string {
	defaultSQLHelp := "A SQL keyword"
	goalHelp := GoalKeywordsHelp()
	sqlKeywords := SQLKeywords()
	sqlHelp := make(map[string]string, len(sqlKeywords))
	for _, x := range sqlKeywords {
		sqlHelp[x] = defaultSQLHelp
	}
	help := make(map[string]map[string]string, 0)
	help["goal"] = goalHelp
	help["sql"] = sqlHelp
	return help
}

// Initialize a new Context without connecting to the database.
func NewContext(dataSourceName string) (*Context, error) {
	ctx := Context{}
	helpDictionary := NewHelp()
	ariHelpFunc := func(s string) string {
		goalHelp, ok := helpDictionary["goal"]
		if !ok {
			panic(`Developer Error: Dictionary in Help must have a \"goal\" entry.`)
		}
		help, found := goalHelp[s]
		if found {
			return help
		}
		return ""
	}
	helpFunc := help.Wrap(ariHelpFunc, help.HelpFunc())
	help := Help{Dictionary: helpDictionary, Func: helpFunc}
	sqlDatabase, err := NewSQLDatabase(dataSourceName)
	if err != nil {
		return nil, err
	}
	goalContext, err := NewGoalContext(&ctx, help, sqlDatabase)
	if err != nil {
		return nil, err
	}
	ctx.GoalContext = goalContext
	ctx.SQLDatabase = sqlDatabase
	ctx.Help = help
	return &ctx, nil
}

// Initialize a new Context that can be used across platforms, including WASM.
func NewUniversalContext() (*Context, error) {
	ctx := Context{}
	helpDictionary := NewHelp()
	ariHelpFunc := func(s string) string {
		goalHelp, ok := helpDictionary["goal"]
		if !ok {
			panic(`Developer Error: Dictionary in Help must have a \"goal\" entry.`)
		}
		help, found := goalHelp[s]
		if found {
			return help
		}
		return ""
	}
	helpFunc := help.Wrap(ariHelpFunc, help.HelpFunc())
	help := Help{Dictionary: helpDictionary, Func: helpFunc}
	goalContext, err := NewUniversalGoalContext(&ctx, help)
	if err != nil {
		return nil, err
	}
	ctx.GoalContext = goalContext
	ctx.Help = help
	return &ctx, nil
}
