//go:build !js || !wasm

// ari: a Goal interpreter with extensions for SQL, HTTP, and GUI (Fyne).
package main

import (
	"fmt"
	"os"
	"strings"

	goal "codeberg.org/anaseto/goal"
	goalcmd "codeberg.org/anaseto/goal/cmd"
	"github.com/semperos/ari"
	arihelp "github.com/semperos/ari/help"
)

func main() {
	ctx, err := ari.New(ari.FullOptions())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ari: %v\n", err)
		os.Exit(1)
	}
	helpFn := arihelp.HelpFunc()
	ctx.RegisterMonad("helps", func(_ *goal.Context, args []goal.V) goal.V {
		if len(args) < 1 {
			return goal.NewS(strings.TrimSpace(helpFn("")))
		}
		arg, ok := args[0].BV().(goal.S)
		if !ok {
			return goal.Panicf("helps x : x must be a string, got %q", args[0].Type())
		}
		return goal.NewS(strings.TrimSpace(helpFn(string(arg))))
	})
	goalcmd.Exit(goalcmd.Run(ctx, goalcmd.Config{
		ProgramName: "ari",
		Help:        helpFn,
	}))
}
