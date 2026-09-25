package url

import (
	"net/url"

	. "github.com/candid82/joker/core"
)

func pathUnescape(s string) string {
	res, err := url.PathUnescape(s)
	if err != nil {
		panic(RT.NewError("Error unescaping string: " + err.Error()))
	}
	return res
}

func queryUnescape(s string) string {
	res, err := url.QueryUnescape(s)
	if err != nil {
		panic(RT.NewError("Error unescaping string: " + err.Error()))
	}
	return res
}

func parseQuery(s string) Object {
	values, err := url.ParseQuery(s)
	PanicOnErr(err)
	res := EmptyArrayMap()
	for k, v := range values {
		res.Add(MakeString(k), MakeStringVector(v))
	}
	return res
}
