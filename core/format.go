package core

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"unicode/utf8"
)

func seqFirst(seq Seq, w io.Writer, indent int) (Seq, int) {
	if !seq.IsEmpty() {
		indent = formatObject(seq.First(), indent, w)
		seq = seq.Rest()
	}
	return seq, indent
}

// TODO: maybe merge it with seqFirstAfterBreak
// or extract common part into a separate function
func seqFirstAfterSpace(seq Seq, w io.Writer, indent int, insideDefRecord bool) (Seq, Object, int) {
	var obj Object
	if !seq.IsEmpty() {
		fmt.Fprint(w, " ")
		obj = seq.First()
		// Seq handling here is needed to properly format methods
		// inside defrecord
		if s, ok := obj.(Seq); ok && !obj.Equals(NIL) {
			if info := obj.GetInfo(); info != nil {
				fmt.Fprint(w, info.prefix)
				indent += utf8.RuneCountInString(info.prefix)
			}
			indent = formatSeqEx(s, w, indent+1, insideDefRecord)
		} else {
			indent = formatObject(obj, indent+1, w)
		}
		seq = seq.Rest()
	}
	return seq, obj, indent
}

func writeNewLines(w io.Writer, prevObj Object, obj Object) int {
	cnt := newLineCount(prevObj, obj)
	for i := 0; i < cnt; i++ {
		fmt.Fprint(w, "\n")
	}
	return cnt
}

func seqFirstAfterBreak(prevObj Object, seq Seq, w io.Writer, indent int, insideDefRecord bool) (Seq, Object, int) {
	var obj Object
	if !seq.IsEmpty() {
		obj = seq.First()
		writeNewLines(w, prevObj, obj)
		writeIndent(w, indent)
		// Seq handling here is needed to properly format methods
		// inside defrecord
		if s, ok := obj.(Seq); ok && !obj.Equals(NIL) {
			if info := obj.GetInfo(); info != nil {
				fmt.Fprint(w, info.prefix)
				indent += utf8.RuneCountInString(info.prefix)
			}
			indent = formatSeqEx(s, w, indent, insideDefRecord)
		} else {
			indent = formatObject(obj, indent, w)
		}
		seq = seq.Rest()
	}
	return seq, obj, indent
}

func seqFirstAfterForcedBreak(seq Seq, w io.Writer, indent int) (Seq, Object, int) {
	var obj Object
	if !seq.IsEmpty() {
		obj = seq.First()
		fmt.Fprint(w, "\n")
		writeIndent(w, indent)
		indent = formatObject(obj, indent, w)
		seq = seq.Rest()
	}
	return seq, obj, indent
}

func formatBindings(v *Vector, w io.Writer, indent int) int {
	return v.Format(w, indent)
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
	if v.count > 0 {
		if isComment(v.at(v.count - 1)) {
			fmt.Fprint(w, "\n")
			writeIndent(w, indent+1)
			newIndent = indent + 1
		}
	}
	fmt.Fprint(w, "]")
	return newIndent + 1
}

var defRegex *regexp.Regexp = regexp.MustCompile("^def.*$")
var ifRegex *regexp.Regexp = regexp.MustCompile("^if(-.+)?$")
var whenRegex *regexp.Regexp = regexp.MustCompile("^when(-.+)?$")
var bodyIndentRegexes []*regexp.Regexp = []*regexp.Regexp{
	regexp.MustCompile("^(bound-fn|if|if-not|case|cond|cond->|cond->>|as->|go|condp|when|while|when-not|when-first|do|future|thread)$"),
	regexp.MustCompile("^(comment|doto|locking|proxy|with-[^\\s]*|reify)$"),
	regexp.MustCompile("^(defprotocol|extend|extend-protocol|extend-type|try|catch|finally|let|letfn|binding|loop|for|go-loop|alt!|alt!!)$"),
	regexp.MustCompile("^(doseq|dotimes|when-let|if-let|defstruct|struct-map|defmethod|testing|are|deftest|context|use-fixtures)$"),
	regexp.MustCompile("^(POST|GET|PUT|DELETE)"),
	regexp.MustCompile("^(handler-case|handle|dotrace|deftrace|match)$"),
}

func isOneAndBodyExpr(obj Object) bool {
	switch s := obj.(type) {
	case Symbol:
		return defRegex.MatchString(*s.name) ||
			ifRegex.MatchString(*s.name) ||
			whenRegex.MatchString(*s.name)
	default:
		return false
	}
}

func isBodyIndent(obj Object) bool {
	switch s := obj.(type) {
	case Symbol:
		for _, re := range bodyIndentRegexes {
			if re.MatchString(*s.name) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func isNewLine(obj, nextObj Object) bool {
	info, nextInfo := obj.GetInfo(), nextObj.GetInfo()
	return !(info == nil || nextInfo == nil || info.endLine == nextInfo.startLine)
}

func newLineCount(obj, nextObj Object) int {
	info, nextInfo := obj.GetInfo(), nextObj.GetInfo()
	if info == nil || nextInfo == nil {
		return 0
	}
	return nextInfo.startLine - info.endLine
}

func formatSeq(seq Seq, w io.Writer, indent int) int {
	return formatSeqEx(seq, w, indent, false)
}

type RequireSort []Object

func (rs RequireSort) Len() int      { return len(rs) }
func (rs RequireSort) Swap(i, j int) { rs[i], rs[j] = rs[j], rs[i] }
func (rs RequireSort) Less(i, j int) bool {
	a := rs[i]
	if s, ok := a.(Seqable); ok {
		a = s.Seq().First()
	}
	b := rs[j]
	if s, ok := b.(Seqable); ok {
		b = s.Seq().First()
	}
	return a.ToString(false) < b.ToString(false)
}

func sortRequire(seq Seq) Seq {
	s := RequireSort(ToSlice(seq))
	sort.Sort(s)
	return &ArraySeq{arr: s}
}

func formatSeqEx(seq Seq, w io.Writer, indent int, formatAsDef bool) int {
	i := indent + 1
	restIndent := indent + 2
	fmt.Fprint(w, "(")
	obj := seq.First()
	prevObj := obj
	seq, i = seqFirst(seq, w, i)
	isDefRecord := false
	if obj.Equals(SYMBOLS.defrecord) ||
		obj.Equals(SYMBOLS.defprotocol) ||
		obj.Equals(SYMBOLS.extendProtocol) ||
		obj.Equals(SYMBOLS.reify) ||
		obj.Equals(SYMBOLS.extendType) {
		isDefRecord = true
	}
	if obj.Equals(SYMBOLS.ns) || isOneAndBodyExpr(obj) {
		seq, prevObj, i = seqFirstAfterSpace(seq, w, i, isDefRecord)
	} else if obj.Equals(KEYWORDS.require) || obj.Equals(KEYWORDS._import) {
		seq = sortRequire(seq)
		seq, obj, _ = seqFirstAfterSpace(seq, w, i, isDefRecord)
		for !seq.IsEmpty() {
			seq, obj, _ = seqFirstAfterForcedBreak(seq, w, i+1)
		}
	} else if obj.Equals(SYMBOLS.catch) {
		if !seq.IsEmpty() {
			seq, prevObj, i = seqFirstAfterSpace(seq, w, i, isDefRecord)
			seq, prevObj, i = seqFirstAfterSpace(seq, w, i, isDefRecord)
		}
	} else if obj.Equals(SYMBOLS.fn) {
		if !seq.IsEmpty() {
			switch seq.First().(type) {
			case *Vector:
				seq, prevObj, i = seqFirstAfterSpace(seq, w, i, isDefRecord)
			case Symbol:
				seq, prevObj, i = seqFirstAfterSpace(seq, w, i, isDefRecord)
				seq, prevObj, i = seqFirstAfterSpace(seq, w, i, isDefRecord)
			default:
				if !isNewLine(obj, seq.First()) {
					restIndent = i + 1
				}
			}
		}
	} else if obj.Equals(SYMBOLS.let) || obj.Equals(SYMBOLS.loop) {
		if v, ok := seq.First().(*Vector); ok {
			fmt.Fprint(w, " ")
			i = formatBindings(v, w, i+1)
			prevObj = seq.First()
			seq = seq.Rest()
		}
	} else if obj.Equals(SYMBOLS.letfn) {
		if v, ok := seq.First().(*Vector); ok {
			fmt.Fprint(w, " ")
			i = formatVectorVertically(v, w, i+1)
			prevObj = seq.First()
			seq = seq.Rest()
		}
	} else if obj.Equals(SYMBOLS.do) || obj.Equals(SYMBOLS.try) || obj.Equals(SYMBOLS.finally) {
		if !seq.IsEmpty() && !isNewLine(obj, seq.First()) {
			restIndent = i + 1
		}
	} else if formatAsDef {
	} else if isBodyIndent(obj) {
		restIndent = indent + 2
	} else {
		// Indent function call arguments.
		restIndent = indent + 1
		if !seq.IsEmpty() && !isNewLine(obj, seq.First()) {
			restIndent = i + 1
		}
	}

	for !seq.IsEmpty() {
		nextObj := seq.First()
		if isNewLine(obj, nextObj) {
			seq, prevObj, i = seqFirstAfterBreak(prevObj, seq, w, restIndent, isDefRecord)
		} else {
			seq, prevObj, i = seqFirstAfterSpace(seq, w, i, isDefRecord)
		}
		obj = nextObj
	}

	if isComment(obj) {
		fmt.Fprint(w, "\n")
		writeIndent(w, restIndent)
		i = restIndent
	}

	fmt.Fprint(w, ")")
	return i + 1
}
