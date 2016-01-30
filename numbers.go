package main

import (
	"math/big"
)

type (
	Number interface {
		Object
		IsZero() bool
		Plus(Number) Number
	}
)

func (i Int) Plus(n Number) Number {
	switch n := n.(type) {
	case Int:
		return i + n
	case Double:
		return Double(i) + n
	case *BigInt:
		res := BigInt(*(&big.Int{}).Add((*big.Int)(n), big.NewInt(int64(i))))
		return &res
	case *BigFloat:
		res := BigFloat(*(&big.Float{}).Add((*big.Float)(n), big.NewFloat(float64(i))))
		return &res
	case *Ratio:
		res := Ratio(*(&big.Rat{}).Add((*big.Rat)(n), big.NewRat(int64(i), 1)))
		return &res
	default:
		panic(&EvalError{msg: "Type of " + n.ToString(false) + " is unknown"})
	}
}

func (d Double) Plus(n Number) Number {
	switch n := n.(type) {
	case Int:
		return d + Double(n)
	case Double:
		return d + n
	case *BigInt:
		return d + Double((*big.Int)(n).Int64())
	case *BigFloat:
		f, _ := (*big.Float)(n).Float64()
		return d + Double(f)
	case *Ratio:
		f, _ := (*big.Rat)(n).Float64()
		return d + Double(f)
	default:
		panic(&EvalError{msg: "Type of " + n.ToString(false) + " is unknown"})
	}
}

func (b *BigInt) Plus(n Number) Number {
	bi := (*big.Int)(b)
	switch n := n.(type) {
	case Int:
		res := BigInt(*(&big.Int{}).Add(bi, big.NewInt(int64(n))))
		return &res
	case Double:
		return Double(n) + Double(bi.Int64())
	case *BigInt:
		res := BigInt(*(&big.Int{}).Add((*big.Int)(n), bi))
		return &res
	case *BigFloat:
		s := (&big.Float{}).SetInt(bi)
		res := BigFloat(*(&big.Float{}).Add((*big.Float)(n), s))
		return &res
	case *Ratio:
		s := (&big.Rat{}).SetInt(bi)
		res := Ratio(*(&big.Rat{}).Add((*big.Rat)(n), s))
		return &res
	default:
		panic(&EvalError{msg: "Type of " + n.ToString(false) + " is unknown"})
	}
}

func (b *BigFloat) Plus(n Number) Number {
	bf := (*big.Float)(b)
	switch n := n.(type) {
	case Int:
		res := BigFloat(*(&big.Float{}).Add(bf, big.NewFloat(float64(n))))
		return &res
	case Double:
		res := BigFloat(*(&big.Float{}).Add(bf, big.NewFloat(float64(n))))
		return &res
	case *BigInt:
		s := (&big.Float{}).SetInt((*big.Int)(n))
		res := BigFloat(*(&big.Float{}).Add(bf, s))
		return &res
	case *BigFloat:
		res := BigFloat(*(&big.Float{}).Add((*big.Float)(n), bf))
		return &res
	case *Ratio:
		f, _ := (*big.Rat)(n).Float64()
		res := BigFloat(*(&big.Float{}).Add(bf, big.NewFloat(f)))
		return &res
	default:
		panic(&EvalError{msg: "Type of " + n.ToString(false) + " is unknown"})
	}
}

func (r *Ratio) Plus(n Number) Number {
	rt := (*big.Rat)(r)
	switch n := n.(type) {
	case Int:
		res := Ratio(*(&big.Rat{}).Add(rt, big.NewRat(int64(n), 1)))
		return &res
	case Double:
		f, _ := rt.Float64()
		return n + Double(f)
	case *BigInt:
		s := (&big.Rat{}).SetInt((*big.Int)(n))
		res := Ratio(*(&big.Rat{}).Add(rt, s))
		return &res
	case *BigFloat:
		f, _ := rt.Float64()
		res := BigFloat(*(&big.Float{}).Add((*big.Float)(n), big.NewFloat(f)))
		return &res
	case *Ratio:
		res := Ratio(*(&big.Rat{}).Add(rt, (*big.Rat)(n)))
		return &res
	default:
		panic(&EvalError{msg: "Type of " + n.ToString(false) + " is unknown"})
	}
}

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
