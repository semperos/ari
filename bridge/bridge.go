// Package bridge wires a let-go (Clojure-flavored Lisp) runtime to an ari
// (Goal) context so that let-go programs can reference Goal namespaces
// transparently.
//
// A Bridge owns both runtimes:
//
//   - *goal.Context — the ari/Goal interpreter (with ari extensions and
//     embedded sources installed by package ari).
//   - *compiler.Context (+ *vm.Consts + user namespace) — the let-go
//     compiler driving the let-go VM.
//
// On construction the bridge installs a custom rt.NSLoader (see
// resolver.go) that intercepts namespace names beginning with the
// configured GoalPrefix (default "goal."): the underlying .goal source is
// found on disk roots or in an embedded FS, evaluated into the Goal
// context, and the resulting Goal globals are mirrored into a let-go
// namespace of the same name. All other namespace names fall through to
// let-go's standard resolver.NSResolver, which handles .lg / .cljc /
// embedded core sources.
package bridge

import (
	"fmt"
	"io/fs"

	goal "codeberg.org/anaseto/goal"

	"github.com/nooga/let-go/pkg/compiler"
	"github.com/nooga/let-go/pkg/resolver"
	"github.com/nooga/let-go/pkg/rt"
	"github.com/nooga/let-go/pkg/vm"
)

// Options configures bridge construction.
type Options struct {
	// Goal is the prepared Goal context. Required. Typically built via
	// ari.New(ari.DefaultOptions()).
	Goal *goal.Context

	// LetGoNS is the initial let-go namespace name. Defaults to "user".
	LetGoNS string

	// LetGoSearchPaths are filesystem roots searched by let-go's
	// standard NSResolver for .lg / .cljc sources. Defaults to ["."].
	LetGoSearchPaths []string

	// GoalPrefix is the let-go namespace prefix that triggers the
	// Goal-aware loader. Defaults to "goal.". Example: requiring
	// [goal.analytics :as an] sources analytics.goal.
	GoalPrefix string

	// GoalRoots are filesystem roots searched for .goal sources after
	// GoalEmbed is consulted. Defaults to ["."].
	GoalRoots []string

	// GoalEmbed is an optional embedded fs.FS searched before GoalRoots
	// for .goal sources. Typical use:
	//
	//   //go:embed scripts/*.goal
	//   var scripts embed.FS
	//   bridge.New(bridge.Options{ GoalEmbed: scripts, ... })
	GoalEmbed fs.FS

	// Debug enables let-go's debug compiler.
	Debug bool
}

// Bridge owns both runtimes and the integration plumbing between them.
type Bridge struct {
	// Goal is the Goal interpreter context. Safe to use directly.
	Goal *goal.Context

	// Compiler is the let-go compiler context. Exposed so vendored CLI
	// code (cmd/ari/clojure) can drive REPL / compile / bundle modes
	// against the same runtime state the bridge has set up.
	Compiler *compiler.Context

	// Consts is the let-go constant pool shared with Compiler.
	Consts *vm.Consts

	// LGResolver is let-go's standard namespace resolver (handles
	// .lg / .cljc / embedded core sources). Kept on the struct so the
	// CLI can serialize its LoadedChunks during bytecode bundling.
	LGResolver *resolver.NSResolver

	// Loader is the chained loader we install via rt.SetNSLoader.
	// It dispatches Goal-prefixed namespaces to LoadGoalNamespace and
	// delegates everything else to LGResolver.
	Loader *GoalNSLoader

	goalPrefix string
}

// New builds a Bridge from opts. It installs the chained NSLoader globally
// (rt.SetNSLoader); only one bridge per process is meaningful.
func New(opts Options) (*Bridge, error) {
	if opts.Goal == nil {
		return nil, fmt.Errorf("bridge: Options.Goal is required")
	}
	if opts.LetGoNS == "" {
		opts.LetGoNS = "user"
	}
	if opts.GoalPrefix == "" {
		opts.GoalPrefix = "goal."
	}
	if len(opts.LetGoSearchPaths) == 0 {
		opts.LetGoSearchPaths = []string{"."}
	}
	if len(opts.GoalRoots) == 0 {
		opts.GoalRoots = []string{"."}
	}

	// Build the let-go compiler stack. Mirrors lg.go's initCompiler so
	// that the same *compiler.Context can later be handed to the
	// vendored lgcli.Run for REPL / compile / bundle / wasm modes.
	consts := vm.NewConsts()
	ns := rt.NS(opts.LetGoNS)
	if ns == nil {
		return nil, fmt.Errorf("bridge: cannot resolve let-go ns %q", opts.LetGoNS)
	}
	var cctx *compiler.Context
	if opts.Debug {
		cctx = compiler.NewDebugCompiler(consts, ns)
	} else {
		cctx = compiler.NewCompiler(consts, ns)
	}

	// let-go's standard resolver — handles .lg/.cljc and embedded
	// core/string/set/... namespaces.
	lgRes := resolver.NewNSResolver(cctx, opts.LetGoSearchPaths)

	b := &Bridge{
		Goal:       opts.Goal,
		Compiler:   cctx,
		Consts:     consts,
		LGResolver: lgRes,
		goalPrefix: opts.GoalPrefix,
	}
	b.Loader = &GoalNSLoader{
		bridge:   b,
		inner:    lgRes,
		prefix:   opts.GoalPrefix,
		fsRoots:  opts.GoalRoots,
		embedded: opts.GoalEmbed,
	}
	rt.SetNSLoader(b.Loader)

	return b, nil
}

// GoalPrefix returns the configured Goal-namespace prefix (e.g. "goal.").
func (b *Bridge) GoalPrefix() string { return b.goalPrefix }
