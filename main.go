//go:build !js || !wasm

// ari: a Goal interpreter with extensions for SQL, HTTP, and GUI (Fyne)
package main

import (
	"embed"
	"io/fs"
	"os"

	goal "codeberg.org/anaseto/goal"
	goalzip "codeberg.org/anaseto/goal/archive/zip"
	"codeberg.org/anaseto/goal/cmd"
	goalbase64 "codeberg.org/anaseto/goal/encoding/base64"
	goalfs "codeberg.org/anaseto/goal/io/fs"
	goalmath "codeberg.org/anaseto/goal/math"
	gos "codeberg.org/anaseto/goal/os"
	arihelp "github.com/semperos/ari/help"

	arifyne "github.com/semperos/ari/fyne"
	goalhttp "github.com/semperos/ari/http"
	goalratelimit "github.com/semperos/ari/ratelimit"
	goalsql "github.com/semperos/ari/sql"
)

//go:embed lib
var libFS embed.FS

//go:embed goallib
var goallibFS embed.FS

func main() {
	ctx := goal.NewContext()
	ctx.Log = os.Stderr

	// Standard OS/IO verbs (print, say, read, open, run, shell, …)
	gos.Import(ctx, "")

	// Math verbs (cos, sin, sqrt, round, …)
	goalmath.Import(ctx, "")

	// Base64 verbs (base64.enc, base64.dec, base64.urlenc, base64.urldec)
	goalbase64.Import(ctx, "")

	// Zip verbs (zip.open, zip.write)
	goalzip.Import(ctx, "")

	// Fyne GUI verbs (fyne.app, fyne.window, fyne.label, fyne.button, …)
	arifyne.Import(ctx, "")

	// Rate limiting verbs (ratelimit.new, ratelimit.take)
	goalratelimit.Import(ctx, "")

	// HTTP client verbs (http.client, http.get, http.post, …)
	goalhttp.Import(ctx, "")

	// SQL verbs (sql.open, sql.close, sql.q, sql.exec, sql.tx)
	goalsql.Import(ctx, "")

	// arilib: embedded lib/ — `arilib import "util"` loads lib/util.goal
	sub, err := fs.Sub(libFS, "lib")
	if err != nil {
		panic(err)
	}
	ctx.AssignGlobal("arilib", goalfs.NewFS(sub, "arilib"))

	// goallib: embedded Goal standard lib — goallib import "fmt" loads goallib/fmt.goal
	goalsub, err := fs.Sub(goallibFS, "goallib")
	if err != nil {
		panic(err)
	}
	ctx.AssignGlobal("goallib", goalfs.NewFS(goalsub, "goallib"))

	cmd.Exit(cmd.Run(ctx, cmd.Config{
		ProgramName: "ari",
		Help:        arihelp.HelpFunc(),
	}))
}
