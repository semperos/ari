package bridge_test

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/nooga/let-go/pkg/compiler"
	"github.com/nooga/let-go/pkg/vm"

	"github.com/semperos/ari"
	"github.com/semperos/ari/bridge"
)

// fakeFS builds an fs.FS from a literal map of filename → contents.
func fakeFS(files map[string]string) fs.FS {
	out := fstest.MapFS{}
	for k, v := range files {
		out[k] = &fstest.MapFile{Data: []byte(v)}
	}
	return out
}

// runLG compiles & runs a let-go source snippet against b.Compiler and
// returns the final value of the top-level form.
func runLG(t *testing.T, c *compiler.Context, src string) vm.Value {
	t.Helper()
	_, val, err := c.CompileMultiple(strings.NewReader(src))
	if err != nil {
		t.Fatalf("compile %q: %v", src, err)
	}
	return val
}

func newBridge(t *testing.T, files map[string]string) *bridge.Bridge {
	t.Helper()
	gctx, err := ari.New(ari.DefaultOptions())
	if err != nil {
		t.Fatalf("ari.New: %v", err)
	}
	b, err := bridge.New(bridge.Options{
		Goal:      gctx,
		GoalEmbed: fakeFS(files),
	})
	if err != nil {
		t.Fatalf("bridge.New: %v", err)
	}
	return b
}

func TestDirectLoadGoalNamespace(t *testing.T) {
	b := newBridge(t, nil)
	_, err := b.LoadGoalNamespace("goal.demo", `summarize:{(+/x)%#x}`, "<inline>")
	if err != nil {
		t.Fatalf("LoadGoalNamespace: %v", err)
	}
	// Now call (goal.demo/summarize [1 2 3 4 5]) from let-go.
	// Goal's `%` is float division, so the mean of 1..5 is 3.0.
	v := runLG(t, b.Compiler, `(goal.demo/summarize [1 2 3 4 5])`)
	if got, want := v, vm.Float(3.0); got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestResolverLoadsGoalFile(t *testing.T) {
	b := newBridge(t, map[string]string{
		"analytics.goal": `
sum:{+/x}
mean:{(+/x)%#x}
tag:"goal-says-hello"
`,
	})
	// Reference a goal.* namespace from let-go — the loader should fire.
	// We use the symbol form of require because let-go's runtime require
	// has a known issue with the vector form (case *vm.ArrayVector vs
	// value-type vm.ArrayVector). The ns macro form
	//   (ns x (:require [goal.analytics :as an]))
	// is handled by the reader/compiler and does work; see
	// TestNamespaceFormRequireAlias below.
	_ = runLG(t, b.Compiler, `(require 'goal.analytics)`)

	if got := runLG(t, b.Compiler, `(goal.analytics/sum [10 20 30])`); got != vm.Int(60) {
		t.Errorf("sum: got %v want 60", got)
	}
	if got := runLG(t, b.Compiler, `(goal.analytics/mean [1 2 3 4])`); got != vm.Float(2.5) {
		t.Errorf("mean: got %v want 2.5", got)
	}
	if got := runLG(t, b.Compiler, `goal.analytics/tag`); got != vm.String("goal-says-hello") {
		t.Errorf("tag: got %v", got)
	}
}

func TestNamespaceFormRequireAlias(t *testing.T) {
	b := newBridge(t, map[string]string{
		"shop.goal": `
total:{+/x}
`,
	})
	_ = runLG(t, b.Compiler, `(ns customer.app (:require [goal.shop :as s]))`)
	if got := runLG(t, b.Compiler, `(s/total [3 4 5])`); got != vm.Int(12) {
		t.Errorf("s/total via :as alias: got %v want 12", got)
	}
}

// TestArgOrderPreserved guards against a subtle bug: goal.V.Apply expects
// args in stack order (right-to-left), but callers naturally pass them
// left-to-right. The bridge must reverse them before Apply. A non-symmetric
// function exposes mis-ordering.
func TestArgOrderPreserved(t *testing.T) {
	b := newBridge(t, map[string]string{
		"order.goal": `
minus:{[a;b] a-b}
getA:{[a;b] a}
getB:{[a;b] b}
`,
	})
	_ = runLG(t, b.Compiler, `(require 'goal.order)`)
	if got := runLG(t, b.Compiler, `(goal.order/minus 10 3)`); got != vm.Int(7) {
		t.Errorf("minus 10 3: got %v want 7 (args swapped?)", got)
	}
	if got := runLG(t, b.Compiler, `(goal.order/getA 1 2)`); got != vm.Int(1) {
		t.Errorf("getA 1 2: got %v want 1", got)
	}
	if got := runLG(t, b.Compiler, `(goal.order/getB 1 2)`); got != vm.Int(2) {
		t.Errorf("getB 1 2: got %v want 2", got)
	}
}

func TestNonGoalNamespaceDelegates(t *testing.T) {
	// "string" is one of let-go's embedded stdlib namespaces.
	b := newBridge(t, nil)
	v := runLG(t, b.Compiler, `(do (require 'string) (string/upper-case "hi"))`)
	if v != vm.String("HI") {
		t.Fatalf("string/upper-case: got %v", v)
	}
}
