//go:build js && wasm

package ari

import goal "codeberg.org/anaseto/goal"

// importPlatformExtensions is a no-op in WASM builds.
// OS I/O, Fyne GUI, and SQL require CGo or native OS access and are not
// available in the browser. The opts fields for these are silently ignored.
func importPlatformExtensions(_ *goal.Context, _ Options) {}
