package string

import (
	"bytes"
	"regexp"

	. "github.com/candid/gclojure/core"
)

var stringNamespace = GLOBAL_ENV.EnsureNamespace(MakeSymbol("gclojure.string"))

func intern(name string, proc Proc) {
	stringNamespace.Intern(MakeSymbol(name)).Value = proc
}

var padRight Proc = func(args []Object) Object {
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

var split Proc = func(args []Object) Object {
	str := EnsureString(args, 0).S
	reg := regexp.MustCompile(EnsureRegex(args, 1).R)
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

func init() {
	intern("pad-right", padRight)
	intern("split", split)
	intern("join", join)
}
