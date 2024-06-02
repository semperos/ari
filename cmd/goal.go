package cmd

import (
	"fmt"
	"os"
	"sort"

	"codeberg.org/anaseto/goal"
)

var goalKeywordsFromCtx []string

func goalKeywords(ctx *goal.Context) []string {
	if goalKeywordsFromCtx == nil {
		goalKeywordsFromCtx = ctx.Keywords(nil)
		sort.Strings(goalKeywordsFromCtx)
	}
	return goalKeywordsFromCtx
}

var goalSyntax = map[string]string{
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
	"shift":          "«",
	"lshift":         "«",
	"firsts":         "¿",
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
}

// TODO See how hard it is to change the bubbline help prefix of "<match>:"
var goalSyntaxHelp = map[string]string{
	"'":  `f'x    each      #'(4 5;6 7 8) → 2 3` + "\n" + `x F'y  each      2 3#'4 5 → (4 4;5 5 5)      {(x;y;z)}'[1;2 3;4] → (1 2 4;1 3 4)` + "\n" + `x I'y  case      (6 7 8 9)0 1 0 1'"a""b""c""d" → 6 "b" 8 "d"` + "\n" + `I A'I  at each   m:3$!9;p:!2 2;(m').p → 0 1 3 4`,
	"`":  "x F`" + `y  eachleft  1 2,` + "`" + `"a""b" → (1 "a" "b";2 "a" "b")           (same as F[;y]'x)`,
	"´":  `x F` + "´y  eachright 1 2,´" + `"a""b" → (1 2 "a";1 2 "b")               (same as F[x;]'y)`,
	"/":  `F/x    fold      +/!10 → 45` + "\n" + `x F/y  fold      5 6+/!4 → 11 12                     {x+y-z}/[5;4 3;2 1] → 9` + "\n" + `i f/y  do        3(2*)/4 → 32` + "\n" + `f f/y  while     (100>)(2*)/4 → 128` + "\n" + `f/x    converge  {1+1.0%x}/1 → 1.618033988749895     {-x}/1 → -1` + "\n" + `s/S    join      ","/"a" "b" "c" → "a,b,c"` + "\n" + `I/x    decode    24 60 60/1 2 3 → 3723               2/1 1 0 → 6`,
	`\`:  `F\x    scan      +\!10 → 0 1 3 6 10 15 21 28 36 45` + "\n" + `x F\y  scan      5 6+\!4 → (5 6;6 7;8 9;11 12)       {x+y-z}\[5;4 3;2 1] → 7 9` + "\n" + `i f\y  dos       3(2*)\4 → 4 8 16 32` + "\n" + `f f\y  whiles    (100>)(2*)\4 → 4 8 16 32 64 128` + "\n" + `f\x    converges (-2!)\10 → 10 5 2 1 0               {-x}\1 → 1 -1` + "\n" + `s\s    split     ","\"a,b,c" → "a" "b" "c"           ""\"aπc" → "a" "π" "c"` + "\n" + `r\s    split     rx/[,;]/\"a,b;c,d" → "a" "b" "c" "d"` + "\n" + `i s\s  splitN    (2)","\"a,b,c" → "a" "b,c"` + "\n" + `i r\s  splitN    (3)rx/[,;]/\"a,b;c,d" → "a" "b" "c,d"` + "\n" + `I\x    encode    24 60 60\3723 → 1 2 3               2\6 → 1 1 0`,
	"¿":  `firsts X   mark firsts  firsts 0 0 2 3 0 2 3 4 → 1 0 1 1 0 0 0 1    (same as ¿X)` + "\n" + `x in s     contained    "bc" "ac"in"abcd" → 1 0                    (same as x¿s)` + "\n" + `x in Y     member of    2 3 in 8 2 4 → 1 0                         (same as x¿Y)`,
	":":  `:x  identity    :[42] → 42  (recall that : is also syntax for return and assign)` + "\n" + `x:y right       2:3 → 3                        "a":"b" → "b"`,
	"+":  `+d  swap k/v    +"a""b"!0 1 → 0 1!"a" "b"` + "\n" + `+x  flip        +(1 2;3 4) → (1 3;2 4)         +42 → ,,42` + "\n" + `n+n add         2+3 → 5                        2+3 4 → 5 6` + "\n" + `s+s concat      "a"+"b" → "ab"                 "a" "b"+"c" → "ac" "bc"`,
	"-":  `-n  negate      - 2 3 → -2 -3                  -(1 2.5;3 4) → (-1.0 -2.5;-3 -4)` + "\n" + `-s  rtrim space -"a\tb \r\n" " c d \n" → "a\tb" " c d"   (Unicode's White Space)` + "\n" + `n-n subtract    5-3 → 2                        5 4-3 → 2 1` + "\n" + `s-s trim suffix "file.txt"-".txt" → "file"`,
	"*":  `*x  first       *7 8 9 → 7                     *"ab" → "ab"           *(+;*) → +` + "\n" + `n*n multiply    2*3 → 6                        1 2 3*3 → 3 6 9` + "\n" + `s*i repeat      "a"*3 2 1 0 → "aaa" "aa" "a" ""`,
	"%":  `%X  classify    %7 8 9 7 8 9 → 0 1 2 0 1 2     %"a" "b" "a" → 0 1 0` + "\n" + `n%n divide      3%2 → 1.5                      3 4%2 → 1.5 2.0`,
	"!":  `!i  enum        !5 → 0 1 2 3 4                 !-5 → -5 -4 -3 -2 -1` + "\n" + `!s  fields      !"a b\tc\nd \u00a0e" → "a""b""c""d""e"   (Unicode's White Space)` + "\n" + `!I  odometer    !2 3 → (0 0 0 1 1 1;0 1 2 0 1 2)` + "\n" + `!d  keys        !"a""b"!1 2 → "a" "b"` + "\n" + `i!n mod/div     3!9 8 7 → 0 2 1            -3!9 8 7 → 3 2 2` + "\n" + `i!s pad fields  3!"a" → "a  "              -3!"1" "23" "456" → "  1" " 23" "456"` + "\n" + `s!s fields      ",;"!"a,b;c" → "a""b""c" (fields cut on any of ",;"; ""!s is !s)` + "\n" + `X!Y dict        d:"a""b"!1 2; d"a" → 1              (same as d:..[a:1;b:2]; d..a)`,
	"&":  `&s  byte-count  &"abc" → 3                      &"π" → 2              &"αβγ" → 6` + "\n" + `&I  where       &0 0 1 0 0 0 1 → 2 6            &2 3 → 0 0 1 1 1` + "\n" + `&d  keys where  &"a""b""c""d"!0 1 1 0 → "b" "c"` + "\n" + `x&y min/and     2&3 → 2          4&3 → 3         "b"&"a" → "a"           0&1 → 0`,
	"|":  `|X  reverse     |!5 → 4 3 2 1 0` + "\n" + `x|y max/or      2|3 → 3          4|3 → 4         "b"|"a" → "b"           0|1 → 1`,
	"<":  `<d  sort up     <"a""b""c"!2 3 1 → "c""a""b"!1 2 3` + "\n" + `<X  ascend      <3 5 4 → 0 2 1           (index permutation for ascending order)` + "\n" + `x<y less        2<3 → 1          "c"<"a" → 0                       7 8<6 9 → 0 1`,
	">":  `>d  sort down   >"a""b""c"!2 3 1 → "b""a""c"!3 2 1` + "\n" + `>X  descend     >3 5 4 → 1 2 0          (index permutation for descending order)` + "\n" + `x>y more        2>3 → 0          "c">"a" → 1                       7 8>6 9 → 1 0`,
	"=":  `=s  lines       ="ab\ncd\r\nef gh" → "ab" "cd" "ef gh"` + "\n" + `=I  index-count =1 0 0 2 2 3 -1 2 1 1 1 → 2 4 3 1` + "\n" + `=d  group keys  ="a""b""c"!0 1 0 → ("a" "c";,"b")           ="a""b"!0 -1 → ,,"a"` + "\n" + `f=Y group by    (2!)=!10 → (0 2 4 6 8;1 3 5 7 9)` + "\n" + `x=y equal       2 3 4=3 → 0 1 0                          "ab" = "ba" → 0`,
	"~":  `~x  not         ~0 1 2 → 1 0 0                           ~"a" "" "0" → 0 1 0` + "\n" + `x~y match       3~3 → 1            2 3~3 2 → 0           ("a";%)~'("b";%) → 0 1`,
	",":  `,x  enlist      ,1 → ,1            #,2 3 → 1             (list with one element)` + "\n" + `d,d merge       ("a""b"!1 2),"b""c"!3 4 → "a""b""c"!1 3 4` + "\n" + `x,y join        1,2 → 1 2                       "ab" "c","d" → "ab" "c" "d"`,
	"^":  `^d  sort keys   ^"c""a""b"!1 2 3 → "a""b""c"!2 3 1` + "\n" + `^X  sort        ^3 5 0 → 0 3 5                  ^"ca" "ab" "bc" → "ab" "bc" "ca"` + "\n" + `i^s windows     2^"abcde" → "abcd" "bcde"` + "\n" + `i^Y windows     2^!4 → (0 1 2;1 2 3)                   -2^!4 → (0 1;1 2;2 3)` + "\n" + `s^s trim        " #"^"  #a ## b#  " → "a ## b"         ""^" \na\t b\t" → "a\t b"` + "\n" + `f^y weed out    {0 1 1 0}^4 1 5 3 → 4 3                (0<)^2 -3 1 → ,-3` + "\n" + `X^t w/o keys &  (,"b";1 0)^..[a:6 7;b:8 9] → (,"a")!,,7` + "\n" + `                (0;..a>1;..b<0)^..[a:1 2 3;b:4 -5 6] → "a""b"!(,1;,4)` + "\n" + `X^d w/o keys    (,"b")^"a""b""c"!0 1 2 → "a""c"!0 2` + "\n" + `X^Y w/o values  2 3^1 1 2 3 3 4 → 1 1 4                          (like in[;X]^Y)`,
	"#":  `#x  length      #7 8 9 → 3        #"ab" "cd" → 2       #42 → 1      #"ab" → 1` + "\n" + `i#y take/repeat 2#6 7 8 → 6 7     5#6 7 8 → 6 7 8 6 7               3#1 → 1 1 1` + "\n" + `s#s count       "ab"#"cabdab" "cd" "deab" → 2 0 1                   ""#"αβγ" → 4` + "\n" + `f#y replicate   {0 1 2 0}#4 1 5 3 → 1 5 5              (0<)#2 -3 1 → 2 1` + "\n" + `X#t w/ keys &   (,"a";0 1)#..[a:6 7;b:8 9] → (,"a")!,,7` + "\n" + `                (1;..a>1;..b>0)#..[a:1 2 3;b:4 -5 6] → "a""b"!(,3;,6)` + "\n" + `X#d with keys   "a""c""e"#"a""b""c""a"!2 3 4 5 → "a""c""a"!2 4 5` + "\n" + `X#Y with values 2 3#1 1 2 3 3 4 → 2 3 3                          (like in[;X]#Y)`,
	"_":  `_n  floor       _2.3 → 2.0                  _1.5 3.7 → 1.0 3.0` + "\n" + `_s  to lower    _"ABC" → "abc"              _"AB" "CD" "Π" → "ab" "cd" "π"` + "\n" + `i_s drop bytes  2_"abcde" → "cde"           -2_"abcde" → "abc"` + "\n" + `i_Y drop        2_3 4 5 6 → 5 6             -2_3 4 5 6 → 3 4` + "\n" + `s_s trim prefix "pfx-"_"pfx-name" "pfx2-name" → "name" "pfx2-name"` + "\n" + `f_Y cut where   {0=3!x}_!10 → (0 1 2;3 4 5;6 7 8;,9)          (same as (&f Y)_Y)` + "\n" + `I_s cut string  1 3_"abcdef" → "bc" "def"                          (I ascending)` + "\n" + `I_Y cut         2 5_!10 → (2 3 4;5 6 7 8 9)                        (I ascending)`,
	"$":  `$x  string      $2 3 → "2 3"      $"text" → "\"text\""` + "\n" + `i$s cut shape   3$"abcdefghijk" → "abc" "defg" "hijk"` + "\n" + `i$Y cut shape   3$!6 → (0 1;2 3;4 5)             -3$!6 → (0 1 2;3 4 5)` + "\n" + `s$y strings     "s"$(1;"c";+) → "1""c""+"` + "\n" + `s$s chars/bytes "c"$"aπ" → 97 960                "b"$"aπ" → 97 207 128` + "\n" + `s$i to string   "c"$97 960 → "aπ"                "b"$97 207 128 → "aπ"` + "\n" + `s$n cast        "i"$2.3 → 2                      @"n"$42 → "n"` + "\n" + `s$s parse i/n   "i"$"42" "0b100" → 42 4          "n"$"2.5" "1e+20" → 2.5 1e+20` + "\n" + `s$s parse value "v"$qq/(2 3;"a")/ → (2 3;"a")   ($x inverse for types in "inrs")` + "\n" + `s$y format      "%.2g"$1 4%3 → "0.33" "1.3"      "%s=%03d"$"a" 42 → "a=042"` + "\n" + `X$y binsearch   2 3 5 7$8 2 7 5 5.5 3 0 → 4 1 4 3 3 2 0            (X ascending)`,
	"?":  `?i  uniform     ?2 → 0.6046602879796196 0.9405090880450124     (between 0 and 1)` + "\n" + `?i  normal      ?-2 → -1.233758177597947 -0.12634751070237293    (mean 0, dev 1)` + "\n" + `?X  distinct    ?2 2 4 3 2 3 → 2 4 3                   (keeps first occurrences)` + "\n" + `i?i roll        5?100 → 10 51 21 51 37` + "\n" + `i?Y roll array  5?"a" "b" "c" → "c" "a" "c" "c" "b"` + "\n" + `i?i deal        -5?100 → 19 26 0 73 94                         (always distinct)` + "\n" + `i?Y deal array  -3?"a""b""c" → "a""c""b"     (0i?Y is (-#Y)?Y) (always distinct)` + "\n" + `s?r rindex      "abcde"?rx/b../ → 1 3                            (offset;length)` + "\n" + `s?s index       "a = a + 1"?"=" "+" → 2 6` + "\n" + `d?y find key    ("a""b"!3 4)?4 → "b"                    ("a" "b"!3 4)?5 → ""` + "\n" + `X?y find        9 8 7?8 → 1                              9 8 7?6 → 3`,
	"@":  `@x  type        @2 → "i"    @1.5 → "n"    @"ab" → "s"    @2 3 → "I"     @+ → "f"` + "\n" + `i@y take/pad    2@6 7 8 → 6 7     4@6 7 8 → 6 7 8 0      -4@6 7 8 → 0 6 7 8` + "\n" + `s@i substr      "abcdef"@2  → "cdef"                                 (s[offset])` + "\n" + `r@s match       rx/^[a-z]+$/"abc" → 1                    rx/\s/"abc" → 0` + "\n" + `r@s find group  rx/([a-z])(.)/"&a+c" → "a+" "a" "+"      (whole match, group(s))` + "\n" + `f@y apply at    (|)@1 2 → 2 1            (like |[1 2] → 2 1 or |1 2 or (|).,1 2)` + "\n" + `d@y at key      ..[a:6 7;b:8 9]"a" → 6 7                 (1 2!"a""b")2 → "b"` + "\n" + `t@i at row      ..[a:6 7;b:8 9]0 → "a""b"!6 8` + "\n" + `X@i at          7 8 9@2 → 9             7 8 9[2 0] → 9 7            7 8 9@-2 → 8`,
	".":  `.s  get global  a:3;."a" → 3` + "\n" + `.e  get error   .error"msg" → "msg"` + "\n" + `.d  values      ."a""b"!1 2 → 1 2` + "\n" + `.X  self-dict   ."a""b" → "a""b"!"a""b"          .!3 → 0 1 2!0 1 2` + "\n" + `s.I substr      "abcdef"[2;3] → "cde"                         (s[offset;length])` + "\n" + `r.y findN       rx/[a-z]/["abc";2] → "a""b"      rx/[a-z]/["abc";-1] → "a""b""c"` + "\n" + `r.y findN group rx/[a-z](.)/["abcdef";2] → ("ab" "b";"cd" "d")` + "\n" + `f.y apply       (+).2 3 → 5                      +[2;3] → 5` + "\n" + `d.y deep at     ..[a:6 7;b:8 9]["a";1] → 7` + "\n" + `t.y at row;key  ..[a:6 7;b:8 9][1;"a"] → (,"a")!,7` + "\n" + `X.y deep at     (6 7;8 9)[0;1] → 7               (6 7;8 9)[;1] → 7 9`,
	"«":  `«X  shift       «8 9 → 9 0     «"a" "b" → "b" ""        (ASCII keyword: shift x)` + "\n" + `x«Y shift       "a" "b"«1 2 3 → 3 "a" "b"`,
	"»":  `»X  rshift      »8 9 → 0 8     »"a" "b" → "" "a"       (ASCII keyword: rshift x)` + "\n" + `x»Y rshift      "a" "b"»1 2 3 → "a" "b" 1`,
	"::": `::[s;y]     set global  ::["a";3];a → 3  (brackets needed because :: is monadic)`,
	"@[": `@[d;y;f]    amend       @["a""b""c"!7 8 9;"a""b""b";10+] → "a""b""c"!17 28 9` + "\n" + `@[X;i;f]    amend       @[7 8 9;0 1 1;10+] → 17 28 9` + "\n" + `@[d;y;F;z]  amend       @["a""b""c"!7 8 9;"a";:;42] → "a""b""c"!42 8 9` + "\n" + `@[X;i;F;z]  amend       @[7 8 9;1 2 0;+;10 20 -10] → -3 18 29` + "\n" + `@[f;x;f]    try at      @[2+;3;{"msg"}] → 5           @[2+;"a";{"msg"}] → "msg"`,
	".[": `.[X;y;f]    deep amend  .[(6 7;8 9);0 1;-] → (6 -7;8 9)` + "\n" + `.[X;y;F;z]  deep amend  .[(6 7;8 9);(0 1 0;1);+;10] → (6 27;8 19)` + "\n" + `                        .[(6 7;8 9);(*;1);:;42] → (6 42;8 42)` + "\n" + `.[f;x;f]    try         .[+;2 3;{"msg"}] → 5          .[+;2 "a";{"msg"}] → "msg"`,
}

var goalGlobalsHelp = map[string]string{
	"STDERR": "standard error filehandle (buffered)",
	"STDIN":  "standard input filehandle (buffered)",
	"STDOUT": "standard output filehandle (buffered)",
}

var goalKeywordsHelp = map[string]string{
	"abs":     "abs n    abs -3.0 -1.5 2.0 → 3.0 1.5 2.0",
	"and":     "and[1;2] → 2    and[1;0;3] → 0",
	"atan":    "atan[n;n]",
	"chdir":   "chdir s     change current working directory to s, or return an error",
	"close":   "close h     flush any buffered data, then close handle h",
	"cos":     "cos n",
	"csv":     `csv s      csv read     csv"a,b\n1,2" → ("a" "1";"b" "2")` + "\n" + `csv A      csv write    csv("a" "b";1 2) → "a,1\nb,2\n"` + "\n" + `s csv s    csv read     " "csv"a b\n1 2" → ("a" "1";"b" "2")  (" " as separator)` + "\n" + `s csv A    csv write    " "csv("a" "b";1 2) → "a 1\nb 2\n"    (" " as separator)`,
	"env":     `env s       get environment variable s, or an error if unset` + "\n" + `            return a dictionary representing the whole environment if s~""` + "\n" + `x env s     set environment variable x to s, or return an error` + "\n" + `x env 0     unset environment variable x, or clear environment if x~""`,
	"error":   `error x    error        r:error"msg"; (@r;.r) → "e" "msg"`,
	"eval":    `eval s     comp/run     a:5;eval"a+2" → 7           (unrestricted variant of .s)` + "\n" + `eval[s;loc;pfx]         like eval s, but provide loc as location (usually a` + "\n" + `                        path), and prefix pfx+"." for globals; does not eval` + "\n" + `                        same location more than once`,
	"exp":     "exp n",
	"firsts":  `firsts X   mark firsts  firsts 0 0 2 3 0 2 3 4 → 1 0 1 1 0 0 0 1    (same as ¿X)`,
	"flush":   "flush h     flush any buffered data for handle h",
	"import":  `import s    read/eval wrapper roughly equivalent to eval[read path;path;pfx]` + "\n" + `            where 1) path~s or path~env["GOALLIB"]+s+".goal"` + "\n" + `            2) pfx is path's basename without extension` + "\n" + `x import s  same as import s, but using prefix x for globals`,
	"in":      `x in s     contained    "bc" "ac"in"abcd" → 1 0                    (same as x¿s)` + "\n" + `x in Y     member of    2 3 in 8 2 4 → 1 0                         (same as x¿Y)`,
	"json":    "json s     parse json   json rq`" + `{"a":true,"b":"text"}` + "`" + ` → "a" "b"!1 "text"` + "\n" + `s json y   write json   ""json 1.5 2 → "[1.5,2]" (indent with s;disable with "")` + "\n" + `S json y   write json   like s json y, but with (pfx;indent) for pretty-printing`,
	"log":     "log n",
	"mkdir":   "mkdir s     create new directory named s (parent must already exist)",
	"nan":     `nan n      isNaN        nan(0n;2;sqrt -1) → 1 0 1             nan 2 0i 3 → 0 1 0` + "\n" + `n nan n    fill NaNs    42.0 nan(1.5;sqrt -1) → 1.5 42.0      42 nan 2 0i → 2 42`,
	"ocount":  `ocount X   occur-count  ocount 3 4 5 3 4 4 7 → 0 0 0 1 1 2 0`,
	"open":    "open s      open path s for reading, returning a handle (h)" + "\n" + `x open s    open path s with mode x in "r" "w" "a"` + "\n" + `            or pipe from (x~"-|") or to (x~"|-") command s or S`,
	"or":      "or[0;2] → 2      or[0;0;0] → 0",
	"panic":   `panic s    panic        panic"msg"                (for fatal programming-errors)`,
	"print":   `print s     print"Hello, world!\n"      (uses implicit $x for non-string values)` + "\n" + `x print s   print s to handle/name x               "/path/to/file"print"content"`,
	"read":    `read h      read from handle h until EOF or an error occurs` + "\n" + `read s      read file named s into string         lines:=-read"/path/to/file"` + "\n" + `            or "name""dir"!(S;I) if s is a directory` + "\n" + `i read h    read i bytes from reader h or until EOF, or an error occurs` + "\n" + `s read h    read from reader h until 1-byte s, EOF, or an error occurs`,
	"remove":  "remove s    remove the named file or empty directory",
	"rename":  `x rename y  renames (moves) old path x (s) to new path y (s)`,
	"rotate":  `i rotate Y rotate       2 rotate 7 8 9 → 9 7 8           -2 rotate 7 8 9 → 8 9 7`,
	"round":   "round n",
	"rshift":  `»X  rshift      »8 9 → 0 8     »"a" "b" → "" "a"       (ASCII keyword: rshift x)` + "\n" + `x»Y rshift      "a" "b"»1 2 3 → "a" "b" 1`,
	"rt.get":  `rt.get s        returns various kinds of runtime information` + "\n" + `                "g"   dictionary with copy of all global variables` + "\n" + `                "f"   same but only globals containing functions` + "\n" + `                "v"   same but only non-function globals`,
	"rt.log":  `rt.log x        like :[x] but logs string representation of x       (same as \x)`,
	"rt.ofs":  `rt.ofs s        set output field separator for print S and "$S"    (default " ")` + "\n" + `                returns previous value`,
	"rt.seed": "rt.seed i       set non-secure pseudo-rand seed to i        (used by the ? verb)",
	"rt.time": `rt.time[s;i]    eval s for i times (default 1), return average time (ns)` + "\n" + `rt.time[f;x;i]  call f. x for i times (default 1), return average time (ns)`,
	"run":     `run s       run command s or S (with arguments)   run"pwd"          run"ls" "-l"` + "\n" + `            inherits stdin and stderr, returns its standard output or an error` + "\n" + `            dict with keys "code" "msg" "out"` + "\n" + `x run s     same as run s but with input string x as stdin`,
	"rx":      `rx s       comp. regex  rx"[a-z]"       (like rx/[a-z]/ but compiled at runtime)`,
	"say":     `say s       same as print, but appends a newline                    say!5` + "\n" + `x say s     same as print, but appends a newline`,
	"shell":   `shell s     same as run, but through "/bin/sh" (unix systems only)  shell"ls -l"`,
	"shift":   `«X  shift       «8 9 → 9 0     «"a" "b" → "b" ""        (ASCII keyword: shift x)` + "\n" + `x«Y shift       "a" "b"«1 2 3 → 3 "a" "b"`,
	"sign":    `sign n     sign         sign -3 -1 0 1.5 5 → -1 -1 0 1 1`,
	"sin":     "sin n",
	"sql.q":   "sql.q s    Run SQL query, results as table.",
	"sqrt":    "sqrt n",
	"stat":    `stat x      returns "dir""mtime""size"!(i;i;i)      (for filehandle h or path s)`,
	"sub":     `sub[r;s]   regsub       sub[rx/[a-z]/;"Z"] "aBc" → "ZBZ"` + "\n" + `sub[r;f]   regsub       sub[rx/[A-Z]/;_] "aBc" → "abc"` + "\n" + `sub[s;s]   replace      sub["b";"B"] "abc" → "aBc"` + "\n" + `sub[s;s;i] replaceN     sub["a";"b";2] "aaa" → "bba"        (stop after 2 times)` + "\n" + `sub[S]     replaceS     sub["b" "d" "c" "e"] "abc" → "ade"` + "\n" + `sub[S;S]   replaceS     sub["b" "c";"d" "e"] "abc" → "ade"`,
	"time":    `time cmd              time command with current time` + "\n" + `cmd time t            time command with time t` + "\n" + `time[cmd;t;fmt]       time command with time t in given format` + "\n" + `time[cmd;t;fmt;loc]   time command with time t in given format and location`,
	"uc":      `uc x       upper/ceil   uc 1.5 → 2.0                             uc"abπ" → "ABΠ"`,
	"utf8":    `utf8 s     is UTF-8     utf8 "aπc" → 1                          utf8 "a\xff" → 0` + "\n" + `s utf8 s   to UTF-8     "b" utf8 "a\xff" → "ab"       (replace invalid with "b")`,
}

// Goal Extensions in Go

// Goal Preamble in Goal

func goalLoadExtendedPreamble(ctx *goal.Context) goal.V {
	val, err := ctx.Eval(goalFmtSource)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load Goal fmt code, error: %v", err)
	}
	return val
}

const goalFmtSource = `
/
  Copyright (c) 2022 Yon <anaseto@bardinflor.perso.aquilenet.fr>

  Permission to use, copy, modify, and distribute this software for any
  purpose with or without fee is hereby granted, provided that the above
  copyright notice and this permission notice appear in all copies.

  THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
  WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
  MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
  ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
  WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
  ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
  OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
\
/
  dict[d;f] outputs dict d, assuming string keys and atom or flat array values,
  using format string f for floating point numbers.
\
fmt.dict:{[d;f]
 nk:#d; say"=== Dict ($nk keys) ==="
 k:(|/-1+""#k)!k:!d; v:" "/(..?[(@x)¿"nN";p.f$x;$'x])'.d
 say"\n"/k{"$x| $y"}'v
}
/
  tbl[t;r;c;f] outputs dict t as table, assuming string keys and flat columns,
  and outputs at most r rows and c columns, using format string f for floating
  point numbers. Example: tbl[t;5;8;"%.1f"].
\
fmt.tbl:{[t;r;c;f]
  (nr;nc):(#*t;#t); say"=== Table ${nr}x$nc ==="
  t:t[!r&nr;(c&nc)@!t] / keep up to r rows and c columns
  k:!t; v:(..?[(@x)¿"nN";p.f$x;$'x])'.t; w:(-1+""#k)|(|/-1+""#)'v
  (k;v):(-w)!'´(k;v); say"\n"/,/" "/(k;"-"*w;+v)
}
`
