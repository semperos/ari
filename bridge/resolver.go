package bridge

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nooga/let-go/pkg/resolver"
	"github.com/nooga/let-go/pkg/vm"
)

// GoalNSLoader implements let-go's rt.NSLoader. It chains two strategies:
//
//  1. If the requested namespace name begins with the configured Goal
//     prefix (default "goal."), search the configured embedded FS and
//     then filesystem roots for a matching ".goal" file. On match, the
//     file is evaluated into the bridge's Goal context under that
//     namespace, and the new Goal globals are mirrored into a let-go
//     namespace via Bridge.LoadGoalNamespace.
//
//  2. Otherwise (or if the Goal-prefixed lookup fails), delegate to
//     let-go's standard resolver.NSResolver, which handles .lg / .cljc
//     files and embedded core/* namespaces.
//
// This is the option-3 abstraction described in the architecture notes:
// the user writes
//
//	(ns my.app (:require [goal.analytics :as an]))
//	(an/summarize [1 2 3 4 5])
//
// and the first reference to goal.analytics triggers a one-shot mirror
// of analytics.goal into a let-go namespace of the same long name, with
// :as 'an' providing the short alias in the current ns (let-go's reader
// honours that automatically once the long namespace exists).
type GoalNSLoader struct {
	bridge   *Bridge
	inner    *resolver.NSResolver
	prefix   string
	fsRoots  []string
	embedded fs.FS

	// loading guards against re-entrant loads of the same Goal ns.
	loading map[string]bool
}

// Load implements rt.NSLoader.
func (g *GoalNSLoader) Load(name string) *vm.Namespace {
	if strings.HasPrefix(name, g.prefix) {
		if ns := g.loadGoal(name); ns != nil {
			return ns
		}
		// Fall through to inner resolver on miss so .lg files named
		// like goal_foo.lg can still resolve.
	}
	return g.inner.Load(name)
}

// loadGoal looks up the ".goal" source for name (after stripping the
// Goal prefix), feeds it through the bridge, and returns the resulting
// let-go namespace.
func (g *GoalNSLoader) loadGoal(name string) *vm.Namespace {
	if g.loading == nil {
		g.loading = make(map[string]bool)
	}
	if g.loading[name] {
		return nil
	}
	stem := strings.TrimPrefix(name, g.prefix)
	if stem == "" {
		return nil
	}

	src, loc, ok := g.findGoalSource(stem)
	if !ok {
		return nil
	}

	g.loading[name] = true
	defer delete(g.loading, name)

	ns, err := g.bridge.LoadGoalNamespace(name, src, loc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ari/bridge: load %s: %v\n", name, err)
		return nil
	}
	return ns
}

// findGoalSource tries embedded FS first, then filesystem roots.
//
// For a stem "analytics" the candidate filenames are "analytics.goal".
// For a dotted stem like "data.analytics" the candidates are
// "data/analytics.goal" (preferred) and "data.analytics.goal".
//
// Hyphen-to-underscore variants are also tried, matching let-go's own
// resolver behaviour.
func (g *GoalNSLoader) findGoalSource(stem string) (src, loc string, ok bool) {
	for _, cand := range goalCandidates(stem) {
		if g.embedded != nil {
			if data, err := fs.ReadFile(g.embedded, cand); err == nil {
				return string(data), "embed:" + cand, true
			}
		}
		for _, root := range g.fsRoots {
			p := filepath.Join(root, filepath.FromSlash(cand))
			if data, err := os.ReadFile(p); err == nil {
				return string(data), p, true
			}
		}
	}
	return "", "", false
}

func goalCandidates(stem string) []string {
	parts := strings.Split(stem, ".")
	hyphen := strings.Join(parts, "/")
	for i, p := range parts {
		parts[i] = strings.ReplaceAll(p, "-", "_")
	}
	under := strings.Join(parts, "/")
	cands := []string{hyphen + ".goal"}
	if under != hyphen {
		cands = append(cands, under+".goal")
	}
	// Also try the flat-dotted form ("data.analytics.goal") as a
	// fallback for the rare case where someone literally named a file
	// that way.
	cands = append(cands, stem+".goal")
	return cands
}
