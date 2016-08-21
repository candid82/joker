package os

import (
	"os"
	"strings"

	. "github.com/candid/gclojure/core"
)

var env Proc = func(args []Object) Object {
	res := EmptyArrayMap()
	for _, v := range os.Environ() {
		parts := strings.Split(v, "=")
		res.Add(String{S: parts[0]}, String{S: parts[1]})
	}
	return res
}

var osNamespace = GLOBAL_ENV.EnsureNamespace(MakeSymbol("gclojure.os"))

func intern(name string, proc Proc) {
	osNamespace.Intern(MakeSymbol(name)).Value = proc
}

func init() {
	intern("env", env)
}
