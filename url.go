package ari

import (
	"net/url"

	"codeberg.org/anaseto/goal"
)

// Implements url.encode.
func vfURLEncode(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch arg := x.BV().(type) {
	case goal.S:
		return goal.NewS(url.PathEscape(string(arg)))
	case *goal.D:
		ks := arg.KeyArray()
		vs := arg.ValueArray()
		urlValues := url.Values{}
		for i := 0; i < ks.Len(); i++ {
			k := ks.At(i)
			kS, ok := k.BV().(goal.S)
			if !ok {
				//nolint:lll // Error message is descriptive.
				return goal.Panicf("url.encode expects a dictionary with string keys, but received a dict with a %q key: %v", k.Type(), k)
			}
			v := vs.At(i)
			vS, ok := v.BV().(goal.S)
			if !ok {
				//nolint:lll // Error message is descriptive.
				return goal.Panicf("url.encode expects a dictionary with string values, but received a dict with a %q value: %v", v.Type(), v)
			}
			urlValues.Add(string(kS), string(vS))
		}
		return goal.NewS(urlValues.Encode())
	default:
		return panicType("url.encode s-or-d", "s-or-d", x)
	}
}
