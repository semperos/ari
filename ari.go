package ari

import (
	"codeberg.org/anaseto/goal"
	gos "codeberg.org/anaseto/goal/os"
)

func goalRegisterVariadics(goalContext *goal.Context) {
	// From Goal itself
	gos.Import(goalContext, "")
	// Ari
	goalContext.RegisterDyad("http.client", VFHttpClient)
	goalContext.RegisterDyad("http.get", VFHTTPMaker("GET"))
	goalContext.RegisterDyad("http.post", VFHTTPMaker("POST"))
	goalContext.RegisterDyad("http.put", VFHTTPMaker("PUT"))
	goalContext.RegisterDyad("http.delete", VFHTTPMaker("DELETE"))
	goalContext.RegisterDyad("http.patch", VFHTTPMaker("PATCH"))
	goalContext.RegisterDyad("http.head", VFHTTPMaker("HEAD"))
	goalContext.RegisterDyad("http.options", VFHTTPMaker("OPTIONS"))
	goalContext.RegisterDyad("sql.open", VFSqlOpen)
	goalContext.RegisterDyad("sql.q", VFSqlQ)
}
