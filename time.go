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
func VFTimeDate(_ *goal.Context, args []goal.V) goal.V {
	// year int, month Month, day, hour, min, sec, nsec int, loc *Location
	switch len(args) {
	case monadic:
		yearV := args[0]
		if yearV.IsI() {
			year := int(yearV.I())
			month := time.January
			day := 1
			hour := 0
			min := 0
			sec := 0
			nsec := 0
			loc := time.UTC
			t := time.Date(year, month, day, hour, min, sec, nsec, loc)
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
		min := 0
		sec := 0
		nsec := 0
		loc := time.UTC
		t := time.Date(year, month, day, hour, min, sec, nsec, loc)
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
		min := 0
		sec := 0
		nsec := 0
		loc := time.UTC
		t := time.Date(year, month, day, hour, min, sec, nsec, loc)
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
		min := 0
		sec := 0
		nsec := 0
		loc := time.UTC
		t := time.Date(year, month, day, hour, min, sec, nsec, loc)
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
		min := int(minV.I())
		sec := 0
		nsec := 0
		loc := time.UTC
		t := time.Date(year, month, day, hour, min, sec, nsec, loc)
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
		min := int(minV.I())
		sec := int(secV.I())
		nsec := 0
		loc := time.UTC
		t := time.Date(year, month, day, hour, min, sec, nsec, loc)
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
		min := int(minV.I())
		sec := int(secV.I())
		nsec := int(nsecV.I())
		loc := time.UTC
		t := time.Date(year, month, day, hour, min, sec, nsec, loc)
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
		min := int(minV.I())
		sec := int(secV.I())
		nsec := int(nsecV.I())
		loc := locV.Location
		t := time.Date(year, month, day, hour, min, sec, nsec, loc)
		return goal.NewV(&Time{Time: &t})
	default:
		return goal.Panicf("time.date : too many arguments (%d), expects 1 to 8 arguments", len(args))
	}
}

// Implements time.now function.
func VFTimeNow(_ *goal.Context, _ []goal.V) goal.V {
	now := time.Now()
	return goal.NewV(&Time{&now})
}

// Implements time.unix function.
func VFTimeUnix(_ *goal.Context, args []goal.V) goal.V {
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
func VFTimeYear(_ *goal.Context, args []goal.V) goal.V {
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
func VFTimeYearDay(_ *goal.Context, args []goal.V) goal.V {
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
func VFTimeMonth(_ *goal.Context, args []goal.V) goal.V {
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
func VFTimeDay(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.day time", "time", x)
		}
		day := tLocal.Time.Day()
		return goal.NewI(int64(day))
	default:
		return goal.Panicf("time.day : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.weekday function, for day of week (Sunday = 0).
func VFTimeWeekDay(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.weekday time", "time", x)
		}
		weekday := tLocal.Time.Weekday()
		return goal.NewI(int64(weekday))
	default:
		return goal.Panicf("time.weekday : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.hour function.
func VFTimeHour(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.hour time", "time", x)
		}
		hour := tLocal.Time.Hour()
		return goal.NewI(int64(hour))
	default:
		return goal.Panicf("time.hour : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.minute function.
func VFTimeMinute(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.minute time", "time", x)
		}
		minute := tLocal.Time.Minute()
		return goal.NewI(int64(minute))
	default:
		return goal.Panicf("time.minute : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.second function.
func VFTimeSecond(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.second time", "time", x)
		}
		second := tLocal.Time.Second()
		return goal.NewI(int64(second))
	default:
		return goal.Panicf("time.second : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.millisecond function.
func VFTimeMillisecond(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.millisecond time", "time", x)
		}
		millisecond := tLocal.Time.Nanosecond() / int(time.Millisecond)
		return goal.NewI(int64(millisecond))
	default:
		return goal.Panicf("time.millisecond : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.microsecond function.
func VFTimeMicrosecond(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.microsecond time", "time", x)
		}
		microsecond := tLocal.Time.Nanosecond() / int(time.Microsecond)
		return goal.NewI(int64(microsecond))
	default:
		return goal.Panicf("time.microsecond : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.nanosecond function.
func VFTimeNanosecond(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.nanosecond time", "time", x)
		}
		nanosecond := tLocal.Time.Nanosecond()
		return goal.NewI(int64(nanosecond))
	default:
		return goal.Panicf("time.nanosecond : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.zonename function.
func VFTimeZoneName(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.zonename time", "time", x)
		}
		name, _ := tLocal.Time.Zone()
		return goal.NewS(name)
	default:
		return goal.Panicf("time.zonename : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.zoneoffset function.
func VFTimeZoneOffset(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.zoneoffset time", "time", x)
		}
		_, offset := tLocal.Time.Zone()
		return goal.NewI(int64(offset))
	default:
		return goal.Panicf("time.zoneoffset : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.location function.
func VFTimeLocation(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			return panicType("time.location time", "time", x)
		}
		loc := tLocal.Time.Location()
		return goal.NewV(&Location{Location: loc})
	default:
		return goal.Panicf("time.location : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.locationstring function.
func VFTimeLocationString(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		lloc, ok := x.BV().(*Location)
		if !ok {
			return panicType("time.locationstring location", "location", x)
		}
		name := lloc.Location.String()
		return goal.NewS(name)
	default:
		return goal.Panicf("time.locationstring : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.fixedzone dyad.
func VFTimeFixedZone(_ *goal.Context, args []goal.V) goal.V {
	switch len(args) {
	case dyadic:
		x, ok := args[1].BV().(goal.S)
		if !ok {
			return panicType("time.fixedzone name offset-seconds-east-of-utc", "name", args[1])
		}
		y := args[0]
		if !y.IsI() {
			return panicType("time.fixedzone name offset-seconds-east-of-utc", "offset-seconds-east-of-utc", args[0])
		}
		loc := time.FixedZone(string(x), int(y.I()))
		return goal.NewV(&Location{Location: loc})
	default:
		return goal.Panicf("time.fixedzone : wrong number of arguments (%d), expects 2 arguments", len(args))
	}
}

// Implements time.loadlocation monad.
func VFTimeLoadLocation(_ *goal.Context, args []goal.V) goal.V {
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
func VFTimeUnixMilli(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			if x.IsI() {
				t := time.UnixMilli(x.I())
				return goal.NewV(&Time{&t})
			}
			return panicType("time.unixmilli time-or-i", "time-or-i", x)
		}
		ts := tLocal.Time.UnixMilli()
		return goal.NewI(ts)
	default:
		return goal.Panicf("time.unixmilli : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.unixmicro function.
func VFTimeUnixMicro(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	switch len(args) {
	case monadic:
		tLocal, ok := x.BV().(*Time)
		if !ok {
			if x.IsI() {
				t := time.UnixMicro(x.I())
				return goal.NewV(&Time{&t})
			}
			return panicType("time.unixmicro time-or-i", "time-or-i", x)
		}
		ts := tLocal.Time.UnixMicro()
		return goal.NewI(ts)
	default:
		return goal.Panicf("time.unixmicro : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.unixnano function.
func VFTimeUnixNano(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	tLocal, ok := x.BV().(*Time)
	if !ok {
		return panicType("time.unixnano time", "time", x)
	}
	switch len(args) {
	case monadic:
		ts := tLocal.Time.UnixNano()
		return goal.NewI(ts)
	default:
		return goal.Panicf("time.unixnano : too many arguments (%d), expects 1 argument", len(args))
	}
}

// Implements time.utc function.
func VFTimeUTC(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	tLocal, ok := x.BV().(*Time)
	if !ok {
		return panicType("time.utc time", "time", x)
	}
	switch len(args) {
	case monadic:
		t := tLocal.Time.UTC()
		return goal.NewV(&Time{&t})
	default:
		return goal.Panicf("time.utc : too many arguments (%d), expects 1 argument", len(args))
	}
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
		return goal.NewV(&Time{&t})
	default:
		return goal.Panicf("time.parse : too many arguments (%d), expects 2 arguments", len(args))
	}
}

// Implements time.format function.
func VFTimeFormat(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	t, ok := x.BV().(*Time)
	if !ok {
		return panicType("time time.format sformat", "time", x)
	}
	switch len(args) {
	case dyadic:
		y := args[0]
		formatS, ok := y.BV().(goal.S)
		if !ok {
			return panicType("time time.format sformat", "sformat", y)
		}
		s := t.Time.Format(string(formatS))
		return goal.NewS(s)
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
//nolint:gocognit // I disagree
func VFTimeAdd(_ *goal.Context, args []goal.V) goal.V {
	x := args[len(args)-1]
	t1, ok := x.BV().(*Time)
	if !ok {
		return panicType("time time.add i", "time", x)
	}
	switch len(args) {
	case dyadic:
		y := args[0]
		//nolint:nestif // I disagree
		if !y.IsI() {
			if y.IsBV() {
				switch ai := y.BV().(type) {
				case *goal.AB:
					al := ai.Len()
					if al != yearMonthDay {
						return goal.Panicf("time.add : I arg must have 3 items, had %d", al)
					}
					year := ai.At(yearPos).I()
					month := ai.At(monthPos).I()
					day := ai.At(dayPos).I()
					t := t1.Time.AddDate(int(year), int(month), int(day))
					return goal.NewV(&Time{&t})
				case *goal.AI:
					al := ai.Len()
					if al != yearMonthDay {
						return goal.Panicf("time.add : I arg must have 3 items, had %d", al)
					}
					year := ai.At(yearPos).I()
					month := ai.At(monthPos).I()
					day := ai.At(dayPos).I()
					t := t1.Time.AddDate(int(year), int(month), int(day))
					return goal.NewV(&Time{&t})
				default:
					return panicType("time time.add I", "I", y)
				}
			}
			return panicType("time time.add i", "i", y)
		}
		t := t1.Time.Add(time.Duration(y.I()))
		return goal.NewV(&Time{&t})
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
