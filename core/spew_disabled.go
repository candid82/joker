// +build !go_spew

package core

func Spew() {
}

func SpewThis(obj interface{}) {
}

func SpewObj(obj interface{}) string {
	return ""
}

var procGoSpew ProcFn = func(args []Object) (res Object) {
	return MakeBoolean(false)
}
