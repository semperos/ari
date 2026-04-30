// Package ratelimit provides Goal bindings for leaky-bucket rate limiting.
//
// Import registers ratelimit.* verbs into a Goal context. Call it as:
//
//	ratelimit.Import(ctx, "")
//
// which registers globals prefixed with "ratelimit." (e.g. ratelimit.new).
//
// # Verb summary
//
//	ratelimit.new  i   – create a RateLimiter allowing i requests per second
//	ratelimit.take rl  – block until the limiter allows the next request;
//	                     returns 1i when unblocked
//
// # Usage
//
//	rl: ratelimit.new 10       / allow 10 requests per second
//	ratelimit.take rl          / blocks if needed, then returns 1i
//	http.get "https://..."
//
// The leaky-bucket algorithm means requests are evenly spaced; there is no
// burst capacity. When http.client is configured with RateLimitPerSecond the
// rate limiter is called automatically before each request, so manual use of
// ratelimit.take is only needed when coordinating rate limiting across code
// that does not go through an http.client.
package ratelimit

import (
	goal "codeberg.org/anaseto/goal"
	uber "go.uber.org/ratelimit"
)

// ---------------------------------------------------------------------------
// BV wrapper: ratelimit.limiter
// ---------------------------------------------------------------------------

// Limiter wraps a go.uber.org/ratelimit Limiter as a Goal boxed value.
type Limiter struct{ l uber.Limiter }

func (rl *Limiter) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	return append(dst, "ratelimit.limiter"...)
}
func (rl *Limiter) Matches(y goal.BV) bool { yv, ok := y.(*Limiter); return ok && rl == yv }
func (rl *Limiter) LessT(y goal.BV) bool   { return rl.Type() < y.Type() }
func (rl *Limiter) Type() string           { return "ratelimit.limiter" }

// ---------------------------------------------------------------------------
// Import
// ---------------------------------------------------------------------------

// Import registers all ratelimit.* verbs into ctx.
func Import(ctx *goal.Context, pfx string) {
	ctx.RegisterExtension("ratelimit", "")
	if pfx != "" {
		pfx += "."
	}

	reg := func(name string, f goal.VariadicFunc) {
		fullname := pfx + name
		v := ctx.RegisterMonad("."+fullname, f)
		ctx.AssignGlobal(fullname, v)
	}

	reg("ratelimit.new", vfNew)
	reg("ratelimit.take", vfTake)
}

// ---------------------------------------------------------------------------
// ratelimit.new  (monad: ratelimit.new i)
// ---------------------------------------------------------------------------

func vfNew(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("ratelimit.new i : expected 1 argument, got %d", len(args))
	}
	x := args[0]
	if !x.IsI() {
		return goal.Panicf("ratelimit.new i : expected integer, got %q", x.Type())
	}
	n := int(x.I())
	if n <= 0 {
		return goal.Panicf("ratelimit.new i : rate must be a positive integer, got %d", n)
	}
	return goal.NewV(&Limiter{uber.New(n, uber.WithoutSlack)})
}

// ---------------------------------------------------------------------------
// ratelimit.take  (monad: ratelimit.take rl)
// ---------------------------------------------------------------------------

func vfTake(_ *goal.Context, args []goal.V) goal.V {
	if len(args) != 1 {
		return goal.Panicf("ratelimit.take rl : expected 1 argument, got %d", len(args))
	}
	rl, ok := args[0].BV().(*Limiter)
	if !ok {
		return goal.Panicf("ratelimit.take rl : expected ratelimit.limiter, got %q", args[0].Type())
	}
	rl.l.Take()
	return goal.NewI(1)
}
