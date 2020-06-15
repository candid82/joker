package core

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"
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

func formatObject(obj Object, indent int, w io.Writer) int {
	if info := obj.GetInfo(); info != nil {
		fmt.Fprint(w, info.prefix)
		indent += utf8.RuneCountInString(info.prefix)
	}
	switch obj := obj.(type) {
	case Formatter:
		return obj.Format(w, indent)
	default:
		s := obj.ToString(true)
		fmt.Fprint(w, s)
		return indent + utf8.RuneCountInString(s)
	}
}

func isComment(obj Object) bool {
	if _, ok := obj.(Comment); ok {
		return true
	}
	info := obj.GetInfo()
	if info == nil {
		return false
	}
	return info.prefix == "^" || info.prefix == "#^" || info.prefix == "#_"
}

func maybeNewLine(w io.Writer, obj, nextObj Object, baseIndent, currentIndent int) int {
	if writeNewLines(w, obj, nextObj) > 0 {
		writeIndent(w, baseIndent)
		return baseIndent
	}
	fmt.Fprint(w, " ")
	return currentIndent + 1
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
