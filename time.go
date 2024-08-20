package ari

import (
	"fmt"
	"time"

	"codeberg.org/anaseto/goal"
)

// TODO All functions under Time type https://pkg.go.dev/time@go1.22.5#Time

type Time struct {
	Time *time.Time
}

// Append implements goal.BV.
func (time *Time) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	return append(dst, fmt.Sprintf("<%v %#v>", time.Type(), time.Time)...)
}

// LessT implements goal.BV.
func (time *Time) LessT(y goal.BV) bool {
	// Goal falls back to ordering by type name,
	// and there is no other reasonable way to order
	// these structs.
	return time.Type() < y.Type()
}

// Matches implements goal.BV.
func (time *Time) Matches(y goal.BV) bool {
	switch yv := y.(type) {
	case *Time:
		return time == yv
	default:
		return false
	}
}

// Type implements goal.BV.
func (time *Time) Type() string {
	return "ari.Time"
}

// Implement time.now function.
func VFTimeNow(_ *goal.Context, args []goal.V) goal.V {
	now := time.Now()
	tt := Time{&now}
	return goal.NewV(&tt)
}

// Implements time.parse function.
func VFTimeParse(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	layoutS, ok := x.BV().(goal.S)
	if !ok {
		return panicType("slayout time.parse svalue", "sformat", x)
	}
	switch len(args) {
	case dyadic:
		y := args[0]
		valueS, ok := y.BV().(goal.S)
		if !ok {
			return panicType("slayout time.parse svalue", "svalue", y)
		}
		t, err := time.Parse(string(layoutS), string(valueS))
		if err != nil {
			return goal.NewPanicError(err)
		}
		tt := Time{&t}
		return goal.NewV(&tt)
	default:
		return goal.Panicf("time.parse : too many arguments (%d), expects 2 arguments", len(args))
	}
}

// Implements time.add function.
func VFTimeAdd(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	t1, ok := x.BV().(*Time)
	if !ok {
		return panicType("time time.add i", "time", x)
	}
	switch len(args) {
	case dyadic:
		y := args[0]
		if !y.IsI() {
			if y.IsBV() {
				switch ai := y.BV().(type) {
				case *goal.AB:
					al := ai.Len()
					if al != 3 {
						return goal.Panicf("time.add : I arg must have 3 items, had %d", al)
					}
					year := ai.At(0).I()
					month := ai.At(1).I()
					day := ai.At(2).I()
					t := t1.Time.AddDate(int(year), int(month), int(day))
					tt := Time{&t}
					return goal.NewV(&tt)
				case *goal.AI:
					al := ai.Len()
					if al != 3 {
						return goal.Panicf("time.add : I arg must have 3 items, had %d", al)
					}
					year := ai.At(0).I()
					month := ai.At(1).I()
					day := ai.At(2).I()
					t := t1.Time.AddDate(int(year), int(month), int(day))
					tt := Time{&t}
					return goal.NewV(&tt)
				default:
					return panicType("time time.add I", "I", y)
				}
			}
			return panicType("time time.add i", "i", y)
		}
		t := t1.Time.Add(time.Duration(y.I()))
		tt := Time{&t}
		return goal.NewV(&tt)
	default:
		return goal.Panicf("time.add : too many arguments (%d), expects 2 arguments", len(args))
	}
}

// Implements time.sub function.
func VFTimeSub(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	t1, ok := x.BV().(*Time)
	if !ok {
		return panicType("ari.Time1 time.sub ari.Time2", "time.Time1", x)
	}
	switch len(args) {
	case dyadic:
		y := args[0]
		t2, ok := y.BV().(*Time)
		if !ok {
			return panicType("ari.Time1 time.sub ari.Time2", "time.Time1", y)
		}
		durNanos := t1.Time.Sub(*t2.Time)
		return goal.NewI(durNanos.Nanoseconds())
	default:
		return goal.Panicf("time.sub : too many arguments (%d), expects 2 arguments", len(args))
	}
}

func registerTimeGlobals(goalContext *goal.Context) {
	goalContext.AssignGlobal("time.Layout", goal.NewS(time.Layout))
	goalContext.AssignGlobal("time.ANSIC", goal.NewS(time.ANSIC))
	goalContext.AssignGlobal("time.UnixDate", goal.NewS(time.UnixDate))
	goalContext.AssignGlobal("time.RubyDate", goal.NewS(time.RubyDate))
	goalContext.AssignGlobal("time.RFC822", goal.NewS(time.RFC822))
	goalContext.AssignGlobal("time.RFC822Z", goal.NewS(time.RFC822Z))
	goalContext.AssignGlobal("time.RFC850", goal.NewS(time.RFC850))
	goalContext.AssignGlobal("time.RFC1123", goal.NewS(time.RFC1123))
	goalContext.AssignGlobal("time.RFC1123Z", goal.NewS(time.RFC1123Z))
	goalContext.AssignGlobal("time.RFC3339", goal.NewS(time.RFC3339))
	goalContext.AssignGlobal("time.RFC3339Nano", goal.NewS(time.RFC3339Nano))
	goalContext.AssignGlobal("time.Kitchen", goal.NewS(time.Kitchen))
	goalContext.AssignGlobal("time.Stamp", goal.NewS(time.Stamp))
	goalContext.AssignGlobal("time.StampMilli", goal.NewS(time.StampMilli))
	goalContext.AssignGlobal("time.StampMicro", goal.NewS(time.StampMicro))
	goalContext.AssignGlobal("time.StampNano", goal.NewS(time.StampNano))
	goalContext.AssignGlobal("time.DateTime", goal.NewS(time.DateTime))
	goalContext.AssignGlobal("time.DateOnly", goal.NewS(time.DateOnly))
	goalContext.AssignGlobal("time.TimeOnly", goal.NewS(time.TimeOnly))

	goalContext.AssignGlobal("time.Nanosecond", goal.NewI(int64(time.Nanosecond)))
	goalContext.AssignGlobal("time.Microsecond", goal.NewI(int64(time.Microsecond)))
	goalContext.AssignGlobal("time.Millisecond", goal.NewI(int64(time.Millisecond)))
	goalContext.AssignGlobal("time.Second", goal.NewI(int64(time.Second)))
	goalContext.AssignGlobal("time.Minute", goal.NewI(int64(time.Minute)))
	goalContext.AssignGlobal("time.Hour", goal.NewI(int64(time.Hour)))

	goalContext.AssignGlobal("time.Sunday", goal.NewI(int64(time.Sunday)))
	goalContext.AssignGlobal("time.Monday", goal.NewI(int64(time.Monday)))
	goalContext.AssignGlobal("time.Tuesday", goal.NewI(int64(time.Tuesday)))
	goalContext.AssignGlobal("time.Wednesday", goal.NewI(int64(time.Wednesday)))
	goalContext.AssignGlobal("time.Thursday", goal.NewI(int64(time.Thursday)))
	goalContext.AssignGlobal("time.Friday", goal.NewI(int64(time.Friday)))
	goalContext.AssignGlobal("time.Saturday", goal.NewI(int64(time.Saturday)))
}