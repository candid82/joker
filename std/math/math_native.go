package math

import "math"
import . "github.com/candid82/joker/core"

func sin(x Number) float64 {
	return math.Sin(x.Double().D)
}

func cos(x Number) float64 {
	return math.Cos(x.Double().D)
}
