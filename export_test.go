package ari

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// Relies on unexported noHelpString.
func TestGoalKeywordsHaveHelp(t *testing.T) {
	tests := map[string]struct {
		kw string
	}{
		"help": {
			kw: "help",
		},
	}
	for name, test := range tests {
		test := test
		ctx, err := NewContext("")
		goalCtx := ctx.GoalContext
		if err != nil {
			t.Fatalf("error creating ari Context: %v", err)
		}
		t.Run(name, func(t *testing.T) {
			// Capture os.Stdout for testing
			osStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Prints
			goalV, err := goalCtx.Eval(fmt.Sprintf("help\"%v\"", test.kw))

			outC := make(chan string)
			errC := make(chan error)
			go func() {
				var buf bytes.Buffer
				_, pipeErr := io.Copy(&buf, r)
				if pipeErr != nil {
					errC <- pipeErr
				}
				outC <- buf.String()
			}()
			w.Close()
			os.Stdout = osStdout
			select {
			case pipeErr := <-errC:
				t.Fatalf("Failed to read from Pipe buffer while testing: %v", pipeErr)
			case out := <-outC:
				// err from Goal evaluation
				if err != nil {
					t.Fatalf("Context.GoalContext.Eval(%q) should return a string, but instead returned an error: %v",
						test.kw,
						goalV.Sprint(goalCtx, false))
				}
				if strings.TrimSpace(out) == noHelpString {
					t.Fatalf("Context.GoalContext.Eval(%q) should return a unique help string, but instead returned the default %q",
						test.kw,
						noHelpString,
					)
				}
			}
		})
	}
}
