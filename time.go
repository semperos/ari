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
	switch yv := y.(type) {
	case *Time:
		return time.Time.Before(*yv.Time)
	default:
		return time.Type() < y.Type()
	}
}

// Matches implements goal.BV.
func (time *Time) Matches(y goal.BV) bool {
	switch yv := y.(type) {
	case *Time:
		return time.Time.Equal(*yv.Time)
	default:
		return false
	}
}

// Type implements goal.BV.
func (time *Time) Type() string {
	return "ari.Time"
}

type Location struct {
	Location *time.Location
}

// Append implements goal.BV.
func (location *Location) Append(_ *goal.Context, dst []byte, _ bool) []byte {
	return append(dst, fmt.Sprintf("<%v %#v>", location.Type(), location.Location)...)
}

// LessT implements goal.BV.
func (location *Location) LessT(y goal.BV) bool {
	switch yv := y.(type) {
	case *Location:
		return location.Location.String() < yv.Location.String()
	default:
		return location.Type() < y.Type()
	}
}

// Matches implements goal.BV.
func (location *Location) Matches(_ goal.BV) bool {
	// You can have two location objects with the same name but different offsets and the offset is not exposed publicly,
	// so we don't support equality directly.
	return false
}

// Type implements goal.BV.
func (location *Location) Type() string {
	return "ari.Location"
}

// Implements time.date function.
//
//nolint:cyclop,funlen,gocognit,gocyclo // the different arities are flat and similar, keeping them close is better
func vfTimeDate(_ *goal.Context, args []goal.V) goal.V {
	// year int, month Month, day, hour, min, sec, nsec int, loc *Location
	switch len(args) {
	case monadic:
		yearV := args[0]
		if yearV.IsI() {
			year := int(yearV.I())
			month := time.January
			day := 1
			hour := 0
			minute := 0
			sec := 0
			nsec := 0
			loc := time.UTC
			t := time.Date(year, month, day, hour, minute, sec, nsec, loc)
			return goal.NewV(&Time{Time: &t})
		}
		return panicType("time.date year", "year", yearV)
	case dyadic:
		yearV := args[1]
		monthV := args[0]
		if !yearV.IsI() {
			return panicType("time.date year month", "year", yearV)
		}
		if !monthV.IsI() {
			return panicType("time.date year month", "month", monthV)
		}
		year := int(yearV.I())
		month := time.Month(monthV.I())
		day := 1
		hour := 0
		minute := 0
		sec := 0
		nsec := 0
		loc := time.UTC
		t := time.Date(year, month, day, hour, minute, sec, nsec, loc)
		return goal.NewV(&Time{Time: &t})
	case triadic:
		yearV := args[2]
		monthV := args[1]
		dayV := args[0]
		if !yearV.IsI() {
			return panicType("time.date year month day", "year", yearV)
		}
		if !monthV.IsI() {
			return panicType("time.date year month day", "month", monthV)
		}
		if !dayV.IsI() {
			return panicType("time.date year month day", "day", dayV)
		}
		year := int(yearV.I())
		month := time.Month(monthV.I())
		day := int(dayV.I())
		hour := 0
		minute := 0
		sec := 0
		nsec := 0
		loc := time.UTC
		t := time.Date(year, month, day, hour, minute, sec, nsec, loc)
		return goal.NewV(&Time{Time: &t})
	case tesseradic:
		yearV := args[3]
		monthV := args[2]
		dayV := args[1]
		hourV := args[0]
		if !yearV.IsI() {
			return panicType("time.date year month day hour", "year", yearV)
		}
		if !monthV.IsI() {
			return panicType("time.date year month day hour", "month", monthV)
		}
		if !dayV.IsI() {
			return panicType("time.date year month day hour", "day", dayV)
		}
		if !hourV.IsI() {
			return panicType("time.date year month day hour", "hour", hourV)
		}
		year := int(yearV.I())
		month := time.Month(monthV.I())
		day := int(dayV.I())
		hour := int(hourV.I())
		minute := 0
		sec := 0
		nsec := 0
		loc := time.UTC
		t := time.Date(year, month, day, hour, minute, sec, nsec, loc)
		return goal.NewV(&Time{Time: &t})
	case pentadic:
		yearV := args[4]
		monthV := args[3]
		dayV := args[2]
		hourV := args[1]
		minV := args[0]
		if !yearV.IsI() {
			return panicType("time.date year month day hour minute", "year", yearV)
		}
		if !monthV.IsI() {
			return panicType("time.date year month day hour minute", "month", monthV)
		}
		if !dayV.IsI() {
			return panicType("time.date year month day hour minute", "day", dayV)
		}
		if !hourV.IsI() {
			return panicType("time.date year month day hour minute", "hour", hourV)
		}
		if !minV.IsI() {
			return panicType("time.date year month day hour minute", "minute", minV)
		}
		year := int(yearV.I())
		month := time.Month(monthV.I())
		day := int(dayV.I())
		hour := int(hourV.I())
		minute := int(minV.I())
		sec := 0
		nsec := 0
		loc := time.UTC
		t := time.Date(year, month, day, hour, minute, sec, nsec, loc)
		return goal.NewV(&Time{Time: &t})
	case hexadic:
		yearV := args[5]
		monthV := args[4]
		dayV := args[3]
		hourV := args[2]
		minV := args[1]
		secV := args[0]
		if !yearV.IsI() {
			return panicType("time.date year month day hour minute second", "year", yearV)
		}
		if !monthV.IsI() {
			return panicType("time.date year month day hour minute second", "month", monthV)
		}
		if !dayV.IsI() {
			return panicType("time.date year month day hour minute second", "day", dayV)
		}
		if !hourV.IsI() {
			return panicType("time.date year month day hour minute second", "hour", hourV)
		}
		if !minV.IsI() {
			return panicType("time.date year month day hour minute second", "minute", minV)
		}
		if !secV.IsI() {
			return panicType("time.date year month day hour minute second", "second", secV)
		}
		year := int(yearV.I())
		month := time.Month(monthV.I())
		day := int(dayV.I())
		hour := int(hourV.I())
		minute := int(minV.I())
		sec := int(secV.I())
		nsec := 0
		loc := time.UTC
		t := time.Date(year, month, day, hour, minute, sec, nsec, loc)
		return goal.NewV(&Time{Time: &t})
	case heptadic:
		yearV := args[6]
		monthV := args[5]
		dayV := args[4]
		hourV := args[3]
		minV := args[2]
		secV := args[1]
		nsecV := args[0]
		if !yearV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond", "year", yearV)
		}
		if !monthV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond", "month", monthV)
		}
		if !dayV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond", "day", dayV)
		}
		if !hourV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond", "hour", hourV)
		}
		if !minV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond", "minute", minV)
		}
		if !secV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond", "second", secV)
		}
		if !nsecV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond", "nanosecond", nsecV)
		}
		year := int(yearV.I())
		month := time.Month(monthV.I())
		day := int(dayV.I())
		hour := int(hourV.I())
		minute := int(minV.I())
		sec := int(secV.I())
		nsec := int(nsecV.I())
		loc := time.UTC
		t := time.Date(year, month, day, hour, minute, sec, nsec, loc)
		return goal.NewV(&Time{Time: &t})
	case octadic:
		yearV := args[7]
		monthV := args[6]
		dayV := args[5]
		hourV := args[4]
		minV := args[3]
		secV := args[2]
		nsecV := args[1]
		locV, ok := args[0].BV().(*Location)
		if !ok {
			return panicType("time.date year month day hour minute second nanosecond location", "location", args[0])
		}
		if !yearV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond location", "year", yearV)
		}
		if !monthV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond location", "month", monthV)
		}
		if !dayV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond location", "day", dayV)
		}
		if !hourV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond location", "hour", hourV)
		}
		if !minV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond location", "minute", minV)
		}
		if !secV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond location", "second", secV)
		}
		if !nsecV.IsI() {
			return panicType("time.date year month day hour minute second nanosecond location", "nanosecond", nsecV)
		}
		year := int(yearV.I())
		month := time.Month(monthV.I())
		day := int(dayV.I())
		hour := int(hourV.I())
		minute := int(minV.I())
		sec := int(secV.I())
		nsec := int(nsecV.I())
		loc := locV.Location
		t := time.Date(year, month, day, hour, minute, sec, nsec, loc)
		return goal.NewV(&Time{Time: &t})
	default:
		return goal.Panicf("time.date : too many arguments (%d), expects 1 to 8 arguments", len(args))
	}
}

// Implements time.now function.
func vfTimeNow(_ *goal.Context, _ []goal.V) goal.V {
	now := time.Now()
	return goal.NewV(&Time{&now})
}

// Implements time.unix function.
func vfTimeUnix(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			if x.IsI() {
				t := time.Unix(x.I(), 0)
				return goal.NewV(&Time{&t})
			}
			return panicType("time.unix time-or-i", "time-or-i", x)
		}
		ts := tLocal.Time.Unix()
		return goal.NewI(ts)
	case dyadic:
		if x.IsI() {
			y := args[0]
			if y.IsI() {
				sec := x.I()
				nsec := y.I()
				t := time.Unix(sec, nsec)
				return goal.NewV(&Time{&t})
			}
			return panicType("isec time.unix insec", "insec", y)
		}
		return panicType("isec time.unix insec", "isec", x)
	default:
		return goal.Panicf("time.unix : too many arguments (%d), expects 1 or 2 arguments", len(args))
	}
}

// Implements time.year function.
func vfTimeYear(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.year time", "time", x)
		}
		year := tLocal.Time.Year()
		return goal.NewI(int64(year))
	default:
		return goal.Panicf("time.year : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.yearday function.
func vfTimeYearDay(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.yearday time", "time", x)
		}
		yearday := tLocal.Time.YearDay()
		return goal.NewI(int64(yearday))
	default:
		return goal.Panicf("time.yearday : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.month function (January = 1).
func vfTimeMonth(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.month time", "time", x)
		}
		month := tLocal.Time.Month()
		return goal.NewI(int64(month))
	default:
		return goal.Panicf("time.month : too many arguments (%d), expects 1 argument", len(args))
	}
}

// CONSIDER: ISOWeek which is a tuple of year, week

// Implements time.day function, for day of month.
func vfTimeDay(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		switch xv := x.BV().(type) {
		case *Time:
			return goal.NewI(int64(xv.Time.Day()))
		case *goal.AV:
			r := make([]int64, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					r[i] = int64(t.Time.Day())
				} else {
					return goal.Panicf("time.day time : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAI(r)
		default:
			return panicType("time.day time", "time", x)
		}
	default:
		return goal.Panicf("time.day : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.weekday function, for day of week (Sunday = 0).
func vfTimeWeekDay(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		switch xv := x.BV().(type) {
		case *Time:
			return goal.NewI(int64(xv.Time.Weekday()))
		case *goal.AV:
			r := make([]int64, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					r[i] = int64(t.Time.Weekday())
				} else {
					return goal.Panicf("time.weekday time : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAI(r)
		default:
			return panicType("time.weekday time", "time", x)
		}
	default:
		return goal.Panicf("time.weekday : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.hour function.
func vfTimeHour(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		switch xv := x.BV().(type) {
		case *Time:
			return goal.NewI(int64(xv.Time.Hour()))
		case *goal.AV:
			r := make([]int64, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					r[i] = int64(t.Time.Hour())
				} else {
					return goal.Panicf("time.hour time : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAI(r)
		default:
			return panicType("time.hour time", "time", x)
		}
	default:
		return goal.Panicf("time.hour : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.minute function.
func vfTimeMinute(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		switch xv := x.BV().(type) {
		case *Time:
			return goal.NewI(int64(xv.Time.Minute()))
		case *goal.AV:
			r := make([]int64, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					r[i] = int64(t.Time.Minute())
				} else {
					return goal.Panicf("time.minute time : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAI(r)
		default:
			return panicType("time.minute time", "time", x)
		}
	default:
		return goal.Panicf("time.minute : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.second function.
func vfTimeSecond(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		switch xv := x.BV().(type) {
		case *Time:
			return goal.NewI(int64(xv.Time.Second()))
		case *goal.AV:
			r := make([]int64, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					r[i] = int64(t.Time.Second())
				} else {
					return goal.Panicf("time.second time : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAI(r)
		default:
			return panicType("time.second time", "time", x)
		}
	default:
		return goal.Panicf("time.second : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.millisecond function.
func vfTimeMillisecond(_ *goal.Context, args []goal.V) goal.V {
	return vfTimeMilliMicro("time.millisecond", time.Millisecond, args)
}

// Implements time.microsecond function.
func vfTimeMicrosecond(_ *goal.Context, args []goal.V) goal.V {
	return vfTimeMilliMicro("time.microsecond", time.Microsecond, args)
}

// Helper function for near-duplicate time.millisecond and time.microsecond implementations.
func vfTimeMilliMicro(goalName string, unit time.Duration, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		switch xv := x.BV().(type) {
		case *Time:
			return goal.NewI(int64(xv.Time.Nanosecond() / int(unit)))
		case *goal.AV:
			r := make([]int64, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					r[i] = int64(t.Time.Nanosecond() / int(unit))
				} else {
					return goal.Panicf("%v time : expected array of times, "+
						"but array has a %q value: %v", goalName, xi.Type(), xi)
				}
			}
			return goal.NewAI(r)
		default:
			return panicType(fmt.Sprintf("%v time", goalName), "time", x)
		}
	default:
		return goal.Panicf("%v : too many arguments (%d), expects 1 argument", goalName, len(args))
	}
}

// Implements time.nanosecond function.
func vfTimeNanosecond(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		switch xv := x.BV().(type) {
		case *Time:
			return goal.NewI(int64(xv.Time.Nanosecond()))
		case *goal.AV:
			r := make([]int64, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					r[i] = int64(t.Time.Nanosecond())
				} else {
					return goal.Panicf("time.nanosecond time : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAI(r)
		default:
			return panicType("time.nanosecond time", "time", x)
		}
	default:
		return goal.Panicf("time.nanosecond : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.zonename function.
func vfTimeZoneName(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		switch xv := x.BV().(type) {
		case *Time:
			name, _ := xv.Time.Zone()
			return goal.NewS(name)
		case *goal.AV:
			r := make([]string, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					name, _ := t.Time.Zone()
					r[i] = name
				} else {
					return goal.Panicf("time.zonename time : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAS(r)
		default:
			return goal.Panicf("time.zonename time : expected time or array of times, "+
				"but received a %q: %v", xv.Type(), xv)
		}
	default:
		return goal.Panicf("time.zonename : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.zoneoffset function.
func vfTimeZoneOffset(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		switch xv := x.BV().(type) {
		case *Time:
			_, offset := xv.Time.Zone()
			return goal.NewI(int64(offset))
		case *goal.AV:
			r := make([]int64, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					_, offset := t.Time.Zone()
					r[i] = int64(offset)
				} else {
					return goal.Panicf("time.zoneoffset time : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAI(r)
		default:
			return goal.Panicf("time.zoneoffset time : expected time or array of times, "+
				"but received a %q: %v", xv.Type(), xv)
		}
	default:
		return goal.Panicf("time.zoneoffset : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.location function.
func vfTimeLocation(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		switch xv := x.BV().(type) {
		case *Time:
			loc := xv.Time.Location()
			return goal.NewV(&Location{Location: loc})
		case *goal.AV:
			r := make([]goal.V, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					loc := t.Time.Location()
					r[i] = goal.NewV(&Location{Location: loc})
				} else {
					return goal.Panicf("time.location time : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAV(r)
		default:
			return panicType("time.location time", "time", x)
		}
	default:
		return goal.Panicf("time.location : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.locationstring function.
func vfTimeLocationString(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		switch xv := x.BV().(type) {
		case *Location:
			return goal.NewS(xv.Location.String())
		case *goal.AV:
			r := make([]string, xv.Len())
			for i, xi := range xv.Slice {
				if l, ok := xi.BV().(*Location); ok {
					r[i] = l.Location.String()
				} else {
					return goal.Panicf("time.locationstring time : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAS(r)
		default:
			return goal.Panicf("time.locationstring time : expected time or array of times, "+
				"but received a %q: %v", xv.Type(), xv)
		}
	default:
		return goal.Panicf("time.locationstring : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.fixedZone dyad.
func vfTimeFixedZone(_ *goal.Context, args []goal.V) goal.V {
	switch len(args) {
	case dyadic:
		x, ok := args[1].BV().(goal.S)
		if !ok {
			return panicType("time.fixedZone name offset-seconds-east-of-utc", "name", args[1])
		}
		y := args[0]
		if !y.IsI() {
			return panicType("time.fixedZone name offset-seconds-east-of-utc", "offset-seconds-east-of-utc", args[0])
		}
		loc := time.FixedZone(string(x), int(y.I()))
		return goal.NewV(&Location{Location: loc})
	default:
		return goal.Panicf("time.fixedZone : wrong number of arguments (%d), expects 2 arguments", len(args))
	}
}

// Implements time.loadlocation monad.
func vfTimeLoadLocation(_ *goal.Context, args []goal.V) goal.V {
	switch len(args) {
	case monadic:
		name, ok := args[0].BV().(goal.S)
		if !ok {
			return panicType("time.loadlocation name", "name", args[0])
		}
		loc, err := time.LoadLocation(string(name))
		// Can fail to load the system's IANA database
		if err != nil {
			return goal.Errorf("%v", err)
		}
		return goal.NewV(&Location{Location: loc})
	default:
		return goal.Panicf("time.loadlocation : wrong number of arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.unixmilli function.
//
// Given numbers, produces times; given times, produces numbers.
//
//nolint:dupl // yes, but not interested in further conditional logic here.
func vfTimeUnixMilli(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		if x.IsI() {
			t := time.UnixMilli(x.I())
			return goal.NewV(&Time{&t})
		}
		switch xv := x.BV().(type) {
		case *Time:
			return goal.NewI(xv.Time.UnixMilli())
		case *goal.AB:
			r := make([]goal.V, xv.Len())
			for i, xi := range xv.Slice {
				t := time.UnixMilli(int64(xi))
				r[i] = goal.NewV(&Time{&t})
			}
			return goal.NewAV(r)
		case *goal.AI:
			r := make([]goal.V, xv.Len())
			for i, xi := range xv.Slice {
				t := time.UnixMilli(xi)
				r[i] = goal.NewV(&Time{&t})
			}
			return goal.NewAV(r)
		case *goal.AV:
			r := make([]int64, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					r[i] = t.Time.UnixMilli()
				} else {
					return goal.Panicf("time.unixmilli time-or-i : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAI(r)
		default:
			return panicType("time.unixmilli time-or-i", "time", x)
		}
	default:
		return goal.Panicf("time.unixmilli : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.unixmicro function.
//
// Given numbers, produces times; given times, produces numbers.
//
//nolint:dupl // yes, but not interested in further conditional logic here.
func vfTimeUnixMicro(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		if x.IsI() {
			t := time.UnixMicro(x.I())
			return goal.NewV(&Time{&t})
		}
		switch xv := x.BV().(type) {
		case *Time:
			return goal.NewI(xv.Time.UnixMicro())
		case *goal.AB:
			r := make([]goal.V, xv.Len())
			for i, xi := range xv.Slice {
				t := time.UnixMicro(int64(xi))
				r[i] = goal.NewV(&Time{&t})
			}
			return goal.NewAV(r)
		case *goal.AI:
			r := make([]goal.V, xv.Len())
			for i, xi := range xv.Slice {
				t := time.UnixMicro(xi)
				r[i] = goal.NewV(&Time{&t})
			}
			return goal.NewAV(r)
		case *goal.AV:
			r := make([]int64, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					r[i] = t.Time.UnixMicro()
				} else {
					return goal.Panicf("time.unixmicro time-or-i : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAI(r)
		default:
			return panicType("time.unixmicro time-or-i", "time", x)
		}
	default:
		return goal.Panicf("time.unixmicro : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.unixnano function.
//
// Providing a time returns a number; providing a number returns a time.
// Given how Go's time API is designed, the latter case is handled by dividing
// the given number by time.Second so that the time.Unix(sec int64, nsec int64)
// function can be used to construct the appropriate time struct.
func vfTimeUnixNano(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		if x.IsI() {
			return goal.NewV(&Time{unixNano(x.I())})
		}
		switch xv := x.BV().(type) {
		case *Time:
			return goal.NewI(xv.Time.UnixNano())
		case *goal.AB:
			r := make([]goal.V, xv.Len())
			for i, xi := range xv.Slice {
				r[i] = goal.NewV(&Time{unixNano(int64(xi))})
			}
			return goal.NewAV(r)
		case *goal.AI:
			r := make([]goal.V, xv.Len())
			for i, xi := range xv.Slice {
				r[i] = goal.NewV(&Time{unixNano(xi)})
			}
			return goal.NewAV(r)
		default:
			return panicType("time.unixnano time-or-i", "time-or-i", x)
		}
	default:
		return goal.Panicf("time.unixnano : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Given nanoseconds from the Unix epoch, return a time.Time struct for that instant in time.
func unixNano(nanos int64) *time.Time {
	num, denom := nanos, int64(time.Second)
	secs, nsecs := num/denom, num%denom
	t := time.Unix(secs, nsecs)
	return &t
}

// Implements time.utc function.
func vfTimeUTC(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		switch xv := x.BV().(type) {
		case *Time:
			t := xv.Time.UTC()
			return goal.NewV(&Time{&t})
		case *goal.AV:
			r := make([]goal.V, xv.Len())
			for i, xi := range xv.Slice {
				if t, ok := xi.BV().(*Time); ok {
					t := t.Time.UTC()
					r[i] = goal.NewV(&Time{&t})
				} else {
					return goal.Panicf("time.utc time : expected array of times, "+
						"but array has a %q value: %v", xi.Type(), xi)
				}
			}
			return goal.NewAV(r)
		default:
			return panicType("time.utc time", "time", x)
		}
	default:
		return goal.Panicf("time.utc : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.parse function.
func vfTimeParse(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	layoutS, ok := x.BV().(goal.S)
	if !ok {
		return panicType("slayout time.parse svalue", "sformat", x)
	}
	layout := string(layoutS)
	switch len(args) {
	case dyadic:
		y := args[0]
		switch yv := y.BV().(type) {
		case goal.S:
			if !ok {
				return panicType("slayout time.parse svalue", "svalue", y)
			}
			t, err := time.Parse(layout, string(yv))
			if err != nil {
				return goal.NewPanicError(err)
			}
			return goal.NewV(&Time{&t})
		case *goal.AS:
			r := make([]goal.V, yv.Len())
			for i, yi := range yv.Slice {
				t, err := time.Parse(layout, yi)
				if err != nil {
					return goal.NewPanicError(err)
				}
				r[i] = goal.NewV(&Time{&t})
			}
			return goal.NewAV(r)
		default:
			return goal.Panicf("slayout time.parse svalue : svalue must be a string or array of strings, "+
				"but received a %q: %v", yv.Type(), yv)
		}
	default:
		return goal.Panicf("time.parse : too many arguments (%d), expects 2 arguments", len(args))
	}
}

// Implements time.format function.
func vfTimeFormat(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	formatS, ok := x.BV().(goal.S)
	if !ok {
		return panicType("sformat time.format time", "sformat", x)
	}
	format := string(formatS)
	switch len(args) {
	case dyadic:
		y := args[0]
		switch yv := y.BV().(type) {
		case *Time:
			s := yv.Time.Format(format)
			return goal.NewS(s)
		case *goal.AV:
			r := make([]string, yv.Len())
			for i, yi := range yv.Slice {
				if t, ok := yi.BV().(*Time); ok {
					r[i] = t.Time.Format(format)
				} else {
					return goal.Panicf("sformat time.format time : expected array of times, "+
						"but array has a %q value: %v", yi.Type(), yi)
				}
			}
			return goal.NewAS(r)
		default:
			return goal.Panicf("sformat time.format time : expected time or array of times, "+
				"but received a %q: %v", yv.Type(), yv)
		}
	default:
		return goal.Panicf("time.format : too many arguments (%d), expects 2 arguments", len(args))
	}
}

const (
	yearPos      = 0
	monthPos     = 1
	dayPos       = 2
	yearMonthDay = 3
)

// Implements time.add function.
//
// Accepts one time argument and one integer argument, returning a new time which is addition
// of the integer argument as nanoseconds to the time. Pervasive.
func vfTimeAdd(_ *goal.Context, args []goal.V) goal.V {
	if len(args) > dyadic {
		return goal.Panicf("time.add : too many arguments (%d), expects 2 arguments", len(args))
	}
	x, y := args[1], args[0]
	if x.IsI() {
		return addIV(x.I(), y)
	}
	switch xv := x.BV().(type) {
	case *Time:
		return addTimeV(xv, y)
	case *goal.AB:
		// return timeAddABV(xv, y)
		return goal.Panicf("not yet implemented")
	case *goal.AI:
		// return timeAddAIV(xv, y)
		return goal.Panicf("not yet implemented")
	case *goal.AV:
		// Check length of both args
		// if yv.Len() != xv.Len() {
		// return panicLength("X+Y", xv.Len(), yv.Len())
		// }
		// return timeAddAVV(xv, y)
		return goal.Panicf("not yet implemented")
	case *goal.D:
		// return timeAddDV(xv, y)
		return goal.Panicf("not yet implemented")
	default:
		return panicType("time-or-i1 time.add time-or-i2", "time-or-i1", x)
	}
}

func addIV(x int64, y goal.V) goal.V {
	switch yv := y.BV().(type) {
	case *Time:
		t := yv.Time.Add(time.Duration(x))
		return goal.NewV(&Time{&t})
	case *goal.AV:
		dur := time.Duration(x)
		r := make([]goal.V, yv.Len())
		for i, yi := range yv.Slice {
			if t, ok := yi.BV().(*Time); ok {
				t := t.Time.Add(dur)
				r[i] = goal.NewV(&Time{&t})
			}
		}
		return goal.NewAV(r)
	default:
		return panicType("time-or-i1 time.add time-or-i2", "time-or-i2", y)
	}
}

func addTimeV(x *Time, y goal.V) goal.V {
	if y.IsI() {
		t := x.Time.Add(time.Duration(y.I()))
		return goal.NewV(&Time{&t})
	}
	switch yv := y.BV().(type) {
	case *goal.AB:
		r := make([]goal.V, yv.Len())
		for i, yi := range yv.Slice {
			t := x.Time.Add(time.Duration(yi))
			r[i] = goal.NewV(&Time{&t})
		}
		return goal.NewAV(r)
	case *goal.AI:
		r := make([]goal.V, yv.Len())
		for i, yi := range yv.Slice {
			t := x.Time.Add(time.Duration(yi))
			r[i] = goal.NewV(&Time{&t})
		}
		return goal.NewAV(r)
	default:
		return panicType("time-or-i1 time.add time-or-i2", "time-or-i2", y)
	}
}

// Implements time.addDate function.
func vfTimeAddDate(_ *goal.Context, args []goal.V) goal.V {
	if len(args) > dyadic {
		return goal.Panicf("time.addDate : too many arguments (%d), expects 2 arguments", len(args))
	}
	x, y := args[1], args[0]
	switch xv := x.BV().(type) {
	case *Time:
		return adddateTimeV(xv, y)
	case *goal.AB:
		// return adddateABV(xv, y)
		return goal.Panicf("not yet implemented")
	case *goal.AI:
		// return adddateAIV(xv, y)
		return goal.Panicf("not yet implemented")
	case *goal.AV:
		// return adddateAVV(xv, y)
		return goal.Panicf("not yet implemented")
	case *goal.D:
		// return adddateDV(xv, y)
		return goal.Panicf("not yet implemented")
	default:
		return panicType("time-or-I1 time.addDate time-or-I2", "time-or-I1", x)
	}
}

func adddateTimeV(x *Time, y goal.V) goal.V {
	switch yv := y.BV().(type) {
	case *goal.AB:
		if yv.Len() != yearMonthDay {
			return goal.Panicf("time.addDate : I arg must have 3 items, had %d", yv.Len())
		}
		year := yv.At(yearPos).I()
		month := yv.At(monthPos).I()
		day := yv.At(dayPos).I()
		t := x.Time.AddDate(int(year), int(month), int(day))
		return goal.NewV(&Time{&t})
	case *goal.AI:
		if yv.Len() != yearMonthDay {
			return goal.Panicf("time.addDate : I arg must have 3 items, had %d", yv.Len())
		}
		year := yv.At(yearPos).I()
		month := yv.At(monthPos).I()
		day := yv.At(dayPos).I()
		t := x.Time.AddDate(int(year), int(month), int(day))
		return goal.NewV(&Time{&t})
	case *goal.AV:
		r := make([]goal.V, yv.Len())
		for j, yi := range yv.Slice {
			if yi.Len() != yearMonthDay {
				return goal.Panicf("time.addDate : each array must have 3 items, one had %d", yi.Len())
			}
			switch yiv := yi.BV().(type) {
			case *goal.AB:
				year := yiv.At(yearPos).I()
				month := yiv.At(monthPos).I()
				day := yiv.At(dayPos).I()
				t := x.Time.AddDate(int(year), int(month), int(day))
				r[j] = goal.NewV(&Time{&t})
			case *goal.AI:
				year := yiv.At(yearPos).I()
				month := yiv.At(monthPos).I()
				day := yiv.At(dayPos).I()
				t := x.Time.AddDate(int(year), int(month), int(day))
				r[j] = goal.NewV(&Time{&t})
			default:
				return goal.Panicf("time.addDate : each array must be numeric, one is %q: %v", yi.Type(), yi)
			}
		}
		return goal.NewAV(r)
	default:
		return panicType("time-or-i1 time.addDate time-or-i2", "time-or-i2", y)
	}
}

// Implements time.sub function.
func vfTimeSub(_ *goal.Context, args []goal.V) goal.V {
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
	goalContext.AssignGlobal("time.ANSIC", goal.NewS(time.ANSIC))
	goalContext.AssignGlobal("time.DateOnly", goal.NewS(time.DateOnly))
	goalContext.AssignGlobal("time.DateTime", goal.NewS(time.DateTime))
	goalContext.AssignGlobal("time.Kitchen", goal.NewS(time.Kitchen))
	goalContext.AssignGlobal("time.Layout", goal.NewS(time.Layout))
	goalContext.AssignGlobal("time.RFC1123", goal.NewS(time.RFC1123))
	goalContext.AssignGlobal("time.RFC1123Z", goal.NewS(time.RFC1123Z))
	goalContext.AssignGlobal("time.RFC3339", goal.NewS(time.RFC3339))
	goalContext.AssignGlobal("time.RFC3339Nano", goal.NewS(time.RFC3339Nano))
	goalContext.AssignGlobal("time.RFC822", goal.NewS(time.RFC822))
	goalContext.AssignGlobal("time.RFC822Z", goal.NewS(time.RFC822Z))
	goalContext.AssignGlobal("time.RFC850", goal.NewS(time.RFC850))
	goalContext.AssignGlobal("time.RubyDate", goal.NewS(time.RubyDate))
	goalContext.AssignGlobal("time.Stamp", goal.NewS(time.Stamp))
	goalContext.AssignGlobal("time.StampMicro", goal.NewS(time.StampMicro))
	goalContext.AssignGlobal("time.StampMilli", goal.NewS(time.StampMilli))
	goalContext.AssignGlobal("time.StampNano", goal.NewS(time.StampNano))
	goalContext.AssignGlobal("time.TimeOnly", goal.NewS(time.TimeOnly))
	goalContext.AssignGlobal("time.UnixDate", goal.NewS(time.UnixDate))

	local := time.Local
	locationLocal := Location{Location: local}
	goalContext.AssignGlobal("time.Local", goal.NewV(&locationLocal))
	utc := time.UTC
	locationUTC := Location{Location: utc}
	goalContext.AssignGlobal("time.UTC", goal.NewV(&locationUTC))

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
