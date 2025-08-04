package ari

import (
	_ "embed"
	"fmt"
	"os"
	"slices"
	"strings"

	"codeberg.org/anaseto/goal"
	"codeberg.org/anaseto/goal/archive/zip"
	"codeberg.org/anaseto/goal/encoding/base64"
	"codeberg.org/anaseto/goal/math"
	gos "codeberg.org/anaseto/goal/os"
	"github.com/semperos/ari/vendored/help"
)

const (
	monadic    = 1
	dyadic     = 2
	triadic    = 3
	quadratic  = 4
	tesseradic = 4
	pentadic   = 5
	hexadic    = 6
	heptadic   = 7
	octadic    = 8
)

// Goal Preamble in Goal

// goalLoadExtendedPreamble loads the goalSource* snippets below,
// loading them into the Goal context.
func goalLoadExtendedPreamble(ctx *goal.Context) error {
	additionalPackages := map[string]string{
		"dict":  goalSourceDict,
		"flag":  goalSourceFlag,
		"fmt":   goalSourceFmt,
		"fs":    goalSourceFs,
		"html":  goalSourceHTML,
		"k":     goalSourceK,
		"math":  goalSourceMath,
		"mods":  goalSourceMods,
		"os":    goalSourceOs,
		"path":  goalSourcePath,
		"table": goalSourceTable,
	}

	for pkg, source := range additionalPackages {
		_, err := ctx.EvalPackage(source, "<builtin>", pkg)
		if err != nil {
			return err
		}
	}
	_, err := ctx.EvalPackage(goalSourceAri, "<builtin>", "")
	if err != nil {
		return err
	}
	return nil
}

//go:embed vendor-goal/dict.goal
var goalSourceDict string

//go:embed vendor-goal/flag.goal
var goalSourceFlag string

//go:embed vendor-goal/fmt.goal
var goalSourceFmt string

//go:embed vendor-goal/fs.goal
var goalSourceFs string

//go:embed vendor-goal/html.goal
var goalSourceHTML string

//go:embed vendor-goal/k.goal
var goalSourceK string

//go:embed vendor-goal/math.goal
var goalSourceMath string

//go:embed vendor-goal/mods.goal
var goalSourceMods string

//go:embed vendor-goal/os.goal
var goalSourceOs string

//go:embed vendor-goal/path.goal
var goalSourcePath string

//go:embed vendor-goal/table.goal
var goalSourceTable string

//go:embed ari.goal
var goalSourceAri string

// Goal functions implemented in Go

// Implements Goal's help monad + Ari's help dyad.
func vfGoalHelp(help Help) func(_ *goal.Context, args []goal.V) goal.V {
	return func(_ *goal.Context, args []goal.V) goal.V {
		switch len(args) {
		case monadic:
			return helpMonadic(help, args)
		case dyadic:
			return helpDyadic(help, args)
		default:
			return goal.Panicf("help : too many arguments (%d), expects 1 or 2 arguments", len(args))
		}
	}
}

func helpMonadic(help Help, args []goal.V) goal.V {
	x := args[0]
	switch xv := x.BV().(type) {
	case goal.S:
		fmt.Fprintln(os.Stdout, strings.TrimSpace(help.Func(string(xv))))
	case *goal.R:
		regex := xv.Regexp()
		goalHelpMap, ok := help.Dictionary["goal"]
		if !ok {
			panic(`Developer Error: Help dictionary should have a "goal" entry.`)
		}
		for binding, helpString := range goalHelpMap {
			if !isGeneralHelpEntry(binding) && (regex.MatchString(binding) || regex.MatchString(helpString)) {
				fmt.Fprintln(os.Stdout, strings.TrimSpace(help.Func(binding)))
			}
		}
	default:
		return goal.Panicf("help x : x must be a string or regex, received a %q: %v", x.Type(), x)
	}
	return goal.NewI(1)
}

func helpDyadic(help Help, args []goal.V) goal.V {
	x := args[1]
	helpKeyword, ok := x.BV().(goal.S)
	if !ok {
		return panicType("s1 help s2", "s1", x)
	}
	y := args[0]
	helpString, ok := y.BV().(goal.S)
	if !ok {
		return panicType("s1 help s2", "s2", y)
	}
	help.Dictionary["goal"][string(helpKeyword)] = string(helpString)
	return goal.NewI(1)
}

// We do not want to match on these help topics for regex help matching, as it makes the output too noisy.
func isGeneralHelpEntry(entry string) bool {
	return slices.Contains([]string{
		"s", "t", "v", "nv", "a", "tm", "rt", "io",
	}, entry)
}

// Go <> Goal helpers

func stringMapFromGoalDict(d *goal.D) (map[string]string, error) {
	ka := d.KeyArray()
	va := d.ValueArray()
	m := make(map[string]string, ka.Len())
	switch kas := ka.(type) {
	case *goal.AS:
		switch vas := va.(type) {
		case *goal.AS:
			vasSlice := vas.Slice
			for i, k := range kas.Slice {
				m[k] = vasSlice[i]
			}
		default:
			return nil, fmt.Errorf("[Developer Error] stringMapFromGoalDict expects a Goal dict "+
				"with string keys and string values, but received values: %v", va)
		}
	default:
		return nil, fmt.Errorf("[Developer Error] stringMapFromGoalDict expects a Goal dict "+
			"with string keys and string values, but received keys: %v", ka)
	}
	return m, nil
}

func goalNewDictEmpty() *goal.D {
	dv := goal.NewD(goal.NewAV(nil), goal.NewAV(nil))
	d, ok := dv.BV().(*goal.D)
	if !ok {
		panic("Developer error: Empty Goal dictionary expected.")
	}
	return d
}

// Integration with other parts of Ari

func goalRegisterUniversalVariadics(ariContext *Context, goalContext *goal.Context, help Help) {
	gos.Import(goalContext, "")      // because Goal imports this without prefix
	math.Import(goalContext, "math") // because this file isn't prefixed
	zip.Import(goalContext, "")      // already zip-prefixed
	base64.Import(goalContext, "")   // already base64-prefixed
	goalContext.RegisterExtension("ari", AriVersion)
	goalContext.AssignGlobal("ari.c", goal.NewI(0))
	// Monads
	goalContext.RegisterMonad("ratelimit.new", vfRateLimitNew)
	goalContext.RegisterMonad("ratelimit.take", vfRateLimitTake)
	goalContext.RegisterMonad("time.day", vfTimeDay)
	goalContext.RegisterMonad("time.hour", vfTimeHour)
	goalContext.RegisterMonad("time.loadlocation", vfTimeLoadLocation)
	goalContext.RegisterMonad("time.location", vfTimeLocation)
	goalContext.RegisterMonad("time.locationstring", vfTimeLocationString)
	goalContext.RegisterMonad("time.microsecond", vfTimeMicrosecond)
	goalContext.RegisterMonad("time.millisecond", vfTimeMillisecond)
	goalContext.RegisterMonad("time.minute", vfTimeMinute)
	goalContext.RegisterMonad("time.month", vfTimeMonth)
	goalContext.RegisterMonad("time.now", vfTimeNow)
	goalContext.RegisterMonad("time.nanosecond", vfTimeNanosecond)
	goalContext.RegisterMonad("time.second", vfTimeSecond)
	goalContext.RegisterMonad("time.unix", vfTimeUnix)
	goalContext.RegisterMonad("time.unixmicro", vfTimeUnixMicro)
	goalContext.RegisterMonad("time.unixmilli", vfTimeUnixMilli)
	goalContext.RegisterMonad("time.unixnano", vfTimeUnixNano)
	goalContext.RegisterMonad("time.utc", vfTimeUTC)
	goalContext.RegisterMonad("time.weekday", vfTimeWeekDay)
	goalContext.RegisterMonad("time.year", vfTimeYear)
	goalContext.RegisterMonad("time.yearday", vfTimeYearDay)
	goalContext.RegisterMonad("time.zonename", vfTimeZoneName)
	goalContext.RegisterMonad("time.zoneoffset", vfTimeZoneOffset)
	goalContext.RegisterMonad("url.encode", vfURLEncode)
	// Dyads
	goalContext.RegisterDyad("help", vfGoalHelp(help))
	goalContext.RegisterDyad("http.client", vfHTTPClientFn())
	goalContext.RegisterDyad("http.delete", vfHTTPMaker(ariContext, "DELETE"))
	goalContext.RegisterDyad("http.get", vfHTTPMaker(ariContext, "GET"))
	goalContext.RegisterDyad("http.head", vfHTTPMaker(ariContext, "HEAD"))
	goalContext.RegisterDyad("http.options", vfHTTPMaker(ariContext, "OPTIONS"))
	goalContext.RegisterDyad("http.patch", vfHTTPMaker(ariContext, "PATCH"))
	goalContext.RegisterDyad("http.post", vfHTTPMaker(ariContext, "POST"))
	goalContext.RegisterDyad("http.put", vfHTTPMaker(ariContext, "PUT"))
	goalContext.RegisterDyad("time.add", vfTimeAdd)
	goalContext.RegisterDyad("time.addDate", vfTimeAddDate)
	goalContext.RegisterDyad("time.date", vfTimeDate)
	goalContext.RegisterDyad("time.fixedZone", vfTimeFixedZone)
	goalContext.RegisterDyad("time.format", vfTimeFormat)
	goalContext.RegisterDyad("time.parse", vfTimeParse)
	goalContext.RegisterDyad("time.sub", vfTimeSub)
	// Globals
	registerTimeGlobals(goalContext)
}

func goalRegisterVariadics(ariContext *Context, goalContext *goal.Context, help Help, sqlDatabase *SQLDatabase) {
	goalRegisterUniversalVariadics(ariContext, goalContext, help)
	// Monads
	goalContext.RegisterMonad("sql.close", vfSQLClose)
	goalContext.RegisterMonad("sql.open", vfSQLOpen)
	// Dyads
	goalContext.RegisterDyad("http.serve", vfServe)
	goalContext.RegisterDyad("sql.q", vfSQLQFn(sqlDatabase))
	goalContext.RegisterDyad("sql.exec", vfSQLExecFn(sqlDatabase))
}

//nolint:funlen
func GoalSyntax() map[string]string {
	return map[string]string{
		"each":           "'",
		"eachleft":       "`", // this is ASCII, but for completeness and less surprise
		"eachright":      "´",
		"fold":           "/",
		"reduce":         "/",
		"do":             "/",
		"while":          "/",
		"converge":       "/",
		"joinstrings":    "/",
		"decode":         "/",
		"scan":           "\\",
		"dos":            "\\",
		"whiles":         "\\",
		"converges":      "\\",
		"split":          "\\",
		"encode":         "\\",
		"rshift":         "»",
		"rightshift":     "»",
		"shiftright":     "»",
		"shift":          "«",
		"lshift":         "«",
		"leftshift":      "«",
		"shiftleft":      "«",
		"firsts":         "¿",
		"in":             "¿",
		"identity":       ":",
		"return":         ":",
		"assign":         ":",
		"flip":           "+",
		"add":            "+",
		"concat":         "+",
		"negate":         "-",
		"rtrim":          "-",
		"trimsuffix":     "-",
		"subtract":       "-",
		"first":          "*",
		"multiply":       "*",
		"times":          "*",
		"repeat":         "*",
		"classify":       "%",
		"divide":         "%",
		"enum":           "!",
		"fields":         "!",
		"odometer":       "!",
		"keys":           "!",
		"mod":            "!",
		"div":            "!",
		"padfields":      "!",
		"dict":           "!",
		"bytecount":      "&",
		"where":          "&",
		"keyswhere":      "&",
		"min":            "&",
		"and-fn":         "&",
		"reverse":        "|",
		"max":            "|",
		"or-fn":          "|",
		"sortup":         "<",
		"ascend":         "<",
		"less":           "<",
		"sortdown":       ">",
		"descend":        ">",
		"more":           ">",
		"lines":          "=",
		"indexcount":     "=",
		"groupkeys":      "=",
		"groupby":        "=",
		"equal":          "=",
		"not":            "~",
		"match":          "~",
		"enlist":         ",",
		"merge":          ",",
		"joinarrays":     ",",
		"sortkeys":       "^",
		"sort":           "^",
		"windows":        "^",
		"trim":           "^",
		"weedout":        "^",
		"withoutkeys":    "^",
		"withoutkeys&":   "^",
		"withoutvalues":  "^",
		"length":         "#",
		"tally":          "#",
		"count":          "#",
		"take/repeat":    "#",
		"replicate":      "#",
		"withkeys":       "#",
		"withkeys&":      "#",
		"withvalues":     "#",
		"floor":          "_",
		"tolower":        "_",
		"lower":          "_",
		"drop":           "_",
		"dropbytes":      "_",
		"trimprefix":     "_",
		"cut":            "_",
		"cutwhere":       "_",
		"cutstring":      "_",
		"string":         "$",
		"cutshape":       "$",
		"strings":        "$",
		"chars":          "$",
		"bytes":          "$",
		"tostring":       "$",
		"cast":           "$",
		"parse":          "$",
		"parsevalue":     "$",
		"format":         "$",
		"binsearch":      "$",
		"uniform":        "?",
		"normal":         "?",
		"distinct":       "?",
		"roll":           "?",
		"rollarray":      "?",
		"deal":           "?",
		"dealarray":      "?",
		"rindex":         "?",
		"index":          "?",
		"find":           "?",
		"findkey":        "?",
		"type":           "@",
		"take/pad":       "@",
		"substrstart":    "@",
		"matchregex":     "@",
		"regexmatch":     "@",
		"findgroup":      "@",
		"applyat":        "@",
		"atkey":          "@",
		"atrow":          "@",
		"at":             "@",
		"getglobal":      ".",
		"geterror":       ".",
		"values":         ".",
		"selfdict":       ".",
		"dictself":       ".",
		"substr":         ".",
		"substrstartend": ".",
		"findn":          ".",
		"findngroup":     ".",
		"deepat":         ".",
		"atrowkey":       ".",
		"setglobal":      "::",
		"amend":          "@",
		"tryat":          "@",
		"deepamend":      ".",
		"try":            ".",
		"cond":           "?",
	}
}

func GoalKeywordsHelp() map[string]string {
	httpclient := strings.Join([]string{
		`http.client d    HTTP client configured by entries in d, based on go-resty.`,
		`                 All entries optional. Dict values for Header, FormData, QueryParam`,
		`                 must be strings or lists of strings:`,
		`                  - AllowGetMethodPayload     i`,
		`                  - AuthScheme                s`,
		`                  - BaseURL                   s`,
		`                  - Debug                     i`,
		`                  - DisableWarn               i`,
		`                  - FormData                  d`,
		`                  - Header                    d`,
		`                  - HeaderAuthorizationKey    s`,
		`                  - PathParams                d`,
		`                  - QueryParam                d`,
		`                  - RawPathParams             d`,
		`                  - RetryCount                i`,
		`                  - RetryMaxWaitTimeMilli     i`,
		`                  - RetryResetReaders         i`,
		`                  - RetryWaitTimeMilli        i`,
		`                  - Token                     s`,
		`                  - UserInfo                  ..[Username:"user";Password:"pass"]`,
	}, "\n")
	sqlopen := `sql.open s    Open DuckDB database with data source name s`
	sqlq := `sql.q s    Run SQL query, results as table.`
	tuiColor := strings.Join([]string{
		`tui.color s           Color string accepted by lipgloss.Color`,
	}, "\n")
	tuiRender := strings.Join([]string{
		`tui.render style s    Return s marked up according to style (see tui.style)`,
	}, "\n")
	tuiStyle := strings.Join([]string{
		`tui.style d           Return a style based on entries in d:
		                        - Align (floats or one of "t", "r", "b", "l", or "c")
								- Background (tui.color)
								- Blink (bool)
								- Bold (bool)
								- Border (list of name + top, right, bottom, left bools)
								- BorderBackground (tui.color)
								- BorderForeground (tui.color)
								- Faint (bool)
		                        - Foreground (tui.color)
								- Height (int)
								- Italic (bool)
								- Margin (top, right, bottom, left ints)
								- Padding (top, right, bottom, left ints)
								- Reverse (bool)
								- Strikethrough (bool)
								- Underline (bool)
								- Width (int)`,
	}, "\n")
	vendoredGoalHelp := help.Map()
	ariGoalHelp := map[string]string{
		"http.client":  httpclient,
		"http.delete":  helpForHTTPFn("delete"),
		"http.get":     helpForHTTPFn("get"),
		"http.head":    helpForHTTPFn("head"),
		"http.options": helpForHTTPFn("options"),
		"http.patch":   helpForHTTPFn("patch"),
		"http.post":    helpForHTTPFn("post"),
		"http.put":     helpForHTTPFn("put"),
		"sql.open":     sqlopen,
		"sql.q":        sqlq,
		"tui.color":    tuiColor,
		"tui.render":   tuiRender,
		"tui.style":    tuiStyle,
	}
	for k, v := range ariGoalHelp {
		vendoredGoalHelp[k] = v
	}
	return vendoredGoalHelp
}

func helpForHTTPFn(s string) string {
	l := strings.ToLower(s)
	u := strings.ToUpper(s)
	return strings.Join([]string{
		fmt.Sprintf("       http.%s s         Make HTTP %s request for URL s", l, u),
		//nolint:lll
		fmt.Sprintf("  opts http.%s s         Make HTTP %s request for URL s with client opts dict (builds one-off client)", l, u),
		fmt.Sprintf("client http.%s s         Make HTTP %s request for URL s with client (from http.client)", l, u),
		fmt.Sprintf("http.%s[client;s;opts]   Make HTTP %s request for URL s with client and request opts dict", l, u),
		"                          For client opts, see http.client",
		"                          Supported request opts (depending on HTTP method):",
		"                          - Body",
		"                          - Debug",
		"                          - FormData",
		"                          - Header",
		"                          - PathParams",
		"                          - QueryParam",
		"                          - RawPathParams",
	}, "\n")
}

// panicType produces an bad type panic.
//
// Copied from Goal's implementation.
func panicType(op, sym string, x goal.V) goal.V {
	return goal.Panicf("%s : bad type %q in %s", op, x.Type(), sym)
}

// panicLength produces an length mismatch panic.
//
// Copied from Goal's implementation.
//
//nolint:unused // Going to use in subsequent commits
func panicLength(op string, n1, n2 int) goal.V {
	return goal.Panicf("%s : length mismatch: %d vs %d", op, n1, n2)
}
