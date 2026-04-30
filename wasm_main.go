//go:build js && wasm

// Browser REPL for ari – Goal + ari extensions (WASM build).
//
// Compile with:
//
//	GOOS=js GOARCH=wasm go build -o wasm/ari.wasm .
//
// Then copy wasm_exec.js:
//
//	cp $(go env GOROOT)/lib/wasm/wasm_exec.js wasm/
//
// Serve wasm/ over HTTP to use, e.g.:
//
//	cd wasm && python3 -m http.server 8080
//
// Note: fyne (GUI) and sql (SQLite/CGo) are excluded — they don't apply in
// a browser context. The math, base64, zip, http, and ratelimit extensions
// are included.
package main

import (
	"compress/zlib"
	"embed"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"log"
	"runtime/debug"
	"strings"
	"syscall/js"

	"codeberg.org/anaseto/goal"
	goalzip "codeberg.org/anaseto/goal/archive/zip"
	goalbase64 "codeberg.org/anaseto/goal/encoding/base64"
	goalmath "codeberg.org/anaseto/goal/math"

	arihelp "github.com/semperos/ari/help"
	goalhttp "github.com/semperos/ari/http"
	goalratelimit "github.com/semperos/ari/ratelimit"
)

//go:embed goallib
var wasmGoallibFS embed.FS

//go:embed lib
var wasmLibFS embed.FS

// wasmEmbeddedLib wraps an fs.FS as a Goal boxed value so that it can be
// used as the left-hand argument to the `import` dyad:
//
//	arilib import "util"   →  loads lib/util.goal
//	goallib import "fmt"   →  loads goallib/fmt.goal
type wasmEmbeddedLib struct {
	fs.FS
	name string
}

func (e *wasmEmbeddedLib) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	return append(dst, e.name...)
}
func (e *wasmEmbeddedLib) Matches(y goal.BV) bool {
	yv, ok := y.(*wasmEmbeddedLib)
	return ok && e.name == yv.name
}
func (e *wasmEmbeddedLib) Type() string { return e.name }

// ariCtx is the persistent Goal context for the browser REPL session.
// Use resetAriCtx() to clear state between sessions.
var ariCtx *goal.Context

func buildAriCtx() *goal.Context {
	ctx := goal.NewContext()

	goalmath.Import(ctx, "")
	goalbase64.Import(ctx, "")
	goalzip.Import(ctx, "")
	goalratelimit.Import(ctx, "")
	goalhttp.Import(ctx, "")

	goalsub, err := fs.Sub(wasmGoallibFS, "goallib")
	if err != nil {
		panic(err)
	}
	ctx.AssignGlobal("goallib", goal.NewV(&wasmEmbeddedLib{goalsub, "goallib"}))

	sub, err := fs.Sub(wasmLibFS, "lib")
	if err != nil {
		panic(err)
	}
	ctx.AssignGlobal("arilib", goal.NewV(&wasmEmbeddedLib{sub, "arilib"}))

	return ctx
}

func getElt(id string) js.Value {
	return js.Global().Get("document").Call("getElementById", id)
}

func evalTextArea() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Caught panic: %v\nStack Trace:\n", r)
			debug.PrintStack()
			getElt("out").Set("value", fmt.Sprintf("Caught panic: %v\n", r))
		}
	}()
	in := getElt("in").Get("value").String()
	out := getElt("out")
	var sb strings.Builder
	ariCtx.Log = &sb
	x, err := ariCtx.Eval(in)
	if err != nil {
		if e, ok := err.(*goal.Panic); ok {
			out.Set("value", sb.String()+e.ErrorStack())
		} else {
			out.Set("value", sb.String()+err.Error())
		}
	} else {
		out.Set("value", sb.String()+x.Sprint(ariCtx, false))
	}
	updateHash()
}

func updateHash() {
	var sb strings.Builder
	in := getElt("in").Get("value").String()
	b64w := base64.NewEncoder(base64.RawURLEncoding, &sb)
	zw := zlib.NewWriter(b64w)
	sb.WriteByte('#')
	fmt.Fprint(zw, in)
	zw.Flush()
	zw.Close()
	b64w.Close()
	js.Global().Get("window").Get("location").Set("hash", sb.String())
}

func updateTextArea() {
	hash := js.Global().Get("window").Get("location").Get("hash").String()
	in := getElt("in")
	if hash == "" {
		in.Set("value", "")
		return
	}
	r := base64.NewDecoder(base64.RawURLEncoding, strings.NewReader(hash[1:]))
	zr, err := zlib.NewReader(r)
	if err != nil {
		log.Printf("zlib reader: %v", err)
		log.Printf("hash: %q", hash)
		return
	}
	defer func() {
		if err := zr.Close(); err != nil {
			log.Printf("zlib reader: close: %v", err)
		}
	}()
	var sb strings.Builder
	if _, err = io.Copy(&sb, zr); err != nil {
		log.Printf("decoding hash: %v", err)
		log.Printf("hash: %q", hash)
		return
	}
	log.Print(sb.String())
	in.Set("value", sb.String())
}

func main() {
	ariCtx = buildAriCtx()
	helpFn := arihelp.HelpFunc()

	updateTextArea()

	// eval button and ctrl-enter
	getElt("eval").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		evalTextArea()
		return nil
	}))
	getElt("in").Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		key := e.Get("key").String()
		switch {
		case e.Get("ctrlKey").Bool() && key == "Enter":
			e.Call("preventDefault")
			evalTextArea()
		case key == "F1":
			getElt("out").Set("value", helpFn(""))
		}
		return nil
	}))

	// copy link to clipboard
	getElt("link").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		updateHash()
		href := js.Global().Get("window").Get("location").Get("href")
		js.Global().Get("navigator").Get("clipboard").Call("writeText", href)
		return nil
	}))

	// help button
	getElt("help").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		getElt("out").Set("value", helpFn(""))
		return nil
	}))

	// reset button – clear context state between experiments
	getElt("reset").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		ariCtx = buildAriCtx()
		getElt("out").Set("value", "")
		return nil
	}))

	// keep hash in sync with textarea
	js.Global().Get("window").Call("addEventListener", "hashchange", js.FuncOf(func(this js.Value, args []js.Value) any {
		updateTextArea()
		return nil
	}))

	getElt("ariVersion").Set("textContent", fmt.Sprintf("ari %s (wasm)", ariCtx.Version()))

	<-make(chan bool)
}
