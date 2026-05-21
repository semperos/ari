// Package ari provides a Goal interpreter with extensions for SQL, HTTP,
// and GUI (Fyne), consumable as a library or via the cmd/ari CLI.
//
// Create a fully-configured Goal context with [New]:
//
//	ctx, err := ari.New(ari.FullOptions())
//
// Embed just the cross-platform subset (no CGo, works in WASM) with:
//
//	ctx, err := ari.New(ari.DefaultOptions())
package ari

import (
	"embed"
	"io/fs"

	goal "codeberg.org/anaseto/goal"
	goalzip "codeberg.org/anaseto/goal/archive/zip"
	goalbase64 "codeberg.org/anaseto/goal/encoding/base64"
	goalfs "codeberg.org/anaseto/goal/io/fs"
	goalmath "codeberg.org/anaseto/goal/math"

	goalhttp "github.com/semperos/ari/http"
	goalratelimit "github.com/semperos/ari/ratelimit"
)

//go:embed lib
var libFS embed.FS

//go:embed goallib
var goallibFS embed.FS

// Options configures which extensions are loaded into the Goal context
// returned by [New].
type Options struct {
	// OS enables standard OS/IO verbs (print, say, read, open, run, shell, …).
	// Not available in WASM builds; this field is ignored there.
	OS bool
	// Fyne enables the Fyne GUI verbs (fyne.app, fyne.window, fyne.label, …).
	// Requires CGo; not available in WASM builds.
	Fyne bool
	// SQL enables the SQL database verbs (sql.open, sql.q, sql.exec, …).
	// Requires CGo; not available in WASM builds.
	SQL bool
}

// DefaultOptions returns a base set of options with only cross-platform
// (non-CGo) extensions enabled. Suitable for WASM builds and lightweight
// embeddings that don't require GUI or database access.
func DefaultOptions() Options {
	return Options{}
}

// FullOptions returns options that enable all available extensions, including
// OS I/O, Fyne GUI, and SQL. Fyne and SQL require CGo and are silently
// omitted in WASM builds regardless of this setting.
func FullOptions() Options {
	return Options{
		OS:   true,
		Fyne: true,
		SQL:  true,
	}
}

// New creates a Goal context configured with the ari extensions requested by
// opts. The returned context has the following always-enabled extensions:
//
//   - Math verbs (cos, sin, sqrt, round, …)
//   - Base64 verbs (base64.enc, base64.dec, …)
//   - Zip verbs (zip.open, zip.write)
//   - Rate-limit verbs (ratelimit.new, ratelimit.take)
//   - HTTP client verbs (http.client, http.get, http.post, …)
//   - arilib global (embedded lib/)
//   - goallib global (embedded goallib/)
//
// Platform-specific extensions (OS I/O, Fyne, SQL) are added when the
// corresponding Options field is true and the build supports them.
func New(opts Options) (*goal.Context, error) {
	ctx := goal.NewContext()

	// Math verbs (cos, sin, sqrt, round, …)
	goalmath.Import(ctx, "")

	// Base64 verbs (base64.enc, base64.dec, base64.urlenc, base64.urldec)
	goalbase64.Import(ctx, "")

	// Zip verbs (zip.open, zip.write)
	goalzip.Import(ctx, "")

	// Rate limiting verbs (ratelimit.new, ratelimit.take)
	goalratelimit.Import(ctx, "")

	// HTTP client verbs (http.client, http.get, http.post, …)
	goalhttp.Import(ctx, "")

	// Platform-specific extensions (OS I/O, Fyne GUI, SQL).
	// Registered by build-tagged files: setup_native.go / setup_wasm.go.
	importPlatformExtensions(ctx, opts)

	// arilib: embedded lib/ — `arilib import "util"` loads lib/util.goal
	sub, err := fs.Sub(libFS, "lib")
	if err != nil {
		return nil, err
	}
	ctx.AssignGlobal("arilib", goalfs.NewFS(sub, "arilib"))

	// goallib: embedded Goal standard lib — `goallib import "fmt"` loads goallib/fmt.goal
	goalsub, err := fs.Sub(goallibFS, "goallib")
	if err != nil {
		return nil, err
	}
	ctx.AssignGlobal("goallib", goalfs.NewFS(goalsub, "goallib"))

	return ctx, nil
}
