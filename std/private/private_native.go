package private

import (
	. "github.com/candid82/joker/core"
)

func types() Object {
	res := EmptyArrayMap()
	for k, v := range TYPES {
		res.Add(String{S: *k}, v)
	}
	return res
}

func initNative() {
}
