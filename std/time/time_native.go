package time

import (
	"time"

	. "github.com/candid82/joker/core"
)

func sleep(d int) Object {
	time.Sleep(time.Duration(d))
	return NIL
}
