//go:build js && wasm

package ari

import (
	goal "codeberg.org/anaseto/goal"
	gos "codeberg.org/anaseto/goal/os"
)

// importPlatformExtensions registers Goal's IO/FS verbs in WASM builds.
//
// Fyne GUI and SQL require CGo and are always excluded. The OS option is
// intentionally ignored here: the "import" dyad (and companions like "read",
// "print", "say", "glob", "subfs") work against embedded fs.FS values
// (e.g. arilib, goallib) without any OS access. Registering them via
// gos.Import is the only way to make `arilib import "lib"` work in the
// browser REPL. OS-only verbs (chdir, mkdir, run, shell, …) are merely
// registered but will return errors if called at runtime, which is
// acceptable. The SQL option is ignored because CGo is unavailable.
func importPlatformExtensions(ctx *goal.Context, _ Options) {
	gos.Import(ctx, "")
}
