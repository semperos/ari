package ari

import (
	"database/sql"
	"os"

	"codeberg.org/anaseto/goal"
)

// See application entrypoints for the full instantiation of this struct.
var GlobalContext Context

type Context struct {
	GoalContext *goal.Context
	SqlDatabase *SqlDatabase
}

// Initialize a Goal language context and set the corresponding Context field.
func ContextInitGoal(context *Context) error {
	goalContext := goal.NewContext()
	goalContext.Log = os.Stderr
	goalRegisterVariadics(goalContext)
	err := goalLoadExtendedPreamble(goalContext)
	if err != nil {
		return err
	}
	context.GoalContext = goalContext
	return nil
}

// Initialize an SQL database and set the corresponding Context field.
//
// Note that the underlying sql.DB is open, and the caller is responsible for closing it.
func ContextInitSql(context *Context, dataSourceName string) error {
	db, err := sql.Open(dbDriver, dataSourceName)
	if err != nil {
		return err
	}
	context.SqlDatabase = &SqlDatabase{DataSource: dataSourceName, DB: db, IsOpen: true}
	return nil
}
