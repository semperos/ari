// Package help exports a function with default help for the core Goal language.
package help

import (
	"bufio"
	"strings"
)

// Wrap produces a help function suitable for use in cmd.Config by combining
// one or more help functions. For a given topic, the new combined help
// function tries each help function in order and returns the first non-empty
// result.
func Wrap(fs ...func(string) string) func(string) string {
	return func(s string) string {
		for _, f := range fs {
			if help := f(s); help != "" {
				return help
			}
		}
		return ""
	}
}

// HelpFunc returns a mapping from topics to help content for use in
// cmd.Config. It can be wrapped by extensions to provide additional entries or
// return alternative content for existing ones.
//
// It includes entries for the core language and the default os package
// extension.
func HelpFunc() func(string) string {
	var help map[string]string
	return func(s string) string {
		if help == nil {
			// lazily compute help when needed
			help = initHelp()
		}
		return help[s]
	}
}

func initHelp() map[string]string {
	help := map[string]string{}
	help[""] = helpTopics
	help["s"] = helpSyntax
	help["t"] = helpTypes
	help["v"] = helpVerbs
	help["nv"] = helpNamedVerbs
	help["a"] = helpAdverbs
	help["tm"] = helpTime
	help["time"] = helpTime // for the builtin name
	help["rt"] = helpRuntime
	help["io"] = helpIO
	const scols = 12  // syntax
	const vcols = 4   // verbs
	const acols = 5   // adverbs
	const nvcols = 10 // named verbs
	help[":"] = getBuiltin(helpSyntax, "assign", scols) + getBuiltin(helpVerbs, ":", vcols)
	help["::"] = getBuiltin(helpSyntax, "assign", scols) + getBuiltin(helpVerbs, "::", vcols)
	help["rx"] = getBuiltin(helpSyntax, "regexp", scols) + getBuiltin(helpNamedVerbs, "rx", nvcols)
	help["qq"] = getBuiltin(helpSyntax, "strings", scols)
	help["rq"] = getBuiltin(helpSyntax, "raw strings", scols)
	help["»"] = getBuiltin(helpVerbs, "»", vcols)
	help["rshift"] = help["»"]
	help["«"] = getBuiltin(helpVerbs, "«", vcols)
	help["shift"] = help["«"]
	for _, v := range []string{"+", "-", "*", "%", "!", "&", "|", "<", ">", "=", "~", ",", "^",
		"#", "_", "$", "?", "@", "."} {
		help[v] = getBuiltin(helpVerbs, v, vcols)
	}
	for _, v := range []string{"'", "/", "\\"} {
		help[v] = getBuiltin(helpAdverbs, v, acols)
	}
	for _, v := range []string{"abs", "bytes", "uc", "error", "eval", "firsts", "json", "ocount", "panic",
		"sign", "csv", "in", "mod", "nan", "rotate", "sub"} {
		help[v] = getBuiltin(helpNamedVerbs, v, nvcols)
	}
	help["¿"] = help["firsts"] + help["in"]
	for _, v := range []string{"atan", "cos", "exp", "log", "round", "sin", "sqrt"} {
		help[v] = getBuiltin(helpNamedVerbs, "MATH", 5)
	}
	for _, v := range []string{"rt.get", "rt.log", "rt.seed", "rt.time", "rt.try"} {
		help[v] = getBuiltin(helpRuntime, v, nvcols)
	}
	for _, v := range []string{"abspath", "chdir", "close", "dirfs", "env", "flush", "glob", "import", "mkdir", "open", "print",
		"read", "remove", "rename", "run", "say", "shell", "stat", "ARGS", "STDIN", "STDOUT", "STDERR"} {
		help[v] = getBuiltin(helpIO, v, nvcols)
	}
	help["subfs"] = help["dirfs"]
	return help
}

func getBuiltin(s string, v string, n int) string {
	var sb strings.Builder
	r := strings.NewReader(s)
	sc := bufio.NewScanner(r)
	match := false
	blanks := strings.Repeat(" ", n)
	for sc.Scan() {
		ln := sc.Text()
		if len(ln) < n {
			match = false
			continue
		}
		if strings.Contains(ln[:n], v) || ln[:n] == blanks && match {
			// NOTE: currently no builtin name is a substring of
			// another. Otherwise, this could match more names than
			// wanted.
			match = true
			sb.WriteString(ln)
			sb.WriteByte('\n')
			continue
		}
		match = false
	}
	return sb.String()
}
