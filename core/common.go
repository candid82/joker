package core

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

var ExitJoker func(rc int)

func SetExitJoker(fn func(rc int)) {
	ExitJoker = fn
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

func FileInfoMap(name string, info os.FileInfo) Map {
	m := EmptyArrayMap()
	m.Add(MakeKeyword("name"), MakeString(name))
	m.Add(MakeKeyword("size"), MakeInt(int(info.Size())))
	m.Add(MakeKeyword("mode"), MakeInt(int(info.Mode())))
	m.Add(MakeKeyword("modtime"), MakeTime(info.ModTime()))
	m.Add(MakeKeyword("dir?"), MakeBoolean(info.IsDir()))
	return m
}
