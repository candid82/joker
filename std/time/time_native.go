package time

import (
	. "github.com/candid82/joker/core"
	"time"
)

func inTimezone(t time.Time, tz string) time.Time {
	loc, err := time.LoadLocation(tz)
	PanicOnErr(err)
	return t.In(loc)
}

func parseInTimezone(layout string, value string, tz string) time.Time {
	loc, err := time.LoadLocation(tz)
	PanicOnErr(err)
	t, err := time.ParseInLocation(layout, value, loc)
	PanicOnErr(err)
	return t
}
