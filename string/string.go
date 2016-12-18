package string

import (
	"bytes"
	"strings"

	. "github.com/candid82/joker/core"
)

var stringNamespace = GLOBAL_ENV.EnsureNamespace(MakeSymbol("string"))

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

var split Proc = func(args []Object) Object {
	CheckArity(args, 2, 2)
	str := EnsureString(args, 0).S
	reg := EnsureRegex(args, 1).R
	indexes := reg.FindAllStringIndex(str, -1)
	lastStart := 0
	result := EmptyVector
	for _, el := range indexes {
		result = result.Conjoin(String{S: str[lastStart:el[0]]})
		lastStart = el[1]
	}
	result = result.Conjoin(String{S: str[lastStart:]})
	return result
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
	stringNamespace.ResetMeta(MakeMeta("Implements simple functions to manipulate strings.", "1.0"))
	intern("pad-right", padRight)
	intern("pad-left", padLeft)
	intern("split", split)
	intern("join", join)
	intern("ends-with?", endsWith)
	intern("starts-with?", startsWith)
	intern("replace", replace)
}
