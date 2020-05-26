package math

import (
	"math"

	. "github.com/candid82/joker/core"
)

func modf(x float64) Object {
	i, f := math.Modf(x)
	res := EmptyVector()
	res = res.Conjoin(MakeDouble(i))
	res = res.Conjoin(MakeDouble(f))
	return res
}
