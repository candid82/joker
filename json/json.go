package json

import (
	"encoding/json"
	"fmt"

	. "github.com/candid/gclojure/core"
)

func toObject(v interface{}) Object {
	switch v := v.(type) {
	case string:
		return MakeString(v)
	case float64:
		return Double{D: v}
	case bool:
		return Bool{B: v}
	case nil:
		return NIL
	case []interface{}:
		res := EmptyVector
		for _, v := range v {
			res = res.Conjoin(toObject(v))
		}
		return res
	case map[string]interface{}:
		res := EmptyArrayMap()
		for k, v := range v {
			res.Add(String{S: k}, toObject(v))
		}
		return res
	default:
		panic(RT.NewError(fmt.Sprintf("Unknown json value: %v", v)))
	}
}

var jsonReadString Proc = func(args []Object) Object {
	var v interface{}
	if err := json.Unmarshal([]byte(EnsureString(args, 0).S), &v); err != nil {
		panic(RT.NewError("Invalid json: " + err.Error()))
	}
	return toObject(v)
}

var jsonNamespace = GLOBAL_ENV.EnsureNamespace(MakeSymbol("gclojure.json"))

func internJson(name string, proc Proc) {
	jsonNamespace.Intern(MakeSymbol(name)).Value = proc
}

func init() {
	internJson("read-string", jsonReadString)
}
