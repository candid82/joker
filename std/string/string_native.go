package string

import (
	"bytes"
	"regexp"
	"unicode"

	. "github.com/candid82/joker/core"
)

var newLine *regexp.Regexp

func padRight(s, pad string, n int) string {
	for {
		s += pad
		if len(s) > n {
			return s[0:n]
		}
	}
}

func padLeft(s, pad string, n int) string {
	for {
		s = pad + s
		if len(s) > n {
			return s[len(s)-n:]
		}
	}
}

func split(s string, r *regexp.Regexp) Object {
	indexes := r.FindAllStringIndex(s, -1)
	lastStart := 0
	result := EmptyVector
	for _, el := range indexes {
		result = result.Conjoin(String{S: s[lastStart:el[0]]})
		lastStart = el[1]
	}
	result = result.Conjoin(String{S: s[lastStart:]})
	return result
}

func join(sep string, seqable Seqable) string {
	seq := seqable.Seq()
	var b bytes.Buffer
	for !seq.IsEmpty() {
		b.WriteString(seq.First().ToString(false))
		seq = seq.Rest()
		if !seq.IsEmpty() {
			b.WriteString(sep)
		}
	}
	return b.String()
}

func isBlank(s Object) bool {
	if s.Equals(NIL) {
		return true
	}
	str := AssertString(s, "").S
	for _, r := range str {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func init() {
	newLine, _ = regexp.Compile("\r?\n")
}
