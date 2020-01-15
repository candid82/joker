// +build !go_spew

package core

var procGoSpew Proc = func(args []Object) (res Object) {
	return MakeBoolean(false)
}
