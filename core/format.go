package core

import (
	"fmt"
	"io"
	"regexp"
)

func seqFirst(seq Seq, w io.Writer, indent int) (Seq, int) {
	if !seq.IsEmpty() {
		indent = formatObject(seq.First(), indent, w)
		seq = seq.Rest()
	}
	return seq, indent
}

func seqFirstAfterSpace(seq Seq, w io.Writer, indent int) (Seq, int) {
	if !seq.IsEmpty() {
		fmt.Fprint(w, " ")
		indent = formatObject(seq.First(), indent+1, w)
		seq = seq.Rest()
	}
	return seq, indent
}

func seqFirstAfterBreak(seq Seq, w io.Writer, indent int) (Seq, int) {
	if !seq.IsEmpty() {
		fmt.Fprint(w, "\n")
		writeIndent(w, indent)
		indent = formatObject(seq.First(), indent, w)
		seq = seq.Rest()
	}
	return seq, indent
}

func formatBindings(v *Vector, w io.Writer, indent int) int {
	fmt.Fprint(w, "[")
	newIndent := indent + 1
	for i := 0; i < v.count; i += 2 {
		newIndent = formatObject(v.at(i), indent+1, w)
		if i+1 < v.count {
			fmt.Fprint(w, " ")
			newIndent = formatObject(v.at(i+1), newIndent+1, w)
		}
		if i+2 < v.count {
			fmt.Fprint(w, "\n")
			writeIndent(w, indent+1)
		}
	}
	fmt.Fprint(w, "]")
	return newIndent + 1
}

func formatVectorVertically(v *Vector, w io.Writer, indent int) int {
	fmt.Fprint(w, "[")
	newIndent := indent + 1
	for i := 0; i < v.count; i++ {
		newIndent = formatObject(v.at(i), indent+1, w)
		if i+1 < v.count {
			fmt.Fprint(w, "\n")
			writeIndent(w, indent+1)
		}
	}
	fmt.Fprint(w, "]")
	return newIndent + 1
}

var defRegex *regexp.Regexp = regexp.MustCompile("def.+")
var ifRegex *regexp.Regexp = regexp.MustCompile("if(-.+)?")
var whenRegex *regexp.Regexp = regexp.MustCompile("when(-.+)?")

func isOneAndBodyExpr(obj Object) bool {
	switch s := obj.(type) {
	case Symbol:
		return defRegex.MatchString(*s.name) || ifRegex.MatchString(*s.name) || whenRegex.MatchString(*s.name)
	default:
		return false
	}
}

func isNewLine(obj, nextObj Object) bool {
	info, nextInfo := obj.GetInfo(), nextObj.GetInfo()
	return !(info == nil || nextInfo == nil || info.endLine == nextInfo.startLine)
}

func formatSeq(seq Seq, w io.Writer, indent int) int {
	i := indent + 1
	fmt.Fprint(w, "(")
	obj := seq.First()
	seq, i = seqFirst(seq, w, i)
	if obj.Equals(MakeSymbol("ns")) || isOneAndBodyExpr(obj) {
		seq, i = seqFirstAfterSpace(seq, w, i)

		// TODO: this should only apply to def*
		if docString, ok := seq.First().(String); ok {
			fmt.Fprint(w, "\n")
			writeIndent(w, indent+2)
			fmt.Fprint(w, "\"")
			fmt.Fprint(w, docString.ToString(false))
			fmt.Fprint(w, "\"")
			seq = seq.Rest()
		}
	} else if obj.Equals(MakeKeyword("require")) || obj.Equals(MakeKeyword("import")) {
		seq, _ = seqFirstAfterSpace(seq, w, i)
		for !seq.IsEmpty() {
			seq, _ = seqFirstAfterBreak(seq, w, i+1)
		}
	} else if obj.Equals(SYMBOLS.fn) || obj.Equals(SYMBOLS.catch) {
		if !seq.IsEmpty() {
			switch seq.First().(type) {
			case *Vector:
				seq, i = seqFirstAfterSpace(seq, w, i)
			default:
				seq, i = seqFirstAfterSpace(seq, w, i)
				seq, i = seqFirstAfterSpace(seq, w, i)
			}
		}
	} else if obj.Equals(SYMBOLS.let) || obj.Equals(SYMBOLS.loop) {
		if v, ok := seq.First().(*Vector); ok {
			fmt.Fprint(w, " ")
			i = formatBindings(v, w, i+1)
			seq = seq.Rest()
		}
	} else if obj.Equals(SYMBOLS.letfn) {
		if v, ok := seq.First().(*Vector); ok {
			fmt.Fprint(w, " ")
			i = formatVectorVertically(v, w, i+1)
			seq = seq.Rest()
		}
	} else if obj.Equals(SYMBOLS.do) || obj.Equals(SYMBOLS.try) || obj.Equals(SYMBOLS.finally) {
		// fmt.Fprint(w, "\n")
	} else {
		newIndent := indent + 1
		if !seq.IsEmpty() && !isNewLine(obj, seq.First()) {
			newIndent = i + 1
		}
		for !seq.IsEmpty() {
			nextObj := seq.First()
			if isNewLine(obj, nextObj) {
				seq, i = seqFirstAfterBreak(seq, w, newIndent)
			} else {
				seq, i = seqFirstAfterSpace(seq, w, i)
			}
			obj = nextObj
		}
		fmt.Fprint(w, ")")
		return i + 1
	}

	for !seq.IsEmpty() {
		nextObj := seq.First()
		if isNewLine(obj, nextObj) {
			seq, i = seqFirstAfterBreak(seq, w, indent+2)
		} else {
			seq, i = seqFirstAfterSpace(seq, w, i)
		}
		obj = nextObj
	}

	fmt.Fprint(w, ")")
	return i + 1
}
