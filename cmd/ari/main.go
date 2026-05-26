//go:build !js || !wasm

// ari: a Goal interpreter with extensions for SQL, HTTP, and GUI (Fyne).
package main

import (
	"fmt"
	"os"

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
	goalcmd.Exit(goalcmd.Run(ctx, goalcmd.Config{
		ProgramName: "ari",
		Help:        arihelp.HelpFunc(),
	}))
}
