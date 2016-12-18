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

var base64Namespace = GLOBAL_ENV.EnsureNamespace(MakeSymbol("base64"))

func internBase64(name string, proc Proc) {
	base64Namespace.Intern(MakeSymbol(name)).Value = proc
}

func init() {
	base64Namespace.ResetMeta(MakeMeta("Implements base64 encoding as specified by RFC 4648.", "1.0"))
	internBase64("decode-string", base64DecodeString)
}
