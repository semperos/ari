//go:build plan9

/*
 * Copyright (c) 2021 Marcin Gasperowicz <xnooga@gmail.com>
 * SPDX-License-Identifier: MIT
 */

package lgcli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nooga/let-go/pkg/compiler"
	"github.com/nooga/let-go/pkg/vm"
)

// repl is a minimal line-by-line REPL for plan9. The alimpfard/line readline
// library depends on termios/ioctl (not available on plan9), so this build
// reads from stdin via bufio.Scanner — no completion, no syntax highlighting,
// no in-line editing. Use rio's edit-line history instead.
func repl(ctx *compiler.Context) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	prompt := ctx.CurrentNS().Name() + "=> "
	for {
		fmt.Print(prompt)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil && err != io.EOF {
				fmt.Println("prompt failed:", err)
			}
			fmt.Println()
			return
		}
		in := strings.TrimRight(scanner.Text(), "\r\n")
		if in == "" {
			continue
		}
		ctx.SetSource("REPL")
		val, err := runForm(ctx, in)
		if err != nil {
			fmt.Print(vm.FormatError(err))
		} else {
			fmt.Println(val.String())
		}
		prompt = ctx.CurrentNS().Name() + "=> "
	}
}
