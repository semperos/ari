package cmd

import (
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
