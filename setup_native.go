//go:build !js || !wasm

package ari

import (
	"os"

	goal "codeberg.org/anaseto/goal"
	arifyne "github.com/semperos/ari/fyne"
	gos "codeberg.org/anaseto/goal/os"
	goalsql "github.com/semperos/ari/sql"
)

// importPlatformExtensions registers OS I/O, Fyne GUI, and SQL extensions
// into ctx based on opts. These all require CGo or OS access and are
// unavailable in WASM builds.
func importPlatformExtensions(ctx *goal.Context, opts Options) {
	if opts.OS {
		ctx.Log = os.Stderr
		// Standard OS/IO verbs (print, say, read, open, run, shell, …)
		gos.Import(ctx, "")
	}
	if opts.Fyne {
		// Fyne GUI verbs (fyne.app, fyne.window, fyne.label, fyne.button, …)
		arifyne.Import(ctx, "")
	}
	if opts.SQL {
		// SQL verbs (sql.open, sql.close, sql.q, sql.exec, sql.tx)
		goalsql.Import(ctx, "")
	}
}
