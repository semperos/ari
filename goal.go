package ari

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"strings"

	"codeberg.org/anaseto/goal"
	gos "codeberg.org/anaseto/goal/os"
)

const (
	monadic = 1
	dyadic  = 2
	triadic = 3
)

// Goal Preamble in Goal

// goalLoadExtendedPreamble loads the goalSource* snippets below,
// loading them into the Goal context.
func goalLoadExtendedPreamble(ctx *goal.Context) error {
	goalPackages := map[string]string{
		"":     goalSourceShape,
		"fmt":  goalSourceFmt,
		"html": goalSourceHTML,
		"k":    goalSourceK,
		"math": goalSourceMath,
		"mods": goalSourceMods,
	}
	for pkg, source := range goalPackages {
		_, err := ctx.EvalPackage(source, "<builtin>", pkg)
		if err != nil {
			return err
		}
	}
	return nil
}

//go:embed vendor-goal/fmt.goal
var goalSourceFmt string

//go:embed vendor-goal/html.goal
var goalSourceHTML string

//go:embed vendor-goal/k.goal
var goalSourceK string

//go:embed vendor-goal/math.goal
var goalSourceMath string

//go:embed vendor-goal/mods.goal
var goalSourceMods string

//go:embed vendor-goal/shape.goal
var goalSourceShape string

// Goal functions implemented in Go

// printV adapted from Goal's implementation found in the os.go file.
func printV(ctx *goal.Context, x goal.V) error {
	switch xv := x.BV().(type) {
	case goal.S:
		_, err := fmt.Fprint(os.Stdout, string(xv))
		return err
	case *goal.AS:
		buf := bufio.NewWriter(os.Stdout)
		imax := xv.Len() - 1
		for i, s := range xv.Slice {
			_, err := buf.WriteString(s)
			if err != nil {
				return err
			}
			if i < imax {
				err := buf.WriteByte(' ')
				if err != nil {
					return err
				}
			}
		}
		return buf.Flush()
	default:
		_, err := fmt.Fprintf(os.Stdout, "%s", x.Append(ctx, nil, false))
		return err
	}
}

func VFHelpFn(help Help) func(goalContext *goal.Context, args []goal.V) goal.V {
	return func(goalContext *goal.Context, args []goal.V) goal.V {
		x := args[len(args)-1]
		switch len(args) {
		case monadic:
			return helpMonadic(goalContext, help, x)
		case dyadic:
			return helpDyadic(help, x, args)
		case triadic:
			return helpTriadic(help, x, args)
		default:
			return goal.Panicf("sql.q : too many arguments (%d), expects 1 or 2 arguments", len(args))
		}
	}
}

const noHelpString = "(no help)"

func helpMonadic(goalContext *goal.Context, help Help, x goal.V) goal.V {
	switch helpKeyword := x.BV().(type) {
	case goal.S:
		helpKeywordString := string(helpKeyword)
		if helpKeywordString == "" {
			return helpReturnAsDict(help)
		}
		return helpPrintExactMatch(help, helpKeyword, goalContext)
	case *goal.R:
		anyMatches := helpPrintRegexMatches(help, helpKeyword.Regexp())
		if anyMatches {
			return goal.NewI(1)
		}
		fmt.Fprintln(os.Stdout, "(no help matches)")
		return goal.NewI(0)
	default:
		return panicType("help s-or-rx", "s-or-rx", x)
	}
}

func helpPrintExactMatch(help Help, helpKeyword goal.S, goalContext *goal.Context) goal.V {
	goalHelp := help["goal"]
	if helpString, ok := goalHelp[string(helpKeyword)]; ok {
		err := printV(goalContext, goal.NewS(helpString))
		if err != nil {
			return goal.NewPanicError(err)
		}
		fmt.Fprintln(os.Stdout)
		return goal.NewI(1)
	}
	err := printV(goalContext, goal.NewS(noHelpString))
	if err != nil {
		return goal.NewPanicError(err)
	}
	fmt.Fprintln(os.Stdout)
	return goal.NewI(0)
}

func helpReturnAsDict(help Help) goal.V {
	categories := make([]string, 0, len(help))
	categoryDicts := make([]goal.V, 0)
	for category, v := range help {
		categories = append(categories, category)
		categoryDicts = append(categoryDicts, stringMapToGoalDict(v))
	}
	return goal.NewD(goal.NewAS(categories), goal.NewAV(categoryDicts))
}

func helpPrintRegexMatches(help Help, r *regexp.Regexp) bool {
	goalHelp := help["goal"]
	anyMatches := false
	for k, v := range goalHelp {
		// Skip roll-up help topics that have a ":" in them,
		// since it produces too much (and duplicated) output.
		if !strings.Contains(k, ":") {
			if r.MatchString(k) || r.MatchString(v) {
				fmt.Fprintln(os.Stdout, v)
				anyMatches = true
			}
		}
	}
	return anyMatches
}

func helpDyadic(help Help, x goal.V, args []goal.V) goal.V {
	helpKeyword, ok := x.BV().(goal.S)
	if !ok {
		return panicType("s1 help s2", "s1", x)
	}
	y := args[0]
	helpString, ok := y.BV().(goal.S)
	if !ok {
		return panicType("s1 help s2", "s2", y)
	}
	goalHelp := help["goal"]
	goalHelp[string(helpKeyword)] = string(helpString)
	return goal.NewI(1)
}

func helpTriadic(help Help, x goal.V, args []goal.V) goal.V {
	helpCategory, ok := x.BV().(goal.S)
	if !ok {
		return panicType("help[category;keyword;helpstring]", "category", x)
	}
	y := args[1]
	helpKeyword, ok := y.BV().(goal.S)
	if !ok {
		return panicType("help[category;keyword;helpstring]", "keyword", y)
	}
	z := args[0]
	helpString, ok := z.BV().(goal.S)
	if !ok {
		return panicType("help[category;keyword;helpstring]", "helpstring", z)
	}
	categoryHelp, ok := help["goal"]
	if !ok {
		help[string(helpCategory)] = map[string]string{string(helpKeyword): string(helpString)}
	} else {
		categoryHelp[string(helpKeyword)] = string(helpString)
	}
	return goal.NewI(1)
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

func stringMapToGoalDict(m map[string]string) goal.V {
	keys := make([]string, 0, len(m))
	values := make([]string, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}
	ks := goal.NewAS(keys)
	vs := goal.NewAS(values)
	return goal.NewD(ks, vs)
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

func goalRegisterVariadics(ariContext *Context, goalContext *goal.Context, help Help, sqlDatabase *SQLDatabase) {
	// From Goal itself, os lib imported without prefix
	gos.Import(goalContext, "")
	// Ari
	// Monads
	goalContext.RegisterMonad("sql.close", VFSqlClose)
	goalContext.RegisterMonad("sql.open", VFSqlOpen)
	goalContext.RegisterMonad("time.now", VFTimeNow)
	// Dyads
	goalContext.RegisterDyad("help", VFHelpFn(help))
	goalContext.RegisterDyad("http.client", VFHTTPClientFn())
	goalContext.RegisterDyad("http.delete", VFHTTPMaker(ariContext, "DELETE"))
	goalContext.RegisterDyad("http.get", VFHTTPMaker(ariContext, "GET"))
	goalContext.RegisterDyad("http.head", VFHTTPMaker(ariContext, "HEAD"))
	goalContext.RegisterDyad("http.options", VFHTTPMaker(ariContext, "OPTIONS"))
	goalContext.RegisterDyad("http.patch", VFHTTPMaker(ariContext, "PATCH"))
	goalContext.RegisterDyad("http.post", VFHTTPMaker(ariContext, "POST"))
	goalContext.RegisterDyad("http.put", VFHTTPMaker(ariContext, "PUT"))
	goalContext.RegisterDyad("sql.q", VFSqlQFn(sqlDatabase))
	goalContext.RegisterDyad("sql.exec", VFSqlExecFn(sqlDatabase))
	goalContext.RegisterDyad("time.add", VFTimeAdd)
	goalContext.RegisterDyad("time.parse", VFTimeParse)
	goalContext.RegisterDyad("time.sub", VFTimeSub)
	// Globals
	registerTimeGlobals(goalContext)
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
		"amend":          "@[",
		"tryat":          "@[",
		"deepamend":      ".[",
		"try":            ".[",
		"cond":           "?[",
	}
}

type mapentry struct {
	k string
	v string
}

func goalAdverbsHelp() []mapentry {
	return []mapentry{
		{k: "'", v: strings.Join([]string{
			`f'x    each      #'(4 5;6 7 8) → 2 3`,
			`x F'y  each      2 3#'4 5 → (4 4;5 5 5)      {(x;y;z)}'[1;2 3;4] → (1 2 4;1 3 4)`,
			`x I'y  case      (6 7 8 9)0 1 0 1'"a""b""c""d" → 6 "b" 8 "d"`,
			`I A'I  at each   m:3$!9;p:!2 2;(m').p → 0 1 3 4`,
		}, "\n")},
		{k: "`", v: "x F`" + `y  eachleft  1 2,` + "`" + `"a""b" → (1 "a" "b";2 "a" "b")           (same as F[;y]'x)`},
		{k: "´", v: `x F` + "´y  eachright 1 2,´" + `"a""b" → (1 2 "a";1 2 "b")               (same as F[x;]'y)`},
		{k: "/", v: strings.Join([]string{
			`F/x    fold      +/!10 → 45`,
			`x F/y  fold      5 6+/!4 → 11 12                     {x+y-z}/[5;4 3;2 1] → 9`,
			`i f/y  do        3(2*)/4 → 32`,
			`f f/y  while     (100>)(2*)/4 → 128`,
			`f/x    converge  {1+1.0%x}/1 → 1.618033988749895     {-x}/1 → -1`,
			`s/S    join      ","/"a" "b" "c" → "a,b,c"`,
			`I/x    decode    24 60 60/1 2 3 → 3723               2/1 1 0 → 6`,
		}, "\n")},
		{k: `\`, v: strings.Join([]string{
			`F\x    scan      +\!10 → 0 1 3 6 10 15 21 28 36 45`,
			`x F\y  scan      5 6+\!4 → (5 6;6 7;8 9;11 12)       {x+y-z}\[5;4 3;2 1] → 7 9`,
			`i f\y  dos       3(2*)\4 → 4 8 16 32`,
			`f f\y  whiles    (100>)(2*)\4 → 4 8 16 32 64 128`,
			`f\x    converges (-2!)\10 → 10 5 2 1 0               {-x}\1 → 1 -1`,
			`s\s    split     ","\"a,b,c" → "a" "b" "c"           ""\"aπc" → "a" "π" "c"`,
			`r\s    split     rx/[,;]/\"a,b;c,d" → "a" "b" "c" "d"`,
			`i s\s  splitN    (2)","\"a,b,c" → "a" "b,c"`,
			`i r\s  splitN    (3)rx/[,;]/\"a,b;c,d" → "a" "b" "c,d"`,
			`I\x    encode    24 60 60\3723 → 1 2 3               2\6 → 1 1 0`,
		}, "\n")},
	}
}

//nolint:funlen
func goalVerbsHelp() []mapentry {
	return []mapentry{
		{k: ":", v: strings.Join([]string{
			`:x  identity    :[42] → 42  (recall that : is also syntax for return and assign)`,
			`x:y right       2:3 → 3                        "a":"b" → "b"`,
		}, "\n")},
		{k: "+", v: strings.Join([]string{
			`+d  swap k/v    +"a""b"!0 1 → 0 1!"a" "b"`,
			`+x  flip        +(1 2;3 4) → (1 3;2 4)         +42 → ,,42`,
			`n+n add         2+3 → 5                        2+3 4 → 5 6`,
			`s+s concat      "a"+"b" → "ab"                 "a" "b"+"c" → "ac" "bc"`,
		}, "\n")},
		{k: "-", v: strings.Join([]string{
			`-n  negate      - 2 3 → -2 -3                  -(1 2.5;3 4) → (-1.0 -2.5;-3 -4)`,
			`-s  rtrim space -"a\tb \r\n" " c d \n" → "a\tb" " c d"   (Unicode's White Space)`,
			`n-n subtract    5-3 → 2                        5 4-3 → 2 1`,
			`s-s trim suffix "file.txt"-".txt" → "file"`,
		}, "\n")},
		{k: "*", v: strings.Join([]string{
			`*x  first       *7 8 9 → 7                     *"ab" → "ab"           *(+;*) → +`,
			`n*n multiply    2*3 → 6                        1 2 3*3 → 3 6 9`,
			`s*i repeat      "a"*3 2 1 0 → "aaa" "aa" "a" ""`,
		}, "\n")},
		{k: "%", v: strings.Join([]string{
			`%X  classify    %7 8 9 7 8 9 → 0 1 2 0 1 2     %"a" "b" "a" → 0 1 0`,
			`n%n divide      3%2 → 1.5                      3 4%2 → 1.5 2.0`,
		}, "\n")},
		{k: "!", v: strings.Join([]string{
			`!i  enum        !5 → 0 1 2 3 4                 !-5 → -5 -4 -3 -2 -1`,
			`!s  fields      !"a b\tc\nd \u00a0e" → "a""b""c""d""e"   (Unicode's White Space)`,
			`!I  odometer    !2 3 → (0 0 0 1 1 1;0 1 2 0 1 2)`,
			`!d  keys        !"a""b"!1 2 → "a" "b"`,
			`i!n mod/div     3!9 8 7 → 0 2 1            -3!9 8 7 → 3 2 2`,
			`i!s pad fields  3!"a" → "a  "              -3!"1" "23" "456" → "  1" " 23" "456"`,
			`s!s fields      ",;"!"a,b;c" → "a""b""c" (fields cut on any of ",;"; ""!s is !s)`,
			`X!Y dict        d:"a""b"!1 2; d"a" → 1              (same as d:..[a:1;b:2]; d..a)`,
		}, "\n")},
		{k: "&", v: strings.Join([]string{
			`&s  byte-count  &"abc" → 3                      &"π" → 2              &"αβγ" → 6`,
			`&I  where       &0 0 1 0 0 0 1 → 2 6            &2 3 → 0 0 1 1 1`,
			`&d  keys where  &"a""b""c""d"!0 1 1 0 → "b" "c"`,
			`x&y min/and     2&3 → 2          4&3 → 3         "b"&"a" → "a"           0&1 → 0`,
		}, "\n")},
		{k: "|", v: strings.Join([]string{
			`|X  reverse     |!5 → 4 3 2 1 0`,
			`x|y max/or      2|3 → 3          4|3 → 4         "b"|"a" → "b"           0|1 → 1`,
		}, "\n")},
		{k: "<", v: strings.Join([]string{
			`<d  sort up     <"a""b""c"!2 3 1 → "c""a""b"!1 2 3`,
			`<X  ascend      <3 5 4 → 0 2 1           (index permutation for ascending order)`,
			`x<y less        2<3 → 1          "c"<"a" → 0                       7 8<6 9 → 0 1`,
		}, "\n")},
		{k: ">", v: strings.Join([]string{
			`>d  sort down   >"a""b""c"!2 3 1 → "b""a""c"!3 2 1`,
			`>X  descend     >3 5 4 → 1 2 0          (index permutation for descending order)`,
			`x>y more        2>3 → 0          "c">"a" → 1                       7 8>6 9 → 1 0`,
		}, "\n")},
		{k: "=", v: strings.Join([]string{
			`=s  lines       ="ab\ncd\r\nef gh" → "ab" "cd" "ef gh"`,
			`=I  index-count =1 0 0 2 2 3 -1 2 1 1 1 → 2 4 3 1`,
			`=d  group keys  ="a""b""c"!0 1 0 → ("a" "c";,"b")           ="a""b"!0 -1 → ,,"a"`,
			`f=Y group by    (2!)=!10 → (0 2 4 6 8;1 3 5 7 9)`,
			`x=y equal       2 3 4=3 → 0 1 0                          "ab" = "ba" → 0`,
		}, "\n")},
		{k: "~", v: strings.Join([]string{
			`~x  not         ~0 1 2 → 1 0 0                           ~"a" "" "0" → 0 1 0`,
			`x~y match       3~3 → 1            2 3~3 2 → 0           ("a";%)~'("b";%) → 0 1`,
		}, "\n")},
		{k: ",", v: strings.Join([]string{
			`,x  enlist      ,1 → ,1            #,2 3 → 1             (list with one element)`,
			`d,d merge       ("a""b"!1 2),"b""c"!3 4 → "a""b""c"!1 3 4`,
			`x,y join        1,2 → 1 2                       "ab" "c","d" → "ab" "c" "d"`,
		}, "\n")},
		{k: "^", v: strings.Join([]string{
			`^d  sort keys   ^"c""a""b"!1 2 3 → "a""b""c"!2 3 1`,
			`^X  sort        ^3 5 0 → 0 3 5                  ^"ca" "ab" "bc" → "ab" "bc" "ca"`,
			`i^s windows     2^"abcde" → "abcd" "bcde"`,
			`i^Y windows     2^!4 → (0 1 2;1 2 3)                   -2^!4 → (0 1;1 2;2 3)`,
			`s^s trim        " #"^"  #a ## b#  " → "a ## b"         ""^" \na\t b\t" → "a\t b"`,
			`f^y weed out    {0 1 1 0}^4 1 5 3 → 4 3                (0<)^2 -3 1 → ,-3`,
			`X^t w/o keys &  (,"b";1 0)^..[a:6 7;b:8 9] → (,"a")!,,7`,
			`                (0;..a>1;..b<0)^..[a:1 2 3;b:4 -5 6] → "a""b"!(,1;,4)`,
			`X^d w/o keys    (,"b")^"a""b""c"!0 1 2 → "a""c"!0 2`,
			`X^Y w/o values  2 3^1 1 2 3 3 4 → 1 1 4                          (like in[;X]^Y)`,
		}, "\n")},
		{k: "#", v: strings.Join([]string{
			`#x  length      #7 8 9 → 3        #"ab" "cd" → 2       #42 → 1      #"ab" → 1`,
			`i#y take/repeat 2#6 7 8 → 6 7     5#6 7 8 → 6 7 8 6 7               3#1 → 1 1 1`,
			`s#s count       "ab"#"cabdab" "cd" "deab" → 2 0 1                   ""#"αβγ" → 4`,
			`f#y replicate   {0 1 2 0}#4 1 5 3 → 1 5 5              (0<)#2 -3 1 → 2 1`,
			`X#t w/ keys &   (,"a";0 1)#..[a:6 7;b:8 9] → (,"a")!,,7`,
			`                (1;..a>1;..b>0)#..[a:1 2 3;b:4 -5 6] → "a""b"!(,3;,6)`,
			`X#d with keys   "a""c""e"#"a""b""c""a"!2 3 4 5 → "a""c""a"!2 4 5`,
			`X#Y with values 2 3#1 1 2 3 3 4 → 2 3 3                          (like in[;X]#Y)`,
		}, "\n")},
		{k: "_", v: strings.Join([]string{
			`_n  floor       _2.3 → 2.0                  _1.5 3.7 → 1.0 3.0`,
			`_s  to lower    _"ABC" → "abc"              _"AB" "CD" "Π" → "ab" "cd" "π"`,
			`i_s drop bytes  2_"abcde" → "cde"           -2_"abcde" → "abc"`,
			`i_Y drop        2_3 4 5 6 → 5 6             -2_3 4 5 6 → 3 4`,
			`s_s trim prefix "pfx-"_"pfx-name" "pfx2-name" → "name" "pfx2-name"`,
			`f_Y cut where   {0=3!x}_!10 → (0 1 2;3 4 5;6 7 8;,9)          (same as (&f Y)_Y)`,
			`I_s cut string  1 3_"abcdef" → "bc" "def"                          (I ascending)`,
			`I_Y cut         2 5_!10 → (2 3 4;5 6 7 8 9)                        (I ascending)`,
		}, "\n")},
		{k: "$", v: strings.Join([]string{
			`$x  string      $2 3 → "2 3"      $"text" → "\"text\""`,
			`i$s cut shape   3$"abcdefghijk" → "abc" "defg" "hijk"                   (reshape)`,
			`i$Y cut shape   3$!6 → (0 1;2 3;4 5)             -3$!6 → (0 1 2;3 4 5)  (reshape)`,
			`s$y strings     "s"$(1;"c";+) → "1""c""+"`,
			`s$s chars/bytes "c"$"aπ" → 97 960                "b"$"aπ" → 97 207 128`,
			`s$i to string   "c"$97 960 → "aπ"                "b"$97 207 128 → "aπ"`,
			`s$n cast        "i"$2.3 → 2                      @"n"$42 → "n"`,
			`s$s parse i/n   "i"$"42" "0b100" → 42 4          "n"$"2.5" "1e+20" → 2.5 1e+20`,
			`s$s parse value "v"$qq/(2 3;"a")/ → (2 3;"a")   ($x inverse for types in "inrs")`,
			`s$y format      "%.2g"$1 4%3 → "0.33" "1.3"      "%s=%03d"$"a" 42 → "a=042"`,
			`X$y binsearch   2 3 5 7$8 2 7 5 5.5 3 0 → 4 1 4 3 3 2 0            (X ascending)`,
		}, "\n")},
		{k: "?", v: strings.Join([]string{
			`?i  uniform     ?2 → 0.6046602879796196 0.9405090880450124     (between 0 and 1)`,
			`?i  normal      ?-2 → -1.233758177597947 -0.12634751070237293    (mean 0, dev 1)`,
			`?X  distinct    ?2 2 4 3 2 3 → 2 4 3                   (keeps first occurrences)`,
			`i?i roll        5?100 → 10 51 21 51 37`,
			`i?Y roll array  5?"a" "b" "c" → "c" "a" "c" "c" "b"`,
			`i?i deal        -5?100 → 19 26 0 73 94                         (always distinct)`,
			`i?Y deal array  -3?"a""b""c" → "a""c""b"     (0i?Y is (-#Y)?Y) (always distinct)`,
			`s?r rindex      "abcde"?rx/b../ → 1 3                            (offset;length)`,
			`s?s index       "a = a + 1"?"=" "+" → 2 6`,
			`d?y find key    ("a""b"!3 4)?4 → "b"                    ("a" "b"!3 4)?5 → ""`,
			`X?y find        9 8 7?8 → 1                              9 8 7?6 → 3`,
		}, "\n")},
		{k: "¿", v: strings.Join([]string{
			`firsts X        mark firsts  firsts 0 0 2 3 0 2 3 4 → 1 0 1 1 0 0 0 1    (same as ¿X)`,
			`x in s          contained    "bc" "ac"in"abcd" → 1 0                    (same as x¿s)`,
			`x in Y          member of    2 3 in 8 2 4 → 1 0                         (same as x¿Y)`,
		}, "\n")},
		{k: "@", v: strings.Join([]string{
			`@x  type        @2 → "i"    @1.5 → "n"    @"ab" → "s"    @2 3 → "I"     @+ → "f"`,
			`i@y take/pad    2@6 7 8 → 6 7     4@6 7 8 → 6 7 8 0      -4@6 7 8 → 0 6 7 8`,
			`s@i substr      "abcdef"@2  → "cdef"                                 (s[offset])`,
			`r@s match       rx/^[a-z]+$/"abc" → 1                    rx/\s/"abc" → 0`,
			`r@s find group  rx/([a-z])(.)/"&a+c" → "a+" "a" "+"      (whole match, group(s))`,
			`f@y apply at    (|)@1 2 → 2 1            (like |[1 2] → 2 1 or |1 2 or (|).,1 2)`,
			`d@y at key      ..[a:6 7;b:8 9]"a" → 6 7                 (1 2!"a""b")2 → "b"`,
			`t@i at row      ..[a:6 7;b:8 9]0 → "a""b"!6 8`,
			`X@i at          7 8 9@2 → 9             7 8 9[2 0] → 9 7            7 8 9@-2 → 8`,
		}, "\n")},
		{k: ".", v: strings.Join([]string{
			`.s  get global  a:3;."a" → 3`,
			`.e  get error   .error"msg" → "msg"`,
			`.d  values      ."a""b"!1 2 → 1 2`,
			`.X  self-dict   ."a""b" → "a""b"!"a""b"          .!3 → 0 1 2!0 1 2`,
			`s.I substr      "abcdef"[2;3] → "cde"                         (s[offset;length])`,
			`r.y findN       rx/[a-z]/["abc";2] → "a""b"      rx/[a-z]/["abc";-1] → "a""b""c"`,
			`r.y findN group rx/[a-z](.)/["abcdef";2] → ("ab" "b";"cd" "d")`,
			`f.y apply       (+).2 3 → 5                      +[2;3] → 5`,
			`d.y deep at     ..[a:6 7;b:8 9]["a";1] → 7`,
			`t.y at row;key  ..[a:6 7;b:8 9][1;"a"] → (,"a")!,7`,
			`X.y deep at     (6 7;8 9)[0;1] → 7               (6 7;8 9)[;1] → 7 9`,
		}, "\n")},
		{k: "«", v: strings.Join([]string{
			`«X  shift       «8 9 → 9 0     «"a" "b" → "b" ""        (ASCII keyword: shift x)`,
			`x«Y shift       "a" "b"«1 2 3 → 3 "a" "b"`,
		}, "\n")},
		{k: "»", v: strings.Join([]string{
			`»X  rshift      »8 9 → 0 8     »"a" "b" → "" "a"       (ASCII keyword: rshift x)`,
			`x»Y rshift      "a" "b"»1 2 3 → "a" "b" 1`,
		}, "\n")},
		{k: "::", v: `::[s;y]     set global  ::["a";3];a → 3  (brackets needed because :: is monadic)`},
		{k: "@[", v: strings.Join([]string{
			`@[d;y;f]    amend       @["a""b""c"!7 8 9;"a""b""b";10+] → "a""b""c"!17 28 9`,
			`@[X;i;f]    amend       @[7 8 9;0 1 1;10+] → 17 28 9`,
			`@[d;y;F;z]  amend       @["a""b""c"!7 8 9;"a";:;42] → "a""b""c"!42 8 9`,
			`@[X;i;F;z]  amend       @[7 8 9;1 2 0;+;10 20 -10] → -3 18 29`,
			`@[f;x;f]    try at      @[2+;3;{"msg"}] → 5           @[2+;"a";{"msg"}] → "msg"`,
		}, "\n")},
		{k: ".[", v: strings.Join([]string{
			`.[X;y;f]    deep amend  .[(6 7;8 9);0 1;-] → (6 -7;8 9)`,
			`.[X;y;F;z]  deep amend  .[(6 7;8 9);(0 1 0;1);+;10] → (6 27;8 19)`,
			`                        .[(6 7;8 9);(*;1);:;42] → (6 42;8 42)`,
			`.[f;x;f]    try         .[+;2 3;{"msg"}] → 5          .[+;2 "a";{"msg"}] → "msg"`,
		}, "\n")},
	}
}

func GoalSyntaxHelp() map[string]string {
	verbHelp := goalVerbsHelp()
	adverbHelp := goalAdverbsHelp()
	m := make(map[string]string, len(verbHelp)+len(adverbHelp))
	for _, entry := range verbHelp {
		m[entry.k] = entry.v
	}
	for _, entry := range adverbHelp {
		m[entry.k] = entry.v
	}
	return m
}

func GoalGlobalsHelp() map[string]string {
	return map[string]string{
		"STDERR": "standard error filehandle (buffered)",
		"STDIN":  "standard input filehandle (buffered)",
		"STDOUT": "standard output filehandle (buffered)",
	}
}

//nolint:funlen
func GoalKeywordsHelp() map[string]string {
	// Some help strings are used for individual entries and topic-based ones.
	rtget := strings.Join([]string{
		`rt.get s        returns various kinds of runtime information`,
		`                "g"   dictionary with copy of all global variables`,
		`                "f"   same but only globals containing functions`,
		`                "v"   same but only non-function globals`,
	}, "\n")
	rtlog := `rt.log x        like :[x] but logs string representation of x       (same as \x)`
	rtseed := "rt.seed i       set non-secure pseudo-rand seed to i        (used by the ? verb)"
	rttime := strings.Join([]string{
		`rt.time[s;i]    eval s for i times (default 1), return average time (ns)`,
		`rt.time[f;x;i]  call f. x for i times (default 1), return average time (ns)`,
	}, "\n")
	abs := `abs n      abs/mag      abs -3.0 -1.5 2.0 → 3.0 1.5 2.0`
	and := `and[1;2] → 2    and[1;0;3] → 0`
	atan := `atan[n;n]`
	chdir := `chdir s     change current working directory to s, or return an error`
	closeHelp := `close h     flush any buffered data, then close handle h`
	cos := `cos n      cos 0 → 1.0`
	csv := strings.Join([]string{
		`  csv s    csv read     csv"a,b\n1,2" → ("a" "1";"b" "2")`,
		`  csv A    csv write    csv("a" "b";1 2) → "a,1\nb,2\n"`,
		`s csv s    csv read     " "csv"a b\n1 2" → ("a" "1";"b" "2")  (" " as separator)`,
		`s csv A    csv write    " "csv("a" "b";1 2) → "a 1\nb 2\n"    (" " as separator)`,
	}, "\n")
	env := strings.Join([]string{
		`  env s     get environment variable s, or an error if unset`,
		`            return a dictionary representing the whole environment if s~""`,
		`x env s     set environment variable x to s, or return an error`,
		`x env 0     unset environment variable x, or clear environment if x~""`,
	}, "\n")
	errorHelp := `error x    error        r:error"msg"; (@r;.r) → "e" "msg"`
	eval := strings.Join([]string{
		`eval s     comp/run     a:5;eval"a+2" → 7           (unrestricted variant of .s)`,
		`eval[s;loc;pfx]         like eval s, but provide loc as location (usually a`,
		`                        path), and prefix pfx+"." for globals; does not eval`,
		`                        same location more than once`,
	}, "\n")
	exp := `exp n      nat exp      exp 1 → 2.718281828459045`
	firsts := `firsts X   mark firsts  firsts 0 0 2 3 0 2 3 4 → 1 0 1 1 0 0 0 1    (same as ¿X)`
	flush := `flush h     flush any buffered data for handle h`
	help := strings.Join([]string{
		`help s                      help string for keyword s`,
		`help ""                     all ari help as a dictionary`,
		`s1 help s2                  set s2 as the help string for keyword s1`,
		`help[category;keyword;s]    set s as the help string for keyword in category (all strings)`,
	}, "\n")
	goalHelp := strings.Join([]string{
		`GOAL HELP`,
		``,
		`Enter help"ari" for info on ari as a whole, or help"goal" for this help`,
		``,
		`Add your own help with "myvar"help"mydoc"`,
		``,
		`GOAL HELP TOPICS`,
		`Enter help"TOPIC" where TOPIC is one of:`,
		``,
		`"goal:s"     syntax`,
		`"goal:t"     value types`,
		`"goal:v"     verbs (like +*-%,)`,
		`"goal:nv"    named verbs (like in, sign)`,
		`"goal:a"     adverbs (/\')`,
		`"goal:tm"    time handling`,
		`"goal:rt"    runtime system`,
		`"goal:io"    IO verbs (like say, open, read)`,
		`op           where op is a builtin's name (like "+" or "in")`,
		``,
		`Shorter goal"s", goal"t", etc. can be used if you haven't overridden them.`,
		``,
		`Notations:`,
		`        i (integer) n (number) s (string) r (regexp)`,
		`        d (dict) t (dict S!Y) h (handle) e (error)`,
		`        f (function) F (dyadic function)`,
		`        x,y,z (any other) I,N,S,X,Y,A (arrays)`,
	}, "\n")
	goalrt := strings.Join([]string{
		"GOAL RUNTIME HELP", rtlog, rtseed, rttime, rtget,
	}, "\n")
	goals := strings.Join([]string{
		`GOAL SYNTAX HELP`,
		`numbers         1     1.5     0b0110     1.7e-3     0xab     0n     0w     3h2m`,
		`strings         "text\x2c\u00FF\n"     "\""     "\u65e5"     "interpolated $var"`,
		"                qq/$var\n or ${var}/   qq#text#  (delimiters :+-*%!&|=~,^#_?@`/)",
		`raw strings     rq/anything until single slash/         rq#doubling ## escapes #`,
		`arrays          1 2 -3 4      1 "ab" -2 "cd"      (1 2;"a";3 "b";(4 2;"c");*)`,
		`regexps         rx/[a-z]/      (see https://pkg.go.dev/regexp/syntax for syntax)`,
		`dyadic verbs    : + - * % ! & | < > = ~ , ^ # _ $ ? @ .     (right-associative)`,
		`monadic verbs   :: +: -: abs uc error ...`,
		`adverbs         / \ ' (alone or after expr. with no space)    (left-associative)`,
		`expressions     2*3+4 → 14       1+|2 3 4 → 5 4 3        +/'(1 2 3;4 5 6) → 6 15`,
		`separator       ; or newline except when ignored after {[( and before )]}`,
		`variables       x  y.z  f  data  t1  π               (. only allowed in globals)`,
		`assign          x:2 (local within lambda, global otherwise)        x::2 (global)`,
		`op assign       x+:1 (sugar for x:x+1 or x::x+1)          x-:2 (sugar for x:x-2)`,
		`list assign     (x;y;z):e (where 2<#e)         (x;y):1 2;y → 2`,
		`eval. order     apply:f[e1;e2]   apply:e1 op e2                      (e2 before)`,
		`                list:(e1;e2)     seq: [e1;e2]     lambda:{e1;e2}     (e1 before)`,
		`sequence        [x:2;y:x+3;x*y] → 10        (bracket not following noun tightly)`,
		`index/apply     x[y] or x y is sugar for x@y; x[] ~ x[*] ~ x[!#x] ~ x (arrays)`,
		`index deep      x[y;z;...] → x.(y;z;...)            (except for x in (?;and;or))`,
		`index assign    x[y]:z → x:@[x;y;:;z]                      (or . for x[y;...]:z)`,
		`index op assign x[y]op:z → x:@[x;y;op;z]                         (for symbol op)`,
		`lambdas         {x+y-z}[3;5;7] → 1        {[a;b;c]a+b-c}[3;5;7] → 1`,
		`                {?[x>1;x*o x-1;1]}5 → 120        (o is recursive self-reference)`,
		`projections     +[2;] 3 → 5               (2+) 3 → 5       (partial application)`,
		`compositions    ~0> → {~0>x}      -+ → {-x+y}      *|: → {*|:x}`,
		`index at field  x..a → x["a"]       (.. binds identifiers tightly, interpolable)`,
		`field expr.     ..a+b → {x["a"]+x["b"]} (field names without . and not in x,y,z)`,
		`                ..p.a+b+q.c → {[p0;x]p0+x["b"]+c}[a;]   (p. projects; q. quotes)`,
		`field expr. at  x.. a+b → {x["a"]+x["b"]}[x]                  (same as (..a+b)x)`,
		`dict fields     ..[a:e1;b:e2;c] → "a""b""c"!(e1;e2;c)`,
		`amend fields    x..[a:e1;b:e2] → @[x;"a""b";:;x..(e1;e2)]`,
		`cond            ?[1;2;3] → 2      ?[0;2;3] → 3     ?[0;2;"";3;4] → 4`,
		`and/or          and[1;2] → 2    and[1;0;3] → 0    or[0;2] → 2      or[0;0;0] → 0`,
		`return          [1;:2;3] → 2                        (a : at start of expression)`,
		`try             'x is sugar for ?["e"~@x;:x;x]         (return if it's an error)`,
		`log             \x logs a string representation of x        (debug/display tool)`,
		`comments        from line with a single / until line with a single \`,
		`                or from / (after space or start of line) to end of line`,
	}, "\n")
	goalt := strings.Join([]string{
		`GOAL TYPES HELP`,
		`atom    array   name            examples`,
		`i       I       integer         0         -2        !5          4 3 -2 5 0i`,
		`n       N       number          0.0       1.5       0.0+!5      1.2 3 0n 1e+10`,
		`s       S       string          "abc"     "d"       "a" "b" "c"`,
		`r               regexp          rx/[a-z]/           rx/\s+/`,
		`d               dictionary      "a""b"!1 2          keys!values`,
		`f               function        +         {x-1}     2*          %[;2]`,
		`h               handle          open"/path/to/file"    "w"open"/path/to/file"`,
		`e               error           error"msg"`,
		`A       generic array   ("a" 1;"b" 2;"c" 3)     (+;-;*;"any")`,
	}, "\n")
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
	importHelp := strings.Join([]string{
		`import s    read/eval wrapper roughly equivalent to eval[read path;path;pfx]`,
		`            where 1) path~s or path~env["GOALLIB"]+s+".goal"`,
		`            2) pfx is path's basename without extension`,
		`x import s  same as import s, but using prefix x for globals`,
	}, "\n")
	in := strings.Join([]string{
		`x in s     contained    "bc" "ac"in"abcd" → 1 0                    (same as x¿s)`,
		`x in Y     member of    2 3 in 8 2 4 → 1 0                         (same as x¿Y)`,
	}, "\n")
	json := strings.Join([]string{
		"json s     parse json   json rq`" + `{"a":true,"b":"text"}` + "`" + ` → "a" "b"!1 "text"`,
		`s json y   write json   ""json 1.5 2 → "[1.5,2]" (indent with s;disable with "")`,
		`S json y   write json   like s json y, but with (pfx;indent) for pretty-printing`,
	}, "\n")
	logHelp := `log n      nat log      log 2.718281828459045 → 1.0`
	mkdir := `mkdir s     create new directory named s (parent must already exist)`
	nan := strings.Join([]string{
		`nan n      isNaN        nan(0n;2;sqrt -1) → 1 0 1             nan 2 0i 3 → 0 1 0`,
		`n nan n    fill NaNs    42.0 nan(1.5;sqrt -1) → 1.5 42.0      42 nan 2 0i → 2 42`,
	}, "\n")
	ocount := `ocount X   occur-count  ocount 3 4 5 3 4 4 7 → 0 0 0 1 1 2 0`
	open := strings.Join([]string{
		"open s      open path s for reading, returning a handle (h)", `x open s    open path s with mode x in "r" "w" "a"`,
		`            or pipe from (x~"-|") or to (x~"|-") command s or S`,
	}, "\n")
	or := `or[0;2] → 2      or[0;0;0] → 0`
	panicHelp := `panic s    panic        panic"msg"                (for fatal programming-errors)`
	printHelp := strings.Join([]string{
		`print s     print"Hello, world!\n"      (uses implicit  for non-string values)`,
		`x print s   print s to handle/name x               "/path/to/file"print"content"`,
	}, "\n")
	read := strings.Join([]string{
		`read h      read from handle h until EOF or an error occurs`,
		`read s      read file named s into string         lines:=-read"/path/to/file"`,
		`            or "name""dir"!(S;I) if s is a directory`,
		`i read h    read i bytes from reader h or until EOF, or an error occurs`,
		`s read h    read from reader h until 1-byte s, EOF, or an error occurs`,
	}, "\n")
	remove := `remove s    remove the named file or empty directory`
	rename := `x rename y  renames (moves) old path x (s) to new path y (s)`
	rotate := `i rotate Y rotate       2 rotate 7 8 9 → 9 7 8           -2 rotate 7 8 9 → 8 9 7`
	round := `round      round n      round 3.4 → 3.0    round 3.5 → 4.0`
	rshift := strings.Join([]string{
		`»X  rshift      »8 9 → 0 8     »"a" "b" → "" "a"       (ASCII keyword: rshift x)`,
		`x»Y rshift      "a" "b"»1 2 3 → "a" "b" 1`,
	}, "\n")
	run := strings.Join([]string{
		`run s       run command s or S (with arguments)   run"pwd"          run"ls" "-l"`,
		`            inherits stdin and stderr, returns its standard output or an error`,
		`            dict with keys "code" "msg" "out"`,
		`x run s     same as run s but with input string x as stdin`,
	}, "\n")
	rx := `rx s       comp. regex  rx"[a-z]"       (like rx/[a-z]/ but compiled at runtime)`
	say := strings.Join([]string{
		`say s       same as print, but appends a newline                    say!5`,
		`x say s     same as print, but appends a newline`,
	}, "\n")
	shell := `shell s     same as run, but through "/bin/sh" (unix systems only)  shell"ls -l"`
	shift := strings.Join([]string{
		`«X  shift       «8 9 → 9 0     «"a" "b" → "b" ""        (ASCII keyword: shift x)`,
		`x«Y shift       "a" "b"«1 2 3 → 3 "a" "b"`,
	}, "\n")
	sign := `sign n     sign         sign -3 -1 0 1.5 5 → -1 -1 0 1 1`
	sin := `sin n      sin        sin 3.141592653589793%2 → 1.0`
	sqlopen := `sql.open s    Open DuckDB database with data source name s`
	sqlq := `sql.q s    Run SQL query, results as table.`
	sqrt := `sqrt n     sqrt 9 → 3.0    sqrt -1 → 0n`
	stat := `stat x      returns "dir""mtime""size"!(i;i;i)      (for filehandle h or path s)`
	sub := strings.Join([]string{
		`sub[r;s]   regsub       sub[rx/[a-z]/;"Z"] "aBc" → "ZBZ"`,
		`sub[r;f]   regsub       sub[rx/[A-Z]/;_] "aBc" → "abc"`,
		`sub[s;s]   replace      sub["b";"B"] "abc" → "aBc"`,
		`sub[s;s;i] replaceN     sub["a";"b";2] "aaa" → "bba"        (stop after 2 times)`,
		`sub[S]     replaceS     sub["b" "d" "c" "e"] "abc" → "ade"`,
		`sub[S;S]   replaceS     sub["b" "c";"d" "e"] "abc" → "ade"`,
	}, "\n")
	time := strings.Join([]string{
		`time cmd              time command with current time`,
		`cmd time t            time command with time t`,
		`time[cmd;t;fmt]       time command with time t in given format`,
		`time[cmd;t;fmt;loc]   time command with time t in given format and location`,
	}, "\n")
	uc := `uc x       upper/ceil   uc 1.5 → 2.0                             uc"abπ" → "ABΠ"`
	utf8 := strings.Join([]string{
		`utf8 s     is UTF-8     utf8 "aπc" → 1                          utf8 "a\xff" → 0`,
		`s utf8 s   to UTF-8     "b" utf8 "a\xff" → "ab"       (replace invalid with "b")`,
	}, "\n")
	// Adverbs
	goalAdverbHelpEntries := goalAdverbsHelp()
	goalaHelps := make([]string, 0, len(goalAdverbHelpEntries))
	for _, entry := range goalAdverbHelpEntries {
		goalaHelps = append(goalaHelps, entry.v)
	}
	goala := strings.Join(goalaHelps, "\n")
	// Verbs
	goalVerbHelpEntries := goalVerbsHelp()
	goalvHelps := make([]string, 0, len(goalVerbHelpEntries))
	for _, entry := range goalVerbHelpEntries {
		goalvHelps = append(goalvHelps, entry.v)
	}
	goalv := strings.Join(goalvHelps, "\n")
	goalnv := strings.Join([]string{
		"GOAL NAMED VERBS HELP", abs, atan, cos, csv, errorHelp, exp, eval, firsts, in,
		json, logHelp, nan, ocount, panicHelp, rotate, round, rx, sign, sin, sqrt, sub, uc, utf8,
	}, "\n")
	goalio := strings.Join([]string{
		"GOAL IO/OS HELP", chdir, closeHelp, env, flush, importHelp, mkdir, open,
		printHelp, read, remove, run, say, shell, stat, "",
		strings.Join([]string{
			`ARGS        command-line arguments, starting with script name`,
			`STDIN       standard input filehandle (buffered)`,
			`STDOUT      standard output filehandle (buffered)`,
			`STDERR      standard error filehandle (buffered)`,
		}, "\n"),
	}, "\n")
	goaltm := strings.Join([]string{
		"GOAL TIME HELP", time,
		`Time t should consist either of integers or strings in the given format ("unix"`,
		`is the default for integers and RFC3339 layout "2006-01-02T15:04:05Z07:00" is`,
		`the default for strings), with optional location (default is "UTC"). See`,
		`https://pkg.go.dev/time for information on layouts and locations, as Goal uses`,
		`the same conventions as Go's time package.  Supported values for cmd are as`,
		`follows:`,
		``,
		`    cmd (s)       result (type)                                 fmt`,
		`    -------       -------------                                 ---`,
		`    "clock"       hour, minute, second (I)`,
		`    "date"        year, month, day (I)                          yes`,
		`    "day"         day number (i)`,
		`    "hour"        0-23 hour (i)`,
		`    "minute"      0-59 minute (i)`,
		`    "month"       0-12 month (i)`,
		`    "second"      0-59 second (i)`,
		`    "unix"        unix epoch time (i)                           yes`,
		`    "unixmicro"   unix (microsecond version) (i)                yes`,
		`    "unixmilli"   unix (millisecond version) (i)                yes`,
		`    "unixnano"    unix (nanosecond version) (i)                 yes`,
		`    "week"        year, week (I)`,
		`    "weekday"     0-6 weekday starting from Sunday (i)`,
		`    "year"        year (i)`,
		`    "yearday"     1-365/6 year day (i)`,
		`    "zone"        name, offset in seconds east of UTC (s;i)`,
		`    format (s)    format time using given layout (s)            yes`,
	}, "\n")
	return map[string]string{
		"abs":          abs,
		"and":          and,
		"atan":         atan,
		"chdir":        chdir,
		"close":        closeHelp,
		"cos":          cos,
		"csv":          csv,
		"env":          env,
		"error":        errorHelp,
		"eval":         eval,
		"exp":          exp,
		"firsts":       firsts,
		"flush":        flush,
		"help":         help,
		"goal":         goalHelp,
		"goal:a":       goala,
		"goal:io":      goalio,
		"goal:nv":      goalnv,
		"goal:rt":      goalrt,
		"goal:s":       goals,
		"goal:t":       goalt,
		"goal:tm":      goaltm,
		"goal:v":       goalv,
		"http.client":  httpclient,
		"http.delete":  helpForHTTPFn("delete"),
		"http.get":     helpForHTTPFn("get"),
		"http.head":    helpForHTTPFn("head"),
		"http.options": helpForHTTPFn("options"),
		"http.patch":   helpForHTTPFn("patch"),
		"http.post":    helpForHTTPFn("post"),
		"http.put":     helpForHTTPFn("put"),
		"import":       importHelp,
		"in":           in,
		"json":         json,
		"log":          logHelp,
		"mkdir":        mkdir,
		"nan":          nan,
		"ocount":       ocount,
		"open":         open,
		"or":           or,
		"panic":        panicHelp,
		"print":        printHelp,
		"read":         read,
		"remove":       remove,
		"rename":       rename,
		"rotate":       rotate,
		"round":        round,
		"rshift":       rshift,
		"rt.get":       rtget,
		"rt.log":       rtlog,
		"rt.seed":      rtseed,
		"rt.time":      rttime,
		"run":          run,
		"rx":           rx,
		"say":          say,
		"shell":        shell,
		"shift":        shift,
		"sign":         sign,
		"sin":          sin,
		"sql.open":     sqlopen,
		"sql.q":        sqlq,
		"sqrt":         sqrt,
		"stat":         stat,
		"sub":          sub,
		"time":         time,
		"uc":           uc,
		"utf8":         utf8,
	}
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
