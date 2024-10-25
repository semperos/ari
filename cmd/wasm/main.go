//go:build js && wasm

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

	"codeberg.org/anaseto/goal"
	"github.com/semperos/ari"
)

func getEltById(id string) js.Value {
	return js.Global().Get("document").Call("getElementById", id)
}

func evalTextArea() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Caught panic: %v\nStack Trace:\n", r)
			debug.PrintStack()
			out := getEltById("out")
			out.Set("value", fmt.Sprintf("Caught panic: %v\n", r))
		}
	}()
	in := getEltById("in").Get("value").String()
	out := getEltById("out")
	ariCtx, err := ari.NewUniversalContext()
	if err != nil {
		log.Printf(err.Error())
	}
	ctx := ariCtx.GoalContext
	var sb strings.Builder
	ctx.Log = &sb
	x, err := ctx.Eval(in)
	if err != nil {
		if e, ok := err.(*goal.Panic); ok {
			out.Set("value", sb.String()+e.ErrorStack())
		} else {
			out.Set("value", sb.String()+err.Error())
		}
	} else {
		out.Set("value", sb.String()+x.Sprint(ctx, false))
	}
	updateHash()
}

func updateHash() {
	var sb strings.Builder
	in := getEltById("in").Get("value").String()
	b64w := base64.NewEncoder(base64.URLEncoding, &sb)
	zw := zlib.NewWriter(b64w)
	sb.WriteByte('#')
	fmt.Fprint(zw, in)
	zw.Flush()
	zw.Close()
	b64w.Close()
	location := js.Global().Get("window").Get("location")
	location.Set("hash", sb.String())
}

func updateTextArea() {
	hash := js.Global().Get("window").Get("location").Get("hash").String()
	in := getEltById("in")
	if hash == "" {
		in.Set("value", "")
		return
	}
	r := base64.NewDecoder(base64.URLEncoding, strings.NewReader(hash[1:]))
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
	_, err = io.Copy(&sb, zr)
	if err != nil {
		log.Printf("decoding hash: %v", err)
		log.Printf("hash: %q", hash)
		return
	}
	log.Print(sb.String())
	in.Set("value", sb.String())
}

func main() {
	updateTextArea()
	eval := getEltById("eval")
	evalFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
		evalTextArea()
		return nil
	})
	eval.Call("addEventListener", "click", evalFunc)
	textIn := getEltById("in")
	textInFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		key := e.Get("key").String()
		if e.Get("ctrlKey").Bool() && key == "Enter" {
			e.Call("preventDefault")
			evalTextArea()
		} else if key == "F1" {
			out := getEltById("out")
			out.Set("value", helpString)
		}
		return nil
	})
	textIn.Call("addEventListener", "keydown", textInFunc)
	link := getEltById("link")
	linkFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
		updateHash()
		href := js.Global().Get("window").Get("location").Get("href")
		js.Global().Get("navigator").Get("clipboard").Call("writeText", href)
		return nil
	})
	link.Call("addEventListener", "click", linkFunc)
	help := getEltById("help")
	helpFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
		out := getEltById("out")
		out.Set("value", helpString)
		return nil
	})
	help.Call("addEventListener", "click", helpFunc)
	hashFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
		updateTextArea()
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "hashchange", hashFunc)
	ariCtx, err := ari.NewUniversalContext()
	if err != nil {
		log.Printf(err.Error())
	}
	ctx := ariCtx.GoalContext
	version := getEltById("goalVersion")
	version.Set("textContent", fmt.Sprintf("goal %s (wasm version)", ctx.Version()))
	wait := make(chan bool)
	<-wait
}
