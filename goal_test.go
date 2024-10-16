package ari_test

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"
	"testing"

	"codeberg.org/anaseto/goal"
	"github.com/jarcoal/httpmock"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/semperos/ari"
)

func TestGoalOk(t *testing.T) {
	// t.Parallel() // go test reports data race
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
			result: strings.TrimSpace(`
(1 2
 3 4)`),
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
			// t.Parallel() // go test reports data race
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

func TestGoalError(t *testing.T) {
	// t.Parallel() // go test reports data race
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
			// t.Parallel() // go test reports data race
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

// Adapted from Goal implementation.
type matchTest struct {
	Fname    string
	Line     int
	Left     string
	Right    string
	IsScript bool
}

// Adapted from Goal implementation.
func getMatchTests(glob string) ([]matchTest, error) {
	d := os.DirFS("testing/via-go/")
	fnames, err := fs.Glob(d, glob)
	if err != nil {
		return nil, err
	}
	mts := []matchTest{}
	for _, fname := range fnames {
		bs, err := fs.ReadFile(d, fname)
		if err != nil {
			return nil, err
		}
		text := string(bs)
		lines := strings.Split(text, "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) == 0 || line[0] == '/' {
				continue
			}
			left, right, found := strings.Cut(line, " /")
			if !found {
				log.Printf("%s:%d: bad line", fname, i+1)
				continue
			}
			mts = append(mts, matchTest{
				Fname:    fname,
				Line:     i + 1,
				Left:     strings.TrimSpace(left),
				Right:    strings.TrimSpace(right),
				IsScript: false,
			})
		}
	}
	return mts, nil
}

// Adapted from Goal implementation.
func getScriptMatchTests(glob string) ([]matchTest, error) {
	d := os.DirFS("testing/scripts")
	fnames, err := fs.Glob(d, glob)
	if err != nil {
		return nil, err
	}
	mts := []matchTest{}
	for _, fname := range fnames {
		bs, err := fs.ReadFile(d, fname)
		if err != nil {
			return nil, err
		}
		text := string(bs)
		body := strings.SplitN(text, "\n/RESULT:\n", 2)
		if len(body) != 2 {
			log.Printf("%s: bad script", fname)
			continue
		}
		left := body[0]
		right := body[1]
		mts = append(mts, matchTest{
			Fname:    fname,
			Left:     strings.TrimSpace(left),
			Right:    strings.TrimSpace(right),
			IsScript: true,
		})
	}
	return mts, nil
}

// Adapted from Goal implementation.
//
//nolint:gocognit // upstream
func TestEval(t *testing.T) {
	mts, err := getMatchTests("*.goal")
	if err != nil {
		t.Fatalf("getMatchTests: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	for _, mt := range mts {
		if mt.Fname == "errors.goal" {
			continue
		}
		mt := mt
		name := fmt.Sprintf("%s:%d", mt.Fname, mt.Line)
		matchString := fmt.Sprintf("(%s) ~ (%s)", mt.Left, mt.Right)
		t.Run(name, func(t *testing.T) {
			ariContextLeft, err := ari.NewContext("")
			if err != nil {
				t.Fatalf("ari context error: %v", err)
			}
			ariContextRight, err := ari.NewContext("")
			if err != nil {
				t.Fatalf("ari context error: %v", err)
			}

			// HTTP mocking
			httpClient, err := ari.NewHTTPClient(goalNewDictEmpty())
			if err != nil {
				t.Fatalf("failed to create HTTP client for testing: %v", err)
			}
			ariContextLeft.HTTPClient = httpClient
			ariContextRight.HTTPClient = httpClient
			httpmock.ActivateNonDefault(ariContextLeft.HTTPClient.Client.GetClient())
			httpmock.ActivateNonDefault(ariContextRight.HTTPClient.Client.GetClient())
			defer httpmock.DeactivateAndReset()
			registerHTTPMocks()

			err = os.Chdir(cwd + "/testing/via-go/")
			if err != nil {
				t.Fatalf("failed to chdir to 'testing/via-go': %v", err)
			}
			err = ariContextLeft.GoalContext.Compile(mt.Left, "", "")
			ps := ariContextLeft.GoalContext.String()
			if err != nil {
				t.Log(ps)
				t.Log(matchString)
				t.Fatalf("compile error: %v", err)
			}
			vLeft, errLeft := ariContextLeft.GoalContext.Run()
			vRight, errRight := ariContextRight.GoalContext.Eval(mt.Right)
			if errLeft != nil || errRight != nil {
				t.Log(ps)
				t.Log(matchString)
				t.Fatalf("return error: `%v` vs `%v`", errLeft, errRight)
			}
			if !vLeft.Matches(vRight) {
				t.Log(ps)
				t.Log(matchString)
				if vLeft != (goal.NewGap()) {
					//nolint:lll // upstream
					t.Logf("results:\n   %s\nvs %s\n", vLeft.Sprint(ariContextLeft.GoalContext, true), vRight.Sprint(ariContextRight.GoalContext, true))
				} else {
					t.Logf("results:\n   %v\nvs %s\n", vLeft, vRight.Sprint(ariContextRight.GoalContext, true))
				}
				t.FailNow()
			}
		})
	}
}

func registerHTTPMocks() {
	httpmock.RegisterResponder("GET", "https://example.com/api/sprockets",
		httpmock.NewStringResponder(200, `[{"id": 1, "name": "Test Sprocket 1"},{"id": 2, "name": "Test Sprocket 2"}]`))
	httpmock.RegisterResponder("GET", "https://example.com/api/sprockets/1",
		httpmock.NewStringResponder(200, `{"id": 1, "name": "Test Sprocket 1"}`))
}

func goalNewDictEmpty() *goal.D {
	dv := goal.NewD(goal.NewAV(nil), goal.NewAV(nil))
	d, ok := dv.BV().(*goal.D)
	if !ok {
		panic("Developer error: Empty Goal dictionary expected.")
	}
	return d
}

// Adapted from Goal implementation.
//
//nolint:gocognit // upstream
func TestErrors(t *testing.T) {
	mts, err := getMatchTests("errors.goal")
	if err != nil {
		t.Fatalf("getMatchTests: %v", err)
	}
	smts, err := getScriptMatchTests("errors.goal")
	if err != nil {
		t.Fatalf("getScriptMatchTests: %v", err)
	}
	for _, mt := range append(mts, smts...) {
		mt := mt
		name := fmt.Sprintf("%s:%d", mt.Fname, mt.Line)
		matchString := mt.Left
		t.Run(name, func(t *testing.T) {
			ariContext, err := ari.NewContext("")
			if err != nil {
				t.Fatalf("ari context error: %v", err)
			}
			err = ariContext.GoalContext.Compile(mt.Left, "", "")
			ps := ariContext.GoalContext.String()
			if err == nil {
				var v goal.V
				v, err = ariContext.GoalContext.Run()
				if err == nil {
					t.Log(ps)
					t.Log(matchString)
					t.Fatalf("no error left: result: %s\nexpected: %v", v.Sprint(ariContext.GoalContext, true), mt.Right)
				}
			}
			//nolint:errorlint // upstream
			e, ok := err.(*goal.Panic)
			if !ok {
				// should never happen
				t.Log(ps)
				t.Log(matchString)
				t.Fatalf("bad error: `%v`\nexpected:`%v`", err, mt.Right)
			}
			msg := e.Error()
			if strings.Contains(mt.Left, "\n") {
				msg = e.ErrorStack()
			}
			if !strings.Contains(e.Error(), mt.Right) {
				t.Log(ps)
				t.Log(matchString)
				t.Logf("\n   error: %q\nexpected: %q", msg, mt.Right)
				t.Fail()
				return
			}
		})
	}
}
