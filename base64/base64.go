package base64

import (
	"encoding/base64"

	. "github.com/candid82/joker/core"
)

var base64DecodeString Proc = func(args []Object) Object {
	decoded, err := base64.StdEncoding.DecodeString(EnsureString(args, 0).S)
	if err != nil {
		panic(RT.NewError("Invalid bas64 string: " + err.Error()))
	}
	return String{S: string(decoded)}
}

var base64Namespace = GLOBAL_ENV.EnsureNamespace(MakeSymbol("joker.base64"))

func init() {
	base64Namespace.ResetMeta(MakeMeta(nil, "Implements base64 encoding as specified by RFC 4648.", "1.0"))
	base64Namespace.InternVar("decode-string", base64DecodeString,
		MakeMeta(
			NewListFrom(NewVectorFrom(MakeSymbol("s"))),
			"Returns the bytes represented by the base64 string s.", "1.0"))
}
