//go:build js && wasm

// Browser REPL for ari – Goal + ari extensions (WASM build).
//
// This binary exposes a clean JavaScript API on window.ari so that any HTML
// page can wire up its own DOM elements without depending on specific element
// IDs. When the interpreter is ready, it calls window.ariOnReady() if that
// function is defined, allowing the page to perform its own initialisation.
//
// # JavaScript API
//
//	ari.eval(code)           → string  evaluate Goal source; returns result+log
//	ari.reset()                        reinitialise the interpreter
//	ari.help(token)          → string  return help text for the given token
//	ari.version()            → string  return ari version string
//	ari.encodeState(code)    → string  compress+base64 code into a "#…" hash
//	ari.decodeState(hash)    → string  reverse of encodeState
//
// # Compile
//
//	GOOS=js GOARCH=wasm go build -o wasm/ari.wasm ./cmd/ari-wasm
//
// Then copy wasm_exec.js:
//
//	cp $(go env GOROOT)/lib/wasm/wasm_exec.js wasm/
//
// Serve wasm/ over HTTP to use, e.g.:
//
//	cd wasm && python3 -m http.server 8080
//
// Note: SQL (SQLite/CGo) is excluded — it doesn't apply in a browser context.
// The math, base64, zip, http, and ratelimit extensions are
// included via ari.New(ari.DefaultOptions()).
package main

import (
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"runtime/debug"
	"strings"
	"syscall/js"

	goal "codeberg.org/anaseto/goal"
	arilib "github.com/semperos/ari"
	arihelp "github.com/semperos/ari/help"
)

// ariCtx is the persistent Goal context for the browser REPL session.
var ariCtx *goal.Context

func buildAriCtx() *goal.Context {
	ctx, err := arilib.New(arilib.DefaultOptions())
	if err != nil {
		panic(fmt.Sprintf("ari: failed to build context: %v", err))
	}
	return ctx
}

// jsEval evaluates Goal source code and returns the result as a string.
// The string includes any log output prepended to the result or error.
func jsEval(code string) string {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Caught panic: %v", r)
			debug.PrintStack()
		}
	}()
	var sb strings.Builder
	ariCtx.Log = &sb
	x, err := ariCtx.Eval(code)
	if err != nil {
		if e, ok := err.(*goal.Panic); ok {
			return sb.String() + e.ErrorStack()
		}
		return sb.String() + err.Error()
	}
	return sb.String() + x.Sprint(ariCtx, false)
}

// encodeState compresses code with zlib, base64-encodes it, and returns a
// string of the form "#<encoded>", suitable for use as window.location.hash.
func encodeState(code string) string {
	var sb strings.Builder
	sb.WriteByte('#')
	b64w := base64.NewEncoder(base64.RawURLEncoding, &sb)
	zw := zlib.NewWriter(b64w)
	fmt.Fprint(zw, code)
	zw.Flush()
	zw.Close()
	b64w.Close()
	return sb.String()
}

// decodeState reverses encodeState. Accepts a hash string with or without the
// leading "#". Returns the original Goal source, or "" on error.
func decodeState(hash string) string {
	if hash == "" || hash == "#" {
		return ""
	}
	if hash[0] == '#' {
		hash = hash[1:]
	}
	r := base64.NewDecoder(base64.RawURLEncoding, strings.NewReader(hash))
	zr, err := zlib.NewReader(r)
	if err != nil {
		log.Printf("decodeState: zlib reader: %v", err)
		return ""
	}
	defer func() {
		if err := zr.Close(); err != nil {
			log.Printf("decodeState: zlib close: %v", err)
		}
	}()
	var sb strings.Builder
	if _, err = io.Copy(&sb, zr); err != nil {
		log.Printf("decodeState: copy: %v", err)
		return ""
	}
	return sb.String()
}

func main() {
	ariCtx = buildAriCtx()
	helpFn := arihelp.HelpFunc()

	// Expose window.ari as a plain object with callable methods.
	ariAPI := js.Global().Get("Object").New()

	ariAPI.Set("eval", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 1 {
			return ""
		}
		return jsEval(args[0].String())
	}))

	ariAPI.Set("reset", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		ariCtx = buildAriCtx()
		return nil
	}))

	ariAPI.Set("help", js.FuncOf(func(_ js.Value, args []js.Value) any {
		token := ""
		if len(args) > 0 {
			token = args[0].String()
		}
		return helpFn(token)
	}))

	ariAPI.Set("version", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		return fmt.Sprintf("%s (wasm)", ariCtx.Version())
	}))

	ariAPI.Set("encodeState", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 1 {
			return ""
		}
		return encodeState(args[0].String())
	}))

	ariAPI.Set("decodeState", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 1 {
			return ""
		}
		return decodeState(args[0].String())
	}))

	js.Global().Set("ari", ariAPI)

	// Notify the page that the interpreter is ready. The page should define
	// window.ariOnReady before loading the WASM binary.
	if onReady := js.Global().Get("ariOnReady"); onReady.Type() == js.TypeFunction {
		onReady.Invoke()
	}

	<-make(chan bool) // keep goroutine alive
}
