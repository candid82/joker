// +build !go_spew

package core

var procGoSpew ProcFn = func(args []Object) (res Object) {
	return MakeBoolean(false)
}
