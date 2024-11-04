package ari

import (
	"fmt"

	"codeberg.org/anaseto/goal"
	"go.uber.org/ratelimit"
)

type RateLimiter struct {
	Limiter *ratelimit.Limiter
}

// Append implements goal.BV.
func (rateLimiter *RateLimiter) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	return append(dst, fmt.Sprintf("<%v %#v>", rateLimiter.Type(), rateLimiter.Limiter)...)
}

// LessT implements goal.BV.
func (rateLimiter *RateLimiter) LessT(y goal.BV) bool {
	return rateLimiter.Type() < y.Type()
}

// Matches implements goal.BV.
func (rateLimiter *RateLimiter) Matches(_ goal.BV) bool {
	return false
}

// Type implements goal.BV.
func (rateLimiter *RateLimiter) Type() string {
	return "ari.RateLimiter"
}

func VFRateLimitNew(_ *goal.Context, args []goal.V) goal.V {
	if len(args) > 1 {
		return goal.Panicf("ratelimit.new : too many arguments (%d), expects 1 argument", len(args))
	}
	x := args[0]
	if x.IsI() {
		limiter := ratelimit.New(int(x.I()), ratelimit.WithoutSlack)
		return goal.NewV(&RateLimiter{Limiter: &limiter})
	}
	return panicType("ratelimit.new i", "i", x)
}

func VFRateLimitTake(_ *goal.Context, args []goal.V) goal.V {
	if len(args) > 1 {
		return goal.Panicf("ratelimit.take : too many arguments (%d), expects 1 argument", len(args))
	}
	x := args[0]
	switch xv := x.BV().(type) {
	case *RateLimiter:
		limiter := xv.Limiter
		now := (*limiter).Take()
		return goal.NewV(&Time{Time: &now})
	default:
		return panicType("ratelimit.take ari.RateLimiter", "ari.RateLimiter", x)
	}
}
