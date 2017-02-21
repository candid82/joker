package string

import (
	"bytes"
	"regexp"
	"strings"

	. "github.com/candid82/joker/core"
)

var stringNamespace = GLOBAL_ENV.EnsureNamespace(MakeSymbol("joker.string"))
var newLine *regexp.Regexp

func intern(name string, proc Proc) {
	stringNamespace.Intern(MakeSymbol(name)).Value = proc
}

var padRight Proc = func(args []Object) Object {
	CheckArity(args, 3, 3)
	str := EnsureString(args, 0).S
	pad := EnsureString(args, 1).S
	n := EnsureInt(args, 2).I
	for {
		str += pad
		if len(str) > n {
			return String{S: str[0:n]}
		}
	}
}

var padLeft Proc = func(args []Object) Object {
	CheckArity(args, 3, 3)
	str := EnsureString(args, 0).S
	pad := EnsureString(args, 1).S
	n := EnsureInt(args, 2).I
	for {
		str = pad + str
		if len(str) > n {
			return String{S: str[len(str)-n:]}
		}
	}
}

func splitString(s string, r *regexp.Regexp) Object {
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

var split Proc = func(args []Object) Object {
	CheckArity(args, 2, 2)
	str := EnsureString(args, 0).S
	reg := EnsureRegex(args, 1).R
	return splitString(str, reg)
}

var splitLines Proc = func(args []Object) Object {
	CheckArity(args, 1, 1)
	return splitString(EnsureString(args, 0).S, newLine)
}

var join Proc = func(args []Object) Object {
	CheckArity(args, 2, 2)
	sep := EnsureString(args, 0).S
	seq := EnsureSeqable(args, 1).Seq()
	var b bytes.Buffer
	for !seq.IsEmpty() {
		b.WriteString(seq.First().ToString(false))
		seq = seq.Rest()
		if !seq.IsEmpty() {
			b.WriteString(sep)
		}
	}
	return String{S: b.String()}
}

var endsWith Proc = func(args []Object) Object {
	CheckArity(args, 2, 2)
	str := EnsureString(args, 0).S
	substr := EnsureString(args, 1).S
	return Bool{B: strings.HasSuffix(str, substr)}
}

var startsWith Proc = func(args []Object) Object {
	CheckArity(args, 2, 2)
	str := EnsureString(args, 0).S
	substr := EnsureString(args, 1).S
	return Bool{B: strings.HasPrefix(str, substr)}
}

var replace Proc = func(args []Object) Object {
	CheckArity(args, 3, 3)
	str := EnsureString(args, 0).S
	old := EnsureString(args, 1).S
	new := EnsureString(args, 2).S
	return String{S: strings.Replace(str, old, new, -1)}
}

func init() {

	newLine, _ = regexp.Compile("\r?\n")

	stringNamespace.ResetMeta(MakeMeta(nil, "Implements simple functions to manipulate strings.", "1.0"))
	stringNamespace.InternVar("pad-right", padRight,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("s"), MakeSymbol("pad"), MakeSymbol("n"))),
			"Returns s padded with pad at the end to length n.", "1.0"))
	stringNamespace.InternVar("pad-left", padLeft,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("s"), MakeSymbol("pad"), MakeSymbol("n"))),
			"Returns s padded with pad at the beginning to length n.", "1.0"))
	stringNamespace.InternVar("split", split,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("s"), MakeSymbol("re"))),
			"Splits string on a regular expression. Returns vector of the splits.", "1.0"))
	stringNamespace.InternVar("split-lines", splitLines,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("s"))),
			"Splits string on \\n or \\r\\n. Returns vector of the splits.", "1.0"))
	stringNamespace.InternVar("join", join,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("separator"), MakeSymbol("coll"))),
			"Returns a string of all elements in coll, as returned by (seq coll), separated by a separator.", "1.0"))
	stringNamespace.InternVar("ends-with?", endsWith,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("s"), MakeSymbol("substr"))),
			"True if s ends with substr.", "1.0"))
	stringNamespace.InternVar("starts-with?", startsWith,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("s"), MakeSymbol("substr"))),
			"True if s starts with substr.", "1.0"))
	stringNamespace.InternVar("replace", replace,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("s"), MakeSymbol("old"), MakeSymbol("new"))),
			"Replaces all instances of string old with string new in string s.", "1.0"))
}
