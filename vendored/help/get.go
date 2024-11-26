// Package help exports a function with default help for the core Goal language.
package help

import (
	"bufio"
	"strings"
	"unicode/utf8"

	"codeberg.org/anaseto/goal/scan"
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
	parseBuiltins(help, getSyntaxKey, helpSyntax, 16)
	help["::"] = help[":"] // for syntax assign entry
	parseBuiltins(help, getVerb, helpVerbs, 4)
	help["rshift"] = help["»"]
	help["shift"] = help["«"]
	parseBuiltins(help, getVerb, helpAdverbs, 7)
	parseBuiltins(help, getNamedVerb, helpNamedVerbs, 11)
	help["¿"] = help["firsts"] + help["in"]
	parseBuiltins(help, getNamedVerb, helpRuntime, 16)
	parseBuiltins(help, getNamedVerb, helpIO, 12)
	help["subfs"] = help["dirfs"]
	return help
}

func parseBuiltins(help map[string]string, f func(string) string, s string, n int) {
	sc := bufio.NewScanner(strings.NewReader(s))
	var verb string // current verb
	blanks := strings.Repeat(" ", n)
	for sc.Scan() {
		ln := sc.Text()
		if len(ln) < n || strings.HasSuffix(ln, " HELP") {
			verb = ""
			continue
		}
		v := f(ln[:n])
		if v != "" {
			verb = v
		}
		if v != "" || ln[:n] == blanks && verb != "" {
			help[verb] += ln + "\n"
			continue
		}
		verb = ""
	}
}

// getNamedVerb returns the first multi-letter word in the string.
func getNamedVerb(s string) string {
	from, to := 0, len(s)
	for i, r := range s {
		if !scan.IsLetter(r) && r != '.' {
			if i-from > 1 {
				// multi-letter word
				to = i
				break
			}
			from = i + utf8.RuneLen(r)
		}
	}
	return s[from:to]
}

// getVerb returns the first non-letter non-space symbol in the string
// (possibly a digraph, for ::).
func getVerb(s string) string {
	from, to := 0, len(s)
	for i, r := range s {
		if scan.IsLetter(r) || r == ' ' || r == '[' || r == ';' || r == ']' {
			if i-from > 0 {
				to = i
				break
			}
			from = i + utf8.RuneLen(r)
		}
	}
	return s[from:to]
}

// getSyntaxKey returns a help key for a few syntax entries.
func getSyntaxKey(s string) string {
	switch {
	case strings.Contains(s, "assign"):
		return ":"
	case strings.Contains(s, "regexp"):
		return "rx"
	case strings.Contains(s, "raw strings"):
		return "rq"
	case strings.Contains(s, "strings"):
		return "qq"
	default:
		return ""
	}
}

func Map() map[string]string {
	return initHelp()
}
