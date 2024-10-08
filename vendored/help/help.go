// Code generated by scripts/help.goal. DO NOT EDIT.

package help

const helpTopics = "TOPICS HELP\nType help TOPIC or h TOPIC where TOPIC is one of:\n\n\"s\"     syntax\n\"t\"     value types\n\"v\"     verbs (like +*-%,)\n\"nv\"    named verbs (like in, sign)\n\"a\"     adverbs (/\\')\n\"tm\"    time handling\n\"rt\"    runtime system\n\"io\"    IO verbs (like say, open, read)\nop      where op is a builtin’s name (like \"+\" or \"in\")\n\nNotations:\n        i (integer) n (number) s (string) r (regexp)\n        d (dict) t (dict S!Y) h (handle) e (error)\n        f (function) F (dyadic function)\n        x,y,z (any other) I,N,S,X,Y,A (arrays)\n"

const helpSyntax = "SYNTAX HELP\nnumbers         1     1.5     0b0110     1.7e-3     0xab     0n     0w     3h2m\nstrings         \"text\\x2c\\u00FF\\n\"     \"\\\"\"     \"\\u65e5\"     \"interpolated $var\"\n                qq/$var\\n or ${var}/   qq#text#  (delimiters :+-*%!&|=~,^#_?@`/)\nraw strings     rq/anything until single slash/          rq#doubling ## escapes#\narrays          1 2 -3 4      1 \"ab\" -2 \"cd\"      (1 2;\"a\";3 \"b\";(4 2;\"c\");*)\nregexps         rx/[a-z]/                 (see FAQ for syntax and usage details)\ndyadic verbs    : + - * % ! & | < > = ~ , ^ # _ $ ? @ .     (right-associative)\nmonadic verbs   :: +: -: abs uc error ...\nadverbs         / \\ ' (alone or after expr. with no space)    (left-associative)\nexpressions     2*3+4 → 14       1+|2 3 4 → 5 4 3        +/'(1 2 3;4 5 6) → 6 15\nseparator       ; or newline except when ignored after {[( and before )]}\nvariables       x  y.z  f  data  t1  π               (. only allowed in globals)\nassign          x:2 (local within lambda, global otherwise)        x::2 (global)\nop assign       x+:1 (sugar for x:x+1 or x::x+1)          x-:2 (sugar for x:x-2)\nlist assign     (x;y;z):e (where 2<#e)         (x;y):1 2;y → 2\neval. order     apply:f[e1;e2]   apply:e1 op e2                      (e2 before)\n                list:(e1;e2)     seq: [e1;e2]     lambda:{e1;e2}     (e1 before)\nsequence        [x:2;y:x+3;x*y] → 10        (bracket not following noun tightly)\nindex/apply     x[y] or x y is sugar for x@y; x[] ~ x[*] ~ x[!#x] ~ x (arrays)\nindex deep      x[y;z;...] → x.(y;z;...)            (except for x in (?;and;or))\nindex assign    x[y]:z → x:@[x;y;:;z]                      (or . for x[y;...]:z)\nindex op assign x[y]op:z → x:@[x;y;op;z]                         (for symbol op)\nlambdas         {x+y-z}[3;5;7] → 1        {[a;b;c]a+b-c}[3;5;7] → 1\n                {?[x>1;x*o x-1;1]}5 → 120        (o is recursive self-reference)\nprojections     +[2;] 3 → 5               (2+) 3 → 5       (partial application)\ncompositions    ~0> → {~0>x}      -+ → {- x+y}      *|: → {*|x}\nindex at field  x..a → x[\"a\"]       (.. binds identifiers tightly, interpolable)\nfield expr.     ..a+b → {x[\"a\"]+x[\"b\"]} (field names without . and not in x,y,z)\n                ..p.a+b+q.c → {[p0;x]p0+x[\"b\"]+c}[a;]   (p. projects; q. quotes)\nfield expr. at  x.. a+b → {x[\"a\"]+x[\"b\"]}[x]                  (same as (..a+b)x)\ndict fields     ..[a:e1;b:e2;c] → \"a\"\"b\"\"c\"!(e1;e2;c)\namend fields    x..[a:e1;b:e2] → @[x;\"a\"\"b\";:;x..(e1;e2)]\ncond            ?[1;2;3] → 2      ?[0;2;3] → 3     ?[0;2;\"\";3;4] → 4\nand/or          and[1;2] → 2    and[1;0;3] → 0    or[0;2] → 2      or[0;0;0] → 0\nreturn          [1;:2;3] → 2                        (a : at start of expression)\ntry             'x is sugar for ?[\"e\"~@x;:x;x]         (return if it’s an error)\nlog             \\x logs a string representation of x        (debug/display tool)\ndiscard         `x discards well-formed x during parsing     (ignore expression)\ncomments        from line with a single / until line with a single \\\n                or from / (after space or start of line) to end of line\n"

const helpTypes = "TYPES HELP\natom    array   name            examples\ni       I       integer         0         -2        !5          4 3 -2 5 0i\nn       N       number          0.0       1.5       0.0+!5      1.2 3 0n 1e+10\ns       S       string          \"abc\"     \"d\"       \"a\" \"b\" \"c\"\nr               regexp          rx/[a-z]/           rx/\\s+/\nd               dictionary      \"a\"\"b\"!1 2          keys!values\nf               function        +         {x-1}     2*          %[;2]\nh               handle          open\"/path/to/file\"    \"w\"open\"/path/to/file\"\ne               error           error\"msg\"\n/               file system     dirfs\"/path/to/dir\"\n        A       generic array   (\"a\" 1;\"b\" 2;\"c\" 3)     (+;-;*;\"any\")\n"

const helpVerbs = "VERBS HELP\n:x  identity    :[42] → 42  (recall that : is also syntax for return and assign)\nx:y right       2:3 → 3                        \"a\":\"b\" → \"b\"\n+d  swap k/v    +\"a\"\"b\"!0 1 → 0 1!\"a\" \"b\"\n+x  flip        +(1 2;3 4) → (1 3;2 4)         +42 → ,,42\nn+n add         2+3 → 5                        2+3 4 → 5 6\ns+s concat      \"a\"+\"b\" → \"ab\"                 \"a\" \"b\"+\"c\" → \"ac\" \"bc\"\n-n  negate      - 2 3 → -2 -3                  -(1 2.5;3 4) → (-1.0 -2.5;-3 -4)\n-s  rtrim space -\"a\\tb \\r\\n\" \" c d \\n\" → \"a\\tb\" \" c d\"   (Unicode’s White Space)\nn-n subtract    5-3 → 2                        5 4-3 → 2 1\ns-s trim suffix \"file.txt\"-\".txt\" → \"file\"\n*x  first       *7 8 9 → 7                     *\"ab\" → \"ab\"           *(+;*) → +\nn*n multiply    2*3 → 6                        1 2 3*3 → 3 6 9\ns*i repeat      \"a\"*3 2 1 0 → \"aaa\" \"aa\" \"a\" \"\"\n%X  classify    %7 8 9 7 8 9 → 0 1 2 0 1 2     %\"a\" \"b\" \"a\" → 0 1 0\nn%n divide      3%2 → 1.5                      3 4%2 → 1.5 2.0\ns%s glob match  \"*.csv\"%\"data.csv\" \"code.goal\" \"dir/data.csv\" → 1 0 0\n!i  enum        !5 → 0 1 2 3 4                 !-5 → -5 -4 -3 -2 -1\n!s  fields      !\"a b\\tc\\nd \\u00a0e\" → \"a\"\"b\"\"c\"\"d\"\"e\"   (Unicode’s White Space)\n!I  odometer    !2 3 → (0 0 0 1 1 1;0 1 2 0 1 2)\n!d  keys        !\"a\"\"b\"!1 2 → \"a\" \"b\"\ni!n mod/div     3!9 8 7 → 0 2 1            -3!9 8 7 → 3 2 2\ni!s pad fields  3!\"a\" → \"a  \"              -3!\"1\" \"23\" \"456\" → \"  1\" \" 23\" \"456\"\ns!s fields      \",;\"!\"a,b;c\" → \"a\"\"b\"\"c\" (fields cut on any of \",;\"; \"\"!s is !s)\nX!Y dict        d:\"a\"\"b\"!1 2; d\"a\" → 1             (same as d:..[a:1;b:2]; d..a)\n&s  byte-count  &\"abc\" → 3                      &\"π\" → 2              &\"αβγ\" → 6\n&I  where       &0 0 1 0 0 0 1 → 2 6            &2 3 → 0 0 1 1 1\n&d  keys where  &\"a\"\"b\"\"c\"\"d\"!0 1 1 0 → \"b\" \"c\"\nx&y min/and     2&3 → 2          4&3 → 3         \"b\"&\"a\" → \"a\"           0&1 → 0\n|X  reverse     |!5 → 4 3 2 1 0\nx|y max/or      2|3 → 3          4|3 → 4         \"b\"|\"a\" → \"b\"           0|1 → 1\n<d  sort up     <\"a\"\"b\"\"c\"!2 3 1 → \"c\"\"a\"\"b\"!1 2 3\n<X  ascend      <3 5 4 → 0 2 1           (index permutation for ascending order)\nx<y less        2<3 → 1          \"c\"<\"a\" → 0                       7 8<6 9 → 0 1\n>d  sort down   >\"a\"\"b\"\"c\"!2 3 1 → \"b\"\"a\"\"c\"!3 2 1\n>X  descend     >3 5 4 → 1 2 0          (index permutation for descending order)\nx>y more        2>3 → 0          \"c\">\"a\" → 1                       7 8>6 9 → 1 0\n=s  lines       =\"ab\\ncd\\r\\nef gh\" → \"ab\" \"cd\" \"ef gh\"\n=I  index-count =1 0 0 2 2 3 -1 2 1 1 1 → 2 4 3 1\n=d  group keys  =\"a\"\"b\"\"c\"!0 1 0 → (\"a\" \"c\";,\"b\")           =\"a\"\"b\"!0 -1 → ,,\"a\"\nf=Y group by    (2!)=!10 → (0 2 4 6 8;1 3 5 7 9)\nx=y equal       2 3 4=3 → 0 1 0                          \"ab\" = \"ba\" → 0\n~x  not         ~0 1 2 → 1 0 0                           ~\"a\" \"\" \"0\" → 0 1 0\nx~y match       3~3 → 1            2 3~3 2 → 0           (\"a\";%)~'(\"b\";%) → 0 1\n,x  enlist      ,1 → ,1            #,2 3 → 1             (list with one element)\nd,d merge       (\"a\"\"b\"!1 2),\"b\"\"c\"!3 4 → \"a\"\"b\"\"c\"!1 3 4\nx,y join        1,2 → 1 2                       \"ab\" \"c\",\"d\" → \"ab\" \"c\" \"d\"\n^d  sort keys   ^\"c\"\"a\"\"b\"!1 2 3 → \"a\"\"b\"\"c\"!2 3 1\n^X  sort        ^3 5 0 → 0 3 5                  ^\"ca\" \"ab\" \"bc\" → \"ab\" \"bc\" \"ca\"\ni^s windows     2^\"abcde\" → \"abcd\" \"bcde\"\ni^Y windows     2^!4 → (0 1 2;1 2 3)                   -2^!4 → (0 1;1 2;2 3)\ns^s trim        \" #\"^\"  #a ## b#  \" → \"a ## b\"         \"\"^\" \\na\\t b\\t\" → \"a\\t b\"\nf^y weed out    {0 1 1 0}^4 1 5 3 → 4 3                (0<)^2 -3 1 → ,-3\nX^t w/o keys &  (,\"b\";1 0)^..[a:6 7;b:8 9] → (,\"a\")!,,7\n                (0;..a>1;..b<0)^..[a:1 2 3;b:4 -5 6] → \"a\"\"b\"!(,1;,4)\nX^d w/o keys    (,\"b\")^\"a\"\"b\"\"c\"!0 1 2 → \"a\"\"c\"!0 2\nX^Y w/o values  2 3^1 1 2 3 3 4 → 1 1 4                          (like in[;X]^Y)\n#x  length      #7 8 9 → 3        #\"ab\" \"cd\" → 2       #42 → 1      #\"ab\" → 1\ni#y take/repeat 2#6 7 8 → 6 7     5#6 7 8 → 6 7 8 6 7               3#1 → 1 1 1\ns#s count       \"ab\"#\"cabdab\" \"cd\" \"deab\" → 2 0 1                   \"\"#\"αβγ\" → 4\nf#y replicate   {0 1 2 0}#4 1 5 3 → 1 5 5              (0<)#2 -3 1 → 2 1\nX#t w/ keys &   (,\"a\";0 1)#..[a:6 7;b:8 9] → (,\"a\")!,,7\n                (1;..a>1;..b>0)#..[a:1 2 3;b:4 -5 6] → \"a\"\"b\"!(,3;,6)\nX#d with keys   \"a\"\"c\"\"e\"#\"a\"\"b\"\"c\"\"a\"!2 3 4 5 → \"a\"\"c\"\"a\"!2 4 5\nX#Y with values 2 3#1 1 2 3 3 4 → 2 3 3                          (like in[;X]#Y)\n_n  floor       _2.3 → 2.0                  _1.5 3.7 → 1.0 3.0\n_s  to lower    _\"ABC\" → \"abc\"              _\"AB\" \"CD\" \"Π\" → \"ab\" \"cd\" \"π\"\ni_s drop bytes  2_\"abcde\" → \"cde\"           -2_\"abcde\" → \"abc\"\ni_Y drop        2_3 4 5 6 → 5 6             -2_3 4 5 6 → 3 4\ns_s trim prefix \"pfx-\"_\"pfx-name\" \"pfx2-name\" → \"name\" \"pfx2-name\"\nf_Y cut where   {0=3!x}_!10 → (0 1 2;3 4 5;6 7 8;,9)          (same as (&f Y)_Y)\nI_s cut string  1 3_\"abcdef\" → \"bc\" \"def\"                          (I ascending)\nI_Y cut         2 5_!10 → (2 3 4;5 6 7 8 9)                        (I ascending)\n$x  string      $2 3 → \"2 3\"      $\"text\" → \"\\\"text\\\"\"\ni$s cut shape   3$\"abcdefghijk\" → \"abc\" \"defg\" \"hijk\"\ni$Y cut shape   3$!6 → (0 1;2 3;4 5)             -3$!6 → (0 1 2;3 4 5)\ns$y strings     \"s\"$(1;\"c\";+) → \"1\"\"c\"\"+\"\ns$s chars/bytes \"c\"$\"aπ\" → 97 960                \"b\"$\"aπ\" → 97 207 128\ns$i to string   \"c\"$97 960 → \"aπ\"                \"b\"$97 207 128 → \"aπ\"\ns$n cast        \"i\"$2.3 → 2                      @\"n\"$42 → \"n\"\ns$s parse i/n   \"i\"$\"42\" \"0b100\" → 42 4          \"n\"$\"2.5\" \"1e+20\" → 2.5 1e+20\ns$s parse value \"v\"$qq/(2 3;\"a\")/ → (2 3;\"a\")   ($x inverse for types in \"inrs\")\ns$y format      \"%.2g\"$1 4%3 → \"0.33\" \"1.3\"      \"%s=%03d\"$\"a\" 42 → \"a=042\"\nX$y binsearch   2 3 5 7$8 2 7 5 5.5 3 0 → 4 1 4 3 3 2 0            (X ascending)\n?i  uniform     ?2 → 0.6046602879796196 0.9405090880450124     (between 0 and 1)\n?i  normal      ?-2 → -1.233758177597947 -0.12634751070237293    (mean 0, dev 1)\n?X  distinct    ?2 2 4 3 2 3 → 2 4 3                   (keeps first occurrences)\ni?i roll        5?100 → 10 51 21 51 37\ni?Y roll array  5?\"a\" \"b\" \"c\" → \"c\" \"a\" \"c\" \"c\" \"b\"\ni?i deal        -5?100 → 19 26 0 73 94                         (always distinct)\ni?Y deal array  -3?\"a\"\"b\"\"c\" → \"a\"\"c\"\"b\"     (0i?Y is (-#Y)?Y) (always distinct)\ns?r rindex      \"abcde\"?rx/b../ → 1 3                            (offset;length)\ns?s index       \"a = a + 1\"?\"=\" \"+\" → 2 6\nd?y find key    (\"a\"\"b\"!3 4)?4 → \"b\"                    (\"a\" \"b\"!3 4)?5 → \"\"\nX?y find        9 8 7?8 → 1                              9 8 7?6 → 3\n@x  type        @2 → \"i\"    @1.5 → \"n\"    @\"ab\" → \"s\"    @2 3 → \"I\"   @(+) → \"f\"\ni@y take/pad    2@6 7 8 → 6 7     4@6 7 8 → 6 7 8 0      -4@6 7 8 → 0 6 7 8\ns@i substr      \"abcdef\"@2  → \"cdef\"                                 (s[offset])\nr@s match       rx/^[a-z]+$/\"abc\" → 1                    rx/\\s/\"abc\" → 0\nr@s find group  rx/([a-z])(.)/\"&a+c\" → \"a+\" \"a\" \"+\"      (whole match, group(s))\nf@y apply at    (|)@1 2 → 2 1            (like |[1 2] → 2 1 or |1 2 or (|).,1 2)\nd@y at key      ..[a:6 7;b:8 9]\"a\" → 6 7                 (1 2!\"a\"\"b\")2 → \"b\"\nt@i at row      ..[a:6 7;b:8 9]0 → \"a\"\"b\"!6 8\nX@i at          7 8 9@2 → 9             7 8 9[2 0] → 9 7            7 8 9@-2 → 8\n.s  get global  a:3;.\"a\" → 3\n.e  get error   .error\"msg\" → \"msg\"\n.d  values      .\"a\"\"b\"!1 2 → 1 2\n.X  self-dict   .\"a\"\"b\" → \"a\"\"b\"!\"a\"\"b\"          .!3 → 0 1 2!0 1 2\ns.I substr      \"abcdef\"[2;3] → \"cde\"                         (s[offset;length])\nr.y findN       rx/[a-z]/[\"abc\";2] → \"a\"\"b\"      rx/[a-z]/[\"abc\";-1] → \"a\"\"b\"\"c\"\nr.y findN group rx/[a-z](.)/[\"abcdef\";2] → (\"ab\" \"b\";\"cd\" \"d\")\nf.y apply       (+).2 3 → 5                      +[2;3] → 5\nd.y deep at     ..[a:6 7;b:8 9][\"a\";1] → 7\nt.y at row;key  ..[a:6 7;b:8 9][1;\"a\"] → (,\"a\")!,7\nX.y deep at     (6 7;8 9)[0;1] → 7               (6 7;8 9)[;1] → 7 9\n«X  shift       «8 9 → 9 0     «\"a\" \"b\" → \"b\" \"\"        (ASCII keyword: shift x)\nx«Y shift       \"a\" \"b\"«1 2 3 → 3 \"a\" \"b\"\n»X  rshift      »8 9 → 0 8     »\"a\" \"b\" → \"\" \"a\"       (ASCII keyword: rshift x)\nx»Y rshift      \"a\" \"b\"»1 2 3 → \"a\" \"b\" 1\n\n::[s;y]     set global  ::[\"a\";3];a → 3  (brackets needed because :: is monadic)\n@[d;y;f]    amend       @[\"a\"\"b\"\"c\"!7 8 9;\"a\"\"b\"\"b\";10+] → \"a\"\"b\"\"c\"!17 28 9\n@[X;i;f]    amend       @[7 8 9;0 1 1;10+] → 17 28 9\n@[d;y;F;z]  amend       @[\"a\"\"b\"\"c\"!7 8 9;\"a\";:;42] → \"a\"\"b\"\"c\"!42 8 9\n@[X;i;F;z]  amend       @[7 8 9;1 2 0;+;10 20 -10] → -3 18 29\n@[f;y;f]    try at      @[2+;\"a\";:] → \"i+y : bad type \\\"s\\\" in y\"   (panic case)\n                        @[2+;3;:] → 5                            (no-panic case)\n.[X;y;f]    deep amend  .[(6 7;8 9);0 1;-] → (6 -7;8 9)\n.[X;y;F;z]  deep amend  .[(6 7;8 9);(0 1 0;1);+;10] → (6 27;8 19)\n                        .[(6 7;8 9);(*;1);:;42] → (6 42;8 42)\n.[f;y;f]    try         .[+;2 \"a\";:] → \"i+y : bad type \\\"s\\\" in y\"  (panic case)\n                        .[+;2 3;:] → 5                           (no-panic case)\n"

const helpNamedVerbs = "NAMED VERBS HELP\nabs n      abs value    abs -3.0 -1.5 2.0 → 3.0 1.5 2.0\ncsv s      csv read     csv\"a,b\\n1,2\" → (\"a\" \"1\";\"b\" \"2\")\ncsv A      csv write    csv(\"a\" \"b\";1 2) → \"a,1\\nb,2\\n\"\nerror x    error        r:error\"msg\"; (@r;.r) → \"e\" \"msg\"\neval s     comp/run     a:5;eval\"a+2\" → 7           (unrestricted variant of .s)\nfirsts X   mark firsts  firsts 0 0 2 3 0 2 3 4 → 1 0 1 1 0 0 0 1    (same as ¿X)\njson s     parse json   json rq`{\"a\":true,\"b\":\"text\"}` → \"a\" \"b\"!0w \"text\"\nnan n      isNaN        nan(0n;2;sqrt -1) → 1 0 1             nan 2 0i 3 → 0 1 0\nocount X   occur-count  ocount 3 4 5 3 4 4 7 → 0 0 0 1 1 2 0\npanic s    panic        panic\"msg\"                (for fatal programming-errors)\nrx s       comp. regex  rx\"[a-z]\"       (like rx/[a-z]/ but compiled at runtime)\nsign n     sign         sign -3 -1 0 1.5 5 → -1 -1 0 1 1\nuc x       upper/ceil   uc 1.5 → 2.0                             uc\"abπ\" → \"ABΠ\"\n\ns csv s    csv read     \" \"csv\"a b\\n1 2\" → (\"a\" \"1\";\"b\" \"2\")  (\" \" as separator)\ns csv A    csv write    \" \"csv(\"a\" \"b\";1 2) → \"a 1\\nb 2\\n\"    (\" \" as separator)\nx in s     contained    \"bc\" \"ac\"in\"abcd\" → 1 0                    (same as x¿s)\nx in Y     member of    2 3 in 8 2 4 → 1 0                         (same as x¿Y)\ns json y   write json   \"\"json 1.5 2 → \"[1.5,2]\" (indent with s;disable with \"\")\nS json y   write json   like s json y, but with (pfx;indent) for pretty-printing\nn nan n    fill NaNs    42.0 nan(1.5;sqrt -1) → 1.5 42.0      42 nan 2 0i → 2 42\ni rotate Y rotate       2 rotate 7 8 9 → 9 7 8           -2 rotate 7 8 9 → 8 9 7\n\nsub[r;s]   regsub       sub[rx/[a-z]/;\"Z\"] \"aBc\" → \"ZBZ\"\nsub[r;f]   regsub       sub[rx/[A-Z]/;_] \"aBc\" → \"abc\"\nsub[s;s]   replace      sub[\"b\";\"B\"] \"abc\" → \"aBc\"\nsub[s;s;i] replaceN     sub[\"a\";\"b\";2] \"aaa\" → \"bba\"        (stop after 2 times)\nsub[S]     replaceS     sub[\"b\" \"d\" \"c\" \"e\"] \"abc\" → \"ade\"\nsub[S;S]   replaceS     sub[\"b\" \"c\";\"d\" \"e\"] \"abc\" → \"ade\"\n\neval[s;loc;pfx]         like eval s, but provide loc as location (usually a\n                        path), and prefix pfx+\".\" for globals; does not eval\n                        same location more than once\n\nutf8 s     is UTF-8     utf8 \"aπc\" → 1                          utf8 \"a\\xff\" → 0\ns utf8 s   to UTF-8     \"b\" utf8 \"a\\xff\" → \"ab\"       (replace invalid with \"b\")\n\nMATH: atan[n;n]; cos n; exp n; log n; round n; sin n; sqrt n\n"

const helpAdverbs = "ADVERBS HELP\nf'x    each      #'(4 5;6 7 8) → 2 3\nx F'y  each      2 3#'4 5 → (4 4;5 5 5)      {(x;y;z)}'[1;2 3;4] → (1 2 4;1 3 4)\nx I'y  case      (6 7 8 9)0 1 0 1'\"a\"\"b\"\"c\"\"d\" → 6 \"b\" 8 \"d\"\nI A'I  at each   m:3$!9;p:!2 2;(m').p → 0 1 3 4\nx F`y  eachleft  1 2,`\"a\"\"b\" → (1 \"a\" \"b\";2 \"a\" \"b\")           (same as F[;y]'x)\nx F´y  eachright 1 2,´\"a\"\"b\" → (1 2 \"a\";1 2 \"b\")               (same as F[x;]'y)\nF/x    fold      +/!10 → 45\nF\\x    scan      +\\!10 → 0 1 3 6 10 15 21 28 36 45\nx F/y  fold      5 6+/!4 → 11 12                     {x+y-z}/[5;4 3;2 1] → 9\nx F\\y  scan      5 6+\\!4 → (5 6;6 7;8 9;11 12)       {x+y-z}\\[5;4 3;2 1] → 7 9\ni f/y  do        3(2*)/4 → 32\ni f\\y  dos       3(2*)\\4 → 4 8 16 32\nf f/y  while     (100>)(2*)/4 → 128\nf f\\y  whiles    (100>)(2*)\\4 → 4 8 16 32 64 128\nf/x    converge  {1+1.0%x}/1 → 1.618033988749895     {-x}/1 → -1\nf\\x    converges (-2!)\\10 → 10 5 2 1 0               {-x}\\1 → 1 -1\ns/S    join      \",\"/\"a\" \"b\" \"c\" → \"a,b,c\"\ns\\s    split     \",\"\\\"a,b,c\" → \"a\" \"b\" \"c\"           \"\"\\\"aπc\" → \"a\" \"π\" \"c\"\nr\\s    split     rx/[,;]/\\\"a,b;c,d\" → \"a\" \"b\" \"c\" \"d\"\ni s\\s  splitN    (2)\",\"\\\"a,b,c\" → \"a\" \"b,c\"\ni r\\s  splitN    (3)rx/[,;]/\\\"a,b;c,d\" → \"a\" \"b\" \"c,d\"\nI/x    decode    24 60 60/1 2 3 → 3723               2/1 1 0 → 6\nI\\x    encode    24 60 60\\3723 → 1 2 3               2\\6 → 1 1 0\n"

const helpTime = "TIME HELP\ntime cmd              time command with current time\ncmd time t            time command with time t\ntime[cmd;t;fmt]       time command with time t in given format\ntime[cmd;t;fmt;loc]   time command with time t in given format and location\n\nTime t should consist either of integers or strings in the given format (\"unix\"\nis the default for integers and RFC3339 layout \"2006-01-02T15:04:05Z07:00\" is\nthe default for strings), with optional location (default is \"UTC\"). See FAQ\nfor information on layouts, locations, and date calculations. Supported values\nfor cmd are as follows:\n\n    cmd (s)       result (type)                                 fmt\n    -------       -------------                                 ---\n    \"clock\"       hour, minute, second (I)\n    \"date\"        year, month, day (I)                          yes\n    \"day\"         day number (i)\n    \"hour\"        0-23 hour (i)\n    \"minute\"      0-59 minute (i)\n    \"month\"       0-12 month (i)\n    \"second\"      0-59 second (i)\n    \"unix\"        unix epoch time (i)                           yes\n    \"unixmicro\"   unix (microsecond version) (i)                yes\n    \"unixmilli\"   unix (millisecond version) (i)                yes\n    \"unixnano\"    unix (nanosecond version) (i)                 yes\n    \"week\"        year, week (I)\n    \"weekday\"     0-7 weekday starting from Sunday (i)\n    \"year\"        year (i)\n    \"yearday\"     1-365/6 year day (i)\n    \"zone\"        name, offset in seconds east of UTC (s;i)\n    format (s)    format time using given layout (s)            yes\n"

const helpRuntime = "RUNTIME HELP\nrt.get s        returns runtime information named by s:\n                  \"!\"     names of globals (S)\n                  \"kw\"    names of verbal keywords (S)\n                  \"loc\"   current eval location (s)\n                  \"os\"    operating system (s) (Go’s runtime.GOOS)\n                  \"pfx\"   current eval prefix for globals (s)\n                  \"v\"     interpreter’s version (s)\nrt.get[\"@\";x]   dict with internal info about x        (for debug purposes only)\nrt.log x        like :[x] but logs string representation of x       (same as \\x)\nrt.seed i       set non-secure pseudo-rand seed to i        (used by the ? verb)\nrt.time[s;i]    eval s for i times (default 1), return average time (ns)\nrt.time[f;y;i]  call f. y for i times (default 1), return average time (ns)\nrt.try[f;y;f]   same as .[f;y;f] but the handler receives an error dict\n                ..[msg:s;err:s] where msg contains the whole error stack trace\n"

const helpIO = "IO/OS HELP\nabspath s   return an absolute representation of path or joined path elements\nchdir s     change current working directory to s\nclose h     flush any buffered data, then close handle h\ndirfs s     return a read-only file system value fs, rooted at directory s,\n            usable as left argument in glob, import, open, read, stat\n            fs subfs dir returns the subtree rooted at fs’ dir\nenv s       return value of environment variable s, or an error if unset\n            return a dict representing the whole environment if s~\"\"\nflush h     flush any buffered data for handle h\nglob s      return file names matching glob pattern(s)     (ignores stat errors)\nimport s    read/eval wrapper roughly equivalent to eval[src;path;pfx] where:\n              s with extension         s without extension\n              ----------------         -----------------------------------\n              pfx~basename s           pfx~sub[\"/\";\".\"]s\n              path~s                   path~s+\".goal\"\n              src~read s               src~(dirfs env[\"GOALLIB\"])read path\nmkdir s     create new directory named s             (parent must already exist)\nopen s      open path s for reading, returning a handle h\nprint x     print\"Hello, world!\\n\"           (uses implicit $x for ~(@x)in\"sSe\")\nread h      read from handle h until EOF or an error occurs\nread s      read file named s into string         lines:=-read\"/path/to/file\"\n            or dict ..[dir:I;name:S] if s is a directory\nremove s    remove the named file or empty directory\nrun s       run command s or S (with arguments)   run\"pwd\"          run\"ls\" \"-l\"\n            inherits stdin and stderr, returns its standard output or an error\n            dict ..[msg:s;code:i;out:s]\nsay x       same as print, but appends a newline                    say!5\nshell s     same as run, but through \"/bin/sh\" (unix systems only)  shell\"ls -l\"\nstat x      returns dict ..[dir:i;mtime:i;size:i]   (for filehandle h or path s)\n\nx env s     set environment variable x to s\nx env 0     unset environment variable x, or clear environment if x~\"\"\nx import s  same as import s, but using prefix x for globals\nx open s    open path s with mode x in \"r\" \"w\" \"a\"      (file read/write/append)\n            or pipe from or to command s or S with \"pr\" \"pw\"   (pipe read/write)\n            returns handle h that may be called on:\n              \"dir\"     whether file is dir (i)   (mode \"r\")\n              \"mode\"    handle’s mode (s)         (available for all modes)\n              \"name\"    file’s name (s)           (modes \"r\" \"w\" \"a\")\n              \"sync\"    file sync (i or e)        (modes \"w\" \"a\")\nx print y   print y to handle/path x               \"/path/to/file\"print\"content\"\ni read h    read i bytes from reader h or until EOF, or an error occurs\ns read h    read from reader h until 1-byte s, EOF, or an error occurs\nx rename y  renames (moves) old path x (s) to new path y (s)\nx run s     same as run s but with input string x as stdin\nx say y     same as print, but appends a newline\n\nARGS        command-line arguments, starting with script name\nSTDIN       standard input filehandle (buffered)\nSTDOUT      standard output filehandle (buffered)\nSTDERR      standard error filehandle (buffered)\n"
