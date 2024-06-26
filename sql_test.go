package ari_test

import (
	"strings"
	"testing"

	"github.com/semperos/ari"
)

func TestSQLOk(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input  string
		result string
	}{
		"simple goal expression": {
			input:  "+/!10",
			result: "45",
		},
		"multiline goal expression": {
			input: `(1 2
			3 4)`,
			result: strings.Trim(`
(1 2
 3 4)`, "\n"),
		},
		"goal json, marshal": {
			input:  `""json -0w`,
			result: `"false"`,
		},
		"goal json, unmarshal": {
			input:  `json "false"`,
			result: `-0w`,
		},
		"goal os.env": {
			input:  `k:"ARI_TEST";k env $1234;env k`,
			result: `"1234"`,
		},
		"ari keyword is defined": {
			input:  "http.post",
			result: "http.post",
		},
	}

	for name, test := range tests {
		test := test
		ctx, err := ari.NewContext("")
		goalCtx := ctx.GoalContext
		if err != nil {
			t.Fatalf("error creating ari Context: %v", err)
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			goalV, err := goalCtx.Eval(test.input)
			if err != nil {
				t.Fatalf("Context.GoalContext.Eval(%q) returned an error: %v", test.input, err)
			}
			goalVString := goalV.Sprint(goalCtx, false)
			if got, expected := goalVString, test.result; got != expected {
				t.Fatalf("Context.GoalContext.Eval(%q) returned %q; expected %q", test.input, got, expected)
			}
		})
	}
}

func TestSQLError(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input  string
		errMsg string
	}{
		"undefined is an error": {
			input:  "http.wacky",
			errMsg: "undefined global: http.wacky",
		},
	}
	for name, test := range tests {
		test := test
		ctx, err := ari.NewContext("")
		goalCtx := ctx.GoalContext
		if err != nil {
			t.Fatalf("error creating ari Context: %v", err)
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			goalV, err := goalCtx.Eval(test.input)
			if err == nil {
				t.Fatalf("Context.GoalContext.Eval(%q) should return an error, but instead returned: %v",
					test.input,
					goalV.Sprint(goalCtx, false))
			}
			// goalVString := goalV.Sprint(goalCtx, false)
			if got, expected := err.Error(), test.errMsg; got != expected {
				t.Fatalf("Context.GoalContext.Eval(%q) should return an error like %q, but instead returned one like %q",
					test.input,
					expected,
					got)
			}
		})
	}
}
