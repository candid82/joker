package base64

import (
	"encoding/base64"

	. "github.com/candid/joker/core"
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
	internBase64("decode-string", base64DecodeString)
}
