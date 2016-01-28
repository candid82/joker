package main

import (
	"math/big"
)

type (
	Number interface {
		IsZero() bool
	}
)

func (i Int) IsZero() bool {
	return int(i) == 0
}

func (d Double) IsZero() bool {
	return float64(d) == 0.0
}

func (b *BigInt) IsZero() bool {
	return (*big.Int)(b).Sign() == 0
}

func (b *BigFloat) IsZero() bool {
	return (*big.Float)(b).Sign() == 0
}

func (b *Ratio) IsZero() bool {
	return (*big.Rat)(b).Sign() == 0
}
