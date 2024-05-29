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

var goalNonAsciis = map[string]string{
	"eachleft":  "`", // this is ASCII, but for completeness and less surprise
	"eachright": "´",
	"rshift":    "»",
	"shift":     "«",
	"firsts":    "¿",
}

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
  k:(|/-1+""#k)!k:"s"$!d; v:" "/'(..?[(@x)¿"nN";p.f$x;$'x])'.d
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
  k:"s"$!t; v:(..?[(@x)¿"nN";p.f$x;$'x])'.t; w:(-1+""#k)|(|/-1+""#)'v
  (k;v):(-w)!'´(k;v); say"\n"/,/(" "/k;" "/"-"*w;" "/'+v)
}
`
