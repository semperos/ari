// Package ari provides an embeddable Goal interpreter pre-configured with
// the ari extension verbs (SQL, HTTP, rate-limiting, optional Fyne GUI),
// the upstream Goal standard libraries (math/base64/zip/os/io), and two
// embedded Goal source trees mounted as globals:
//
//   - goallib  – Goal stdlib source (math, fmt, fs, os, …)
//   - arilib   – ari helper source (util, stats, set, …)
//
// Typical usage:
//
//	ctx, err := ari.New(ari.DefaultOptions())
//	if err != nil { log.Fatal(err) }
//	v, err := ctx.Eval(`1+1`)
//
// All Goal source files live in this package; no external files need to
// travel with a binary that imports it.
package ari

import (
	"embed"
	"io/fs"

	goal "codeberg.org/anaseto/goal"
	goalzip "codeberg.org/anaseto/goal/archive/zip"
	goalbase64 "codeberg.org/anaseto/goal/encoding/base64"
	goalfs "codeberg.org/anaseto/goal/io/fs"
	goalmath "codeberg.org/anaseto/goal/math"
	gos "codeberg.org/anaseto/goal/os"

	goalhttp "github.com/semperos/ari/http"
	goalratelimit "github.com/semperos/ari/ratelimit"
)

//go:embed goallib
var goallibFS embed.FS

//go:embed lib
var libFS embed.FS

// GoallibFS returns the embedded Goal stdlib filesystem rooted at "goallib/".
// Useful for callers that want to mount the sources somewhere other than the
// default "goallib" global.
func GoallibFS() fs.FS {
	sub, err := fs.Sub(goallibFS, "goallib")
	if err != nil {
		panic(err) // statically known to exist
	}
	return sub
}

// ArilibFS returns the embedded ari helper filesystem rooted at "lib/".
func ArilibFS() fs.FS {
	sub, err := fs.Sub(libFS, "lib")
	if err != nil {
		panic(err)
	}
	return sub
}

// New constructs a fresh *goal.Context configured per opts. It is shorthand
// for goal.NewContext() followed by NewContext(ctx, opts).
func New(opts Options) (*goal.Context, error) {
	ctx := goal.NewContext()
	if err := NewContext(ctx, opts); err != nil {
		return nil, err
	}
	return ctx, nil
}

// NewContext installs the ari extensions and embedded source roots into an
// existing Goal context. It is safe to call on a context that already has
// some verbs registered; existing globals with conflicting names are
// overwritten (this matches Goal's own Import semantics).
func NewContext(ctx *goal.Context, opts Options) error {
	pfx := opts.Prefix

	if opts.EnableOSIO {
		gos.Import(ctx, pfx)
	}
	if opts.EnableMath {
		goalmath.Import(ctx, pfx)
	}
	if opts.EnableBase64 {
		goalbase64.Import(ctx, pfx)
	}
	if opts.EnableZip {
		goalzip.Import(ctx, pfx)
	}
	if opts.EnableRateLimit {
		goalratelimit.Import(ctx, pfx)
	}
	if opts.EnableHTTP {
		goalhttp.Import(ctx, pfx)
	}
	if opts.EnableSQL {
		if err := importSQL(ctx, pfx); err != nil {
			return err
		}
	}
	if opts.EnableFyne {
		if err := importFyne(ctx, pfx); err != nil {
			return err
		}
	}

	if opts.EnableArilib {
		ctx.AssignGlobal("arilib", goalfs.NewFS(ArilibFS(), "arilib"))
	}
	if opts.EnableGoallib {
		ctx.AssignGlobal("goallib", goalfs.NewFS(GoallibFS(), "goallib"))
	}

	return nil
}
