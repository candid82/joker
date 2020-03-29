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
