//go:build js && wasm

package ari

import (
	"errors"

	goal "codeberg.org/anaseto/goal"
)

// Under js/wasm the SQL (modernc/sqlite) and Fyne (GUI) extensions are
// excluded from the build. Asking for them is a configuration error.

var (
	errSQLUnavailable  = errors.New("ari: SQL extension is not available in js/wasm builds")
	errFyneUnavailable = errors.New("ari: Fyne extension is not available in js/wasm builds")
)

func importSQL(_ *goal.Context, _ string) error  { return errSQLUnavailable }
func importFyne(_ *goal.Context, _ string) error { return errFyneUnavailable }
