package ari

import (
	"database/sql"
	"os"

	"codeberg.org/anaseto/goal"
)

type Context struct {
	GoalContext *goal.Context
	SQLDatabase *SQLDatabase
}

// Initialize a Goal language context with Ari's extensions.
func InitializeGoal(sqlDatabase *SQLDatabase) (*goal.Context, error) {
	goalContext := goal.NewContext()
	goalContext.Log = os.Stderr
	goalRegisterVariadics(goalContext, sqlDatabase)
	err := goalLoadExtendedPreamble(goalContext)
	if err != nil {
		return nil, err
	}
	return goalContext, nil
}

// Initialize an SQL database.
//
// Note that the underlying sql.DB is open, and the caller is responsible for closing it.
func InitializeSQL(dataSourceName string) (*SQLDatabase, error) {
	db, err := sql.Open(dbDriver, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &SQLDatabase{DataSource: dataSourceName, DB: db, IsOpen: true}, nil
}

func NewContext(dataSourceName string) (*Context, error) {
	sqlDatabase, err := InitializeSQL(dataSourceName)
	if err != nil {
		return nil, err
	}
	goalContext, err := InitializeGoal(sqlDatabase)
	if err != nil {
		return nil, err
	}
	return &Context{GoalContext: goalContext, SQLDatabase: sqlDatabase}, nil
}
