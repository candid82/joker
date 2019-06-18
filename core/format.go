package core

import (
	"fmt"
	"io"
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

func formatSeq(seq Seq, w io.Writer, indent int) int {
	i := indent + 1
	fmt.Fprint(w, "(")
	obj := seq.First()
	if obj.Equals(SYMBOLS._if) {
		seq, i = seqFirst(seq, w, i)
		seq, i = seqFirstAfterSpace(seq, w, i)
	} else if obj.Equals(SYMBOLS.fn) || obj.Equals(SYMBOLS.catch) {
		seq, i = seqFirst(seq, w, i)
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
		seq, i = seqFirst(seq, w, i)
		if v, ok := seq.First().(*Vector); ok {
			fmt.Fprint(w, " ")
			i = formatBindings(v, w, i+1)
			seq = seq.Rest()
		}
	} else if obj.Equals(SYMBOLS.letfn) {
		seq, i = seqFirst(seq, w, i)
		if v, ok := seq.First().(*Vector); ok {
			fmt.Fprint(w, " ")
			i = formatVectorVertically(v, w, i+1)
			seq = seq.Rest()
		}
	} else if obj.Equals(SYMBOLS.do) || obj.Equals(SYMBOLS.try) || obj.Equals(SYMBOLS.finally) {
		seq, i = seqFirst(seq, w, i)
	} else {
		seq, i = seqFirst(seq, w, i)
		for !seq.IsEmpty() {
			seq, i = seqFirstAfterSpace(seq, w, i)
		}
		fmt.Fprint(w, ")")
		return i + 1
	}

	for !seq.IsEmpty() {
		seq, i = seqFirstAfterBreak(seq, w, indent+2)
	}
	fmt.Fprint(w, ")")
	return i + 1
}
