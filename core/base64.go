package core

import (
	"encoding/base64"
)

var base64DecodeString Proc = func(args []Object) Object {
	decoded, err := base64.StdEncoding.DecodeString(ensureString(args, 0).s)
	if err != nil {
		panic(RT.newError("Invalid bas64 string: " + err.Error()))
	}
	return String{s: string(decoded)}
}

var base64Namespace = GLOBAL_ENV.EnsureNamespace(MakeSymbol("gclojure.base64"))

func internBase64(name string, proc Proc) {
	base64Namespace.intern(MakeSymbol(name)).value = proc
}

func init() {
	internBase64("decode-string", base64DecodeString)
}
