// Package lgcli is a vendored copy of the let-go (lg) command-line driver.
//
// SOURCE: github.com/nooga/let-go (MIT), files lg.go / lg_repl*.go /
// lg_ansi*.go / wasm.go. Original copyright:
//
//	Copyright (c) 2021-2026 Marcin Gasperowicz <xnooga@gmail.com>
//	SPDX-License-Identifier: MIT
//
// The MIT license requires that the above copyright notice and the
// permission notice appear in all copies or substantial portions of the
// software. The let-go LICENSE file is reproduced verbatim alongside this
// package as LICENSE.let-go.
//
// CHANGES FROM UPSTREAM:
//
//   - Package renamed from `main` to `lgcli`.
//   - `main()` extracted into `Run(args []string, opts Options) int` so the
//     CLI can be invoked as a subcommand of another binary (here, `ari
//     clojure`). Args and flags are parsed against a caller-supplied
//     *flag.FlagSet rather than the package-level default. Version and
//     commit metadata are passed via Options instead of build-time ldflags
//     on this package.
//   - The caller may supply a pre-built *compiler.Context and
//     *resolver.NSResolver so that, when run as `ari clojure`, the
//     resolver chain installed by the ari/bridge package (Goal-aware
//     namespace loader) is preserved. When the caller passes nil for
//     both, behaviour matches the upstream `lg` binary.
//
// All other code paths (compile to .lgb, bundle to standalone executable,
// build to WASM web app, nREPL server, REPL, -e expr, -r attach-REPL,
// source-paths, LGB autoloading from appended payload) are preserved from
// the upstream lg command unchanged in semantics.
package lgcli

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nooga/let-go/pkg/bytecode"
	"github.com/nooga/let-go/pkg/compiler"
	"github.com/nooga/let-go/pkg/nrepl"
	"github.com/nooga/let-go/pkg/resolver"
	"github.com/nooga/let-go/pkg/rt"
	"github.com/nooga/let-go/pkg/vm"
)

// Options configures a single invocation of Run.
type Options struct {
	// ProgramName is used in usage messages. Defaults to "lg".
	ProgramName string

	// Version / Commit are surfaced via -version and propagated to
	// rt.Version / rt.Commit so System/getProperty exposes them.
	Version string
	Commit  string

	// Compiler is an optional pre-built let-go compiler context. When
	// non-nil, Run uses it directly instead of constructing one. This
	// is how `ari clojure` injects a Compiler whose NSLoader has
	// already been wired to the Goal bridge.
	Compiler *compiler.Context

	// NSResolver is the let-go standard resolver used for bundling and
	// AOT compilation. When Compiler is non-nil, NSResolver must also
	// be non-nil and must be the resolver let-go uses for .lg/.cljc
	// imports. (If the caller's resolver is wrapped by another loader,
	// pass the inner *resolver.NSResolver here.)
	NSResolver *resolver.NSResolver
}

// Footer appended to standalone binaries: [lgb data][8-byte size][4-byte magic]
var bundleMagic = [4]byte{'L', 'G', 'B', 'X'}

// version is set by Run() from Options.Version. Used by wasm.go to
// decide between module-proxy and local-path build mode (preserving the
// original lg.go behaviour where this was a build-time ldflags var).
var version = "dev"

func versionString(version, commit string) string {
	if commit != "none" && commit != "" && len(commit) > 7 {
		return fmt.Sprintf("%s (%s)", version, commit[:7])
	}
	return version
}

func motd(version, commit string) {
	banner := "" +
		" " + ansiBold + " λ" + ansiReset + "   " + ansiBold + "let-go" + ansiReset + " %s\n" +
		" " + ansiBoldCyan + "GO" + ansiReset + "   " + ansiDim + bannerQuitHint + ansiReset + "\n"
	fmt.Printf(banner, versionString(version, commit))
}

func runForm(ctx *compiler.Context, in string) (vm.Value, error) {
	_, val, err := ctx.CompileMultiple(strings.NewReader(in))
	if err != nil {
		return nil, err
	}
	return val, err
}

func runFile(ctx *compiler.Context, filename string) error {
	ctx.SetSource(filename)
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	_, _, err = ctx.CompileMultiple(f)
	errc := f.Close()
	if err != nil {
		return err
	}
	if errc != nil {
		return errc
	}
	return nil
}

func runLGB(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	resolve := func(nsName, name string) *vm.Var {
		n := rt.DefNSBare(nsName)
		v := n.LookupLocal(vm.Symbol(name))
		if v == nil {
			return n.Def(name, vm.NIL)
		}
		return v
	}
	unit, err := bytecode.DecodeToExecUnit(bytes.NewReader(data), resolve)
	if err != nil {
		return fmt.Errorf("decoding %s: %w", filename, err)
	}
	if len(unit.NSOrder) > 0 {
		for _, name := range unit.NSOrder {
			chunk := unit.NSChunks[name]
			if chunk == nil || chunk == unit.MainChunk {
				continue
			}
			f := vm.NewFrame(chunk, nil)
			_, err := f.RunProtected()
			vm.ReleaseFrame(f)
			if err != nil {
				return fmt.Errorf("loading namespace %s: %w", name, err)
			}
		}
	}
	f := vm.NewFrame(unit.MainChunk, nil)
	_, err = f.RunProtected()
	vm.ReleaseFrame(f)
	return err
}

func checkBundledLGB() []byte {
	candidates := make([]string, 0, 3)
	if exe, err := os.Executable(); err == nil && exe != "" {
		candidates = append(candidates, exe)
	}
	if len(os.Args) > 0 && os.Args[0] != "" {
		candidates = append(candidates, os.Args[0])
	}
	candidates = append(candidates, "/proc/self/exe")
	seen := map[string]bool{}
	for _, path := range candidates {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		if data := readBundledLGB(path); data != nil {
			return data
		}
	}
	return nil
}

func readBundledLGB(path string) []byte {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	_, err = f.Seek(-12, io.SeekEnd)
	if err != nil {
		return nil
	}
	var footer [12]byte
	if _, err := io.ReadFull(f, footer[:]); err != nil {
		return nil
	}
	if footer[8] != bundleMagic[0] || footer[9] != bundleMagic[1] ||
		footer[10] != bundleMagic[2] || footer[11] != bundleMagic[3] {
		return nil
	}
	lgbSize := binary.LittleEndian.Uint64(footer[:8])
	_, err = f.Seek(-12-int64(lgbSize), io.SeekEnd)
	if err != nil {
		return nil
	}
	data := make([]byte, lgbSize)
	if _, err := io.ReadFull(f, data); err != nil {
		return nil
	}
	return data
}

func bundleBinary(ctx *compiler.Context, nsRes *resolver.NSResolver, src string, dst string, basePath string) error {
	ctx.SetSource(src)
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	chunk, _, err := ctx.CompileMultiple(f)
	f.Close()
	if err != nil {
		return err
	}
	var lgbBuf bytes.Buffer
	if len(nsRes.LoadedChunks) > 0 {
		mainNS := ctx.CurrentNS().Name()
		nsChunks := make(map[string]*vm.CodeChunk, len(nsRes.LoadedChunks)+1)
		for k, v := range nsRes.LoadedChunks {
			nsChunks[k] = v
		}
		nsChunks[mainNS] = chunk
		nsOrder := append(nsRes.LoadOrder, mainNS)
		if err := bytecode.EncodeBundleOrdered(&lgbBuf, ctx.Consts(), nsChunks, nsOrder); err != nil {
			return err
		}
	} else {
		if err := bytecode.EncodeCompilation(&lgbBuf, ctx.Consts(), chunk); err != nil {
			return err
		}
	}
	if basePath == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("finding executable: %w", err)
		}
		basePath = exe
	}
	srcBin, err := os.Open(basePath)
	if err != nil {
		return err
	}
	defer srcBin.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()
	binSize, err := getBaseBinarySize(srcBin)
	if err != nil {
		return err
	}
	srcBin.Seek(0, io.SeekStart)
	if _, err := io.CopyN(out, srcBin, binSize); err != nil {
		return err
	}
	lgbData := lgbBuf.Bytes()
	if _, err := out.Write(lgbData); err != nil {
		return err
	}
	var footer [12]byte
	binary.LittleEndian.PutUint64(footer[:8], uint64(len(lgbData)))
	copy(footer[8:], bundleMagic[:])
	if _, err := out.Write(footer[:]); err != nil {
		return err
	}
	return nil
}

func getBaseBinarySize(f *os.File) (int64, error) {
	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}
	total := fi.Size()
	if total < 12 {
		return total, nil
	}
	f.Seek(-12, io.SeekEnd)
	var footer [12]byte
	if _, err := io.ReadFull(f, footer[:]); err != nil {
		return total, nil
	}
	if footer[8] == bundleMagic[0] && footer[9] == bundleMagic[1] &&
		footer[10] == bundleMagic[2] && footer[11] == bundleMagic[3] {
		lgbSize := binary.LittleEndian.Uint64(footer[:8])
		return total - int64(lgbSize) - 12, nil
	}
	return total, nil
}

func compileLG(ctx *compiler.Context, nsRes *resolver.NSResolver, src string, dst string) error {
	ctx.SetSource(src)
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	chunk, _, err := ctx.CompileMultiple(f)
	f.Close()
	if err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if len(nsRes.LoadedChunks) > 0 {
		mainNS := ctx.CurrentNS().Name()
		nsChunks := make(map[string]*vm.CodeChunk, len(nsRes.LoadedChunks)+1)
		for k, v := range nsRes.LoadedChunks {
			nsChunks[k] = v
		}
		nsChunks[mainNS] = chunk
		nsOrder := append(nsRes.LoadOrder, mainNS)
		return bytecode.EncodeBundleOrdered(out, ctx.Consts(), nsChunks, nsOrder)
	}
	return bytecode.EncodeCompilation(out, ctx.Consts(), chunk)
}

var nreplServer *nrepl.NreplServer

func nreplServe(ctx *compiler.Context, port int) error {
	nreplServer = nrepl.NewNreplServer(ctx)
	return nreplServer.Start(port)
}

func defaultInitCompiler(debug bool) *compiler.Context {
	consts := vm.NewConsts()
	ns := rt.NS("user")
	if ns == nil {
		fmt.Fprintln(os.Stderr, "namespace not found")
		return nil
	}
	if debug {
		return compiler.NewDebugCompiler(consts, ns)
	}
	return compiler.NewCompiler(consts, ns)
}

// parsedFlags collects the values bound by registerFlags.
type parsedFlags struct {
	nreplPort     int
	runNREPL      bool
	runREPL       bool
	expr          string
	debug         bool
	showVersion   bool
	compileOutput string
	bundleOutput  string
	bundleBase    string
	wasmOutput    string
	sourcePaths   string
}

func registerFlags(fs *flag.FlagSet) *parsedFlags {
	p := &parsedFlags{}
	fs.BoolVar(&p.runREPL, "r", false, "attach REPL after running given files")
	fs.StringVar(&p.expr, "e", "", "eval given expression")
	fs.BoolVar(&p.debug, "d", false, "enable VM debug mode")
	fs.BoolVar(&p.runNREPL, "n", false, "enable nREPL server")
	fs.IntVar(&p.nreplPort, "p", 2137, "set nREPL port, default is 2137")
	fs.BoolVar(&p.showVersion, "v", false, "print version and exit")
	fs.BoolVar(&p.showVersion, "version", false, "print version and exit")
	fs.StringVar(&p.compileOutput, "c", "", "compile .lg file to .lgb bytecode (specify output path)")
	fs.StringVar(&p.bundleOutput, "b", "", "bundle .lg file into a standalone executable (specify output path)")
	fs.StringVar(&p.bundleBase, "bundle-base", "", "path to target-platform lg binary for cross-OS bundling (defaults to current executable)")
	fs.StringVar(&p.wasmOutput, "w", "", "build .lg file into a WASM web app (specify output directory)")
	fs.StringVar(&p.sourcePaths, "source-paths", "",
		"additional namespace search paths separated by the OS path-list separator "+
			"(':' on Unix, ';' on Windows). Falls back to LG_SOURCE_PATHS if unset.")
	return p
}

// buildSearchPaths resolves the resolver's path list from the -source-paths
// flag (preferred) or the LG_SOURCE_PATHS env var (fallback).
func buildSearchPaths(fs *flag.FlagSet, raw string) []string {
	explicitSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "source-paths" {
			explicitSet = true
		}
	})
	return resolver.PathsFromInputs(raw, os.Getenv("LG_SOURCE_PATHS"), explicitSet)
}

// Run is the entry point for the let-go CLI when invoked as a subcommand.
// It returns the process exit code; the caller is responsible for os.Exit.
func Run(args []string, opts Options) int {
	if opts.ProgramName == "" {
		opts.ProgramName = "lg"
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}
	if opts.Commit == "" {
		opts.Commit = "none"
	}
	rt.Version = opts.Version
	rt.Commit = opts.Commit
	// Vendoring shim: wasm.go reads a package-level `version` to decide
	// whether to require let-go from the module proxy or from a local
	// path. Populate it from opts to preserve upstream behaviour.
	version = opts.Version

	// Standalone-binary fast path: if our executable has an appended
	// LGB payload, run it without flag parsing. Mirrors upstream lg.
	if lgbData := checkBundledLGB(); lgbData != nil {
		ctx := opts.Compiler
		nsRes := opts.NSResolver
		if ctx == nil {
			ctx = defaultInitCompiler(false)
			if ctx == nil {
				return 1
			}
			nsRes = resolver.NewNSResolver(ctx, buildSearchPaths(flag.NewFlagSet("", flag.ContinueOnError), ""))
			rt.SetNSLoader(nsRes)
		}
		defer rt.ShutdownAllPods()
		resolve := func(nsName, name string) *vm.Var {
			n := rt.DefNSBare(nsName)
			v := n.LookupLocal(vm.Symbol(name))
			if v == nil {
				return n.Def(name, vm.NIL)
			}
			return v
		}
		unit, err := bytecode.DecodeToExecUnit(bytes.NewReader(lgbData), resolve)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		for _, name := range unit.NSOrder {
			chunk := unit.NSChunks[name]
			if chunk == nil || chunk == unit.MainChunk {
				continue
			}
			f := vm.NewFrame(chunk, nil)
			_, err := f.RunProtected()
			vm.ReleaseFrame(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: loading namespace %s: %v\n", name, err)
				return 1
			}
		}
		f := vm.NewFrame(unit.MainChunk, nil)
		_, err = f.RunProtected()
		vm.ReleaseFrame(f)
		if err != nil {
			fmt.Fprint(os.Stderr, vm.FormatError(err))
			return 1
		}
		_ = nsRes
		return 0
	}

	fs := flag.NewFlagSet(opts.ProgramName, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", opts.ProgramName)
		fs.PrintDefaults()
	}
	p := registerFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if p.showVersion {
		fmt.Printf("%s %s\n", opts.ProgramName, versionString(opts.Version, opts.Commit))
		return 0
	}

	files := fs.Args()
	defer rt.ShutdownAllPods()

	// Use injected compiler/resolver when supplied (bridge-aware
	// invocation) else build a vanilla pair.
	ctx := opts.Compiler
	nsResolver := opts.NSResolver
	if ctx == nil {
		ctx = defaultInitCompiler(p.debug)
		if ctx == nil {
			return 1
		}
		nsResolver = resolver.NewNSResolver(ctx, buildSearchPaths(fs, p.sourcePaths))
		rt.SetNSLoader(nsResolver)
	} else {
		// Refresh the resolver's search path from flags.
		nsResolver.SetPath(buildSearchPaths(fs, p.sourcePaths))
	}

	// Compile / bundle / wasm modes set the AOT flag.
	if p.compileOutput != "" || p.bundleOutput != "" || p.wasmOutput != "" {
		rt.CoreNS.Lookup("*compiling-aot*").(*vm.Var).SetRoot(vm.TRUE)
	}
	if p.compileOutput != "" {
		if len(files) != 1 {
			fmt.Fprintln(os.Stderr, "error: -c requires exactly one input file")
			return 1
		}
		if err := compileLG(ctx, nsResolver, files[0], p.compileOutput); err != nil {
			fmt.Fprint(os.Stderr, vm.FormatError(err))
			return 1
		}
		return 0
	}
	if p.bundleOutput != "" {
		if len(files) != 1 {
			fmt.Fprintln(os.Stderr, "error: -b requires exactly one input file")
			return 1
		}
		if err := bundleBinary(ctx, nsResolver, files[0], p.bundleOutput, p.bundleBase); err != nil {
			fmt.Fprint(os.Stderr, vm.FormatError(err))
			return 1
		}
		return 0
	}
	if p.wasmOutput != "" {
		if len(files) != 1 {
			fmt.Fprintln(os.Stderr, "error: -w requires exactly one input file")
			return 1
		}
		if err := buildWasm(ctx, nsResolver, files[0], p.wasmOutput); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		return 0
	}

	ranSomething := false
	if len(files) >= 1 {
		script := files[0]
		if filepath.Ext(script) == ".lgb" {
			if err := runLGB(script); err != nil {
				fmt.Print(vm.FormatError(err))
			}
		} else {
			if err := runFile(ctx, script); err != nil {
				fmt.Print(vm.FormatError(err))
			}
		}
		ranSomething = true
	}
	if p.expr != "" {
		ctx.SetSource("EXPR")
		val, err := runForm(ctx, p.expr)
		if err != nil {
			fmt.Print(vm.FormatError(err))
		} else {
			fmt.Println(val)
		}
		ranSomething = true
	}
	if !ranSomething || p.runREPL {
		motd(opts.Version, opts.Commit)
		if p.runNREPL {
			if err := nreplServe(ctx, p.nreplPort); err != nil {
				fmt.Println("failed to run nREPL server on port", p.nreplPort, err)
			}
			fmt.Printf("nREPL server running at tcp://127.0.0.1:%d\n", p.nreplPort)
		}
		repl(ctx)
	}
	return 0
}
