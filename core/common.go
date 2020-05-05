package core

import (
	"fmt"
	"io"
	"os"
)

var exitCallbacks []func()

func ExitJoker(rc int) {
	for _, f := range exitCallbacks {
		f()
	}
	os.Exit(rc)
}

func OnExit(f func()) {
	exitCallbacks = append(exitCallbacks, f)
}

type suspendCallbackFns struct {
	leave func()
	rtn   func()
}

var SuspendStopFn func()
var suspendCallbacks []suspendCallbackFns

func SuspendJoker() bool {
	if SuspendStopFn == nil {
		return false
	}
	for _, fns := range suspendCallbacks {
		fns.leave()
	}
	SuspendStopFn()
	for _, fns := range suspendCallbacks {
		fns.rtn()
	}
	return true
}

func OnSuspend(leave func(), rtn func()) {
	suspendCallbacks = append(suspendCallbacks, suspendCallbackFns{leave, rtn})
}

func writeIndent(w io.Writer, n int) {
	space := []byte(" ")
	for i := 0; i < n; i++ {
		w.Write(space)
	}
}

func pprintObject(obj Object, indent int, w io.Writer) int {
	switch obj := obj.(type) {
	case Pprinter:
		return obj.Pprint(w, indent)
	default:
		s := obj.ToString(true)
		fmt.Fprint(w, s)
		return indent + len(s)
	}
}

func FileInfoMap(name string, info os.FileInfo) Map {
	m := EmptyArrayMap()
	m.Add(MakeKeyword("name"), MakeString(name))
	m.Add(MakeKeyword("size"), MakeInt(int(info.Size())))
	m.Add(MakeKeyword("mode"), MakeInt(int(info.Mode())))
	m.Add(MakeKeyword("modtime"), MakeTime(info.ModTime()))
	m.Add(MakeKeyword("dir?"), MakeBoolean(info.IsDir()))
	return m
}

func ToBool(obj Object) bool {
	switch obj := obj.(type) {
	case Nil:
		return false
	case Boolean:
		return obj.B
	default:
		return true
	}
}

func HomeDir() string {
	home, ok := os.LookupEnv("HOME")
	if !ok {
		home, _ = os.LookupEnv("USERPROFILE")
	}
	return home
}
