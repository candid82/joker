package core

import (
	"fmt"
	"io"
)

func formatSeq(seq Seq, w io.Writer, indent int) int {
	i := indent + 1
	fmt.Fprint(w, "(")
	obj := seq.First()
	if obj.Equals(SYMBOLS._if) {
		i = formatObject(obj, indent+1, w)
		seq = seq.Rest()
		if !seq.IsEmpty() {
			fmt.Fprint(w, " ")
			formatObject(seq.First(), i+1, w)
			seq = seq.Rest()
		}
		if !seq.IsEmpty() {
			fmt.Fprint(w, "\n")
			indent += 1
			writeIndent(w, indent+1)
		}
	}

	for iter := iter(seq); iter.HasNext(); {
		i = formatObject(iter.Next(), indent+1, w)
		if iter.HasNext() {
			fmt.Fprint(w, "\n")
			writeIndent(w, indent+1)
		}
	}
	fmt.Fprint(w, ")")
	return i + 1
}
