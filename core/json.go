package core

import (
	"encoding/json"
	"fmt"
)

func toObject(v interface{}) Object {
	switch v := v.(type) {
	case string:
		return String{s: v}
	case float64:
		return Double{d: v}
	case bool:
		return Bool{b: v}
	case nil:
		return NIL
	case []interface{}:
		res := EmptyVector
		for _, v := range v {
			res = res.conj(toObject(v))
		}
		return res
	case map[string]interface{}:
		res := EmptyArrayMap()
		for k, v := range v {
			res.Add(String{s: k}, toObject(v))
		}
		return res
	default:
		panic(RT.newError(fmt.Sprintf("Unknown json value: %v", v)))
	}
}

var jsonReadString Proc = func(args []Object) Object {
	var v interface{}
	if err := json.Unmarshal([]byte(ensureString(args, 0).s), &v); err != nil {
		panic(RT.newError("Invalid json: " + err.Error()))
	}
	return toObject(v)
}

var jsonNamespace = GLOBAL_ENV.EnsureNamespace(MakeSymbol("gclojure.json"))

func internJson(name string, proc Proc) {
	jsonNamespace.intern(MakeSymbol(name)).value = proc
}

func init() {
	internJson("read-string", jsonReadString)
}
