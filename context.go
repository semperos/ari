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

// Initialize SQL struct, but don't open the DB yet.
//
// Call SQLDatabase.open to open the database.
func NewSQLDatabase(dataSourceName string) (*SQLDatabase, error) {
	return &SQLDatabase{DataSource: dataSourceName, DB: nil, IsOpen: false}, nil
}

func NewHelp() map[string]map[string]string {
	defaultSQLHelp := "A SQL keyword"
	// defaultGoalHelp := "A Goal keyword"
	goalGlobalsHelp := GoalGlobalsHelp()
	goalKeywordsHelp := GoalKeywordsHelp()
	goalSyntaxHelp := GoalSyntaxHelp()
	goalHelp := make(map[string]string, len(goalGlobalsHelp)+len(goalKeywordsHelp)+len(goalSyntaxHelp))
	for k, v := range goalGlobalsHelp {
		goalHelp[k] = v
	}
	for k, v := range goalKeywordsHelp {
		goalHelp[k] = v
	}
	for k, v := range goalSyntaxHelp {
		goalHelp[k] = v
	}
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
	// TODO HERE IS WHERE WE Wrap() a function that closes over helpDictionary to provide lookups for user-added help strings
	help := Help{Dictionary: helpDictionary, Func: help.HelpFunc()}
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
