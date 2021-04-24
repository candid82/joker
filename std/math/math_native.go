package math

import (
	"fmt"
	"math"
	"math/big"

	. "github.com/candid82/joker/core"
)

func modf(x float64) Object {
	i, f := math.Modf(x)
	res := EmptyVector()
	res = res.Conjoin(MakeDouble(i))
	res = res.Conjoin(MakeDouble(f))
	return res
}

func precision(x Number) *big.Int {
	switch n := x.(type) {
	case Precision:
		return n.Precision()
	default:
		panic(RT.NewArgTypeError(0, x, "BigInt, BigFloat, Int, or Double"))
	}
}

func setPrecision(prec Number, n *big.Float) *big.Float {
	p := prec.Int().I
	if p < 0 {
		panic(RT.NewError(fmt.Sprintf("prec must be a non-negative Int, but is %d", p)))
	}
	return big.NewFloat(0).Copy(n).SetPrec(uint(p))
}
