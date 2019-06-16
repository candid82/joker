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

func formatSeq(seq Seq, w io.Writer, indent int) int {
	i := indent + 1
	fmt.Fprint(w, "(")
	obj := seq.First()
	if obj.Equals(SYMBOLS._if) {
		seq, i = seqFirst(seq, w, i)
		seq, i = seqFirstAfterSpace(seq, w, i)
	} else if obj.Equals(SYMBOLS.fn) {
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
	}

	for !seq.IsEmpty() {
		seq, i = seqFirstAfterBreak(seq, w, indent + 2)
	}
	fmt.Fprint(w, ")")
	return i + 1
}
