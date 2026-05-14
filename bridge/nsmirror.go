package bridge

import (
	"fmt"
	"strings"

	"github.com/nooga/let-go/pkg/rt"
	"github.com/nooga/let-go/pkg/vm"
)

// LoadGoalNamespace evaluates Goal source under the given let-go ns name
// and mirrors the resulting Goal globals into a let-go namespace.
//
// Steps:
//
//  1. Snapshot the set of currently-defined Goal globals.
//  2. EvalPackage the source under prefix nsName so new globals land as
//     "nsName.<short>".
//  3. For every freshly-defined global, install a corresponding let-go
//     var on rt.DefNSBare(nsName), mapping
//
//       Goal "nsName.foo"  ->  let-go (nsName/foo)
//
//     Function-typed Goal globals are wrapped via wrapGoalFn so calling
//     (nsName/foo a b) Just Works.
//
// The same nsName is used as both the Goal-package prefix and the let-go
// namespace name; callers building the higher-level "goal." resolver
// (resolver.go) pass the *short* name as nsName so users see
// (require '[goal.analytics :as an]) and (an/summarize ...).
func (b *Bridge) LoadGoalNamespace(nsName, src, sourceLoc string) (*vm.Namespace, error) {
	if nsName == "" {
		return nil, fmt.Errorf("bridge: empty namespace name")
	}
	if sourceLoc == "" {
		sourceLoc = "<bridge:" + nsName + ">"
	}

	before := globalSet(b.Goal.GlobalNames(nil))

	if _, err := b.Goal.EvalPackage(src, sourceLoc, nsName); err != nil {
		return nil, fmt.Errorf("bridge: loading %q: %w", nsName, err)
	}

	ns := rt.DefNSBare(nsName)
	dotPrefix := nsName + "."
	for _, gname := range b.Goal.GlobalNames(nil) {
		if before[gname] {
			continue
		}
		short := strings.TrimPrefix(gname, dotPrefix)
		if short == gname {
			// Global appeared but isn't in our prefix — skip.
			continue
		}
		gv, ok := b.Goal.GetGlobal(gname)
		if !ok {
			continue
		}
		var lv vm.Value
		if gv.IsFunction() {
			lv = b.wrapGoalFn(gname, gv)
		} else {
			lv = b.GoalToLG(gv)
		}
		ns.Def(short, lv)
	}
	return ns, nil
}

func globalSet(names []string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}
