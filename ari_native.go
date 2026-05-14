//go:build !js || !wasm

package ari

import (
	goal "codeberg.org/anaseto/goal"

	arifyne "github.com/semperos/ari/fyne"
	goalsql "github.com/semperos/ari/sql"
)

// Native builds register Fyne and SQL verbs directly.

func importSQL(ctx *goal.Context, pfx string) error {
	goalsql.Import(ctx, pfx)
	return nil
}

func importFyne(ctx *goal.Context, pfx string) error {
	arifyne.Import(ctx, pfx)
	return nil
}
