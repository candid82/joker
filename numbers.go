package main

import (
	"math/big"
)

type (
	Number interface {
		Object
		Int() Int
		Double() Double
		BigInt() *big.Int
		BigFloat() *big.Float
		Ratio() *big.Rat
	}
	Ops interface {
		Combine(ops Ops) Ops
		Add(Number, Number) Number
		Subtract(Number, Number) Number
		IsZero(Number) bool
	}
	IntOps      struct{}
	DoubleOps   struct{}
	BigIntOps   struct{}
	BigFloatOps struct{}
	RatioOps    struct{}
)

var (
	INT_OPS      = IntOps{}
	DOUBLE_OPS   = DoubleOps{}
	BIGINT_OPS   = BigIntOps{}
	BIGFLOAT_OPS = BigFloatOps{}
	RATIO_OPS    = RatioOps{}
)

func (ops IntOps) Combine(other Ops) Ops {
	return other
}

func (ops DoubleOps) Combine(other Ops) Ops {
	switch other.(type) {
	case BigFloatOps:
		return other
	default:
		return ops
	}
}

func (ops BigIntOps) Combine(other Ops) Ops {
	switch other.(type) {
	case IntOps:
		return ops
	default:
		return other
	}
}

func (ops BigFloatOps) Combine(other Ops) Ops {
	return ops
}

func (ops RatioOps) Combine(other Ops) Ops {
	switch other.(type) {
	case DoubleOps, BigFloatOps:
		return other
	default:
		return ops
	}
}

func GetOps(obj Object) Ops {
	switch obj.(type) {
	case Double:
		return DOUBLE_OPS
	case *BigInt:
		return BIGINT_OPS
	case *BigFloat:
		return BIGFLOAT_OPS
	case *Ratio:
		return RATIO_OPS
	default:
		return INT_OPS
	}
}

// Int conversions

func (i Int) Int() Int {
	return i
}

func (i Int) Double() Double {
	return Double(i)
}

func (i Int) BigInt() *big.Int {
	return big.NewInt(int64(i))
}

func (i Int) BigFloat() *big.Float {
	return big.NewFloat(float64(i))
}

func (i Int) Ratio() *big.Rat {
	return big.NewRat(int64(i), 1)
}

// Double conversions

func (d Double) Int() Int {
	return Int(d)
}

func (d Double) BigInt() *big.Int {
	return big.NewInt(int64(d))
}

func (d Double) Double() Double {
	return d
}

func (d Double) BigFloat() *big.Float {
	return big.NewFloat(float64(d))
}

func (d Double) Ratio() *big.Rat {
	res := big.Rat{}
	return res.SetFloat64(float64(d))
}

// BigInt conversions

func (b *BigInt) Int() Int {
	return Int(b.BigInt().Int64())
}

func (b *BigInt) BigInt() *big.Int {
	return (*big.Int)(b)
}

func (b *BigInt) Double() Double {
	return Double(b.BigInt().Int64())
}

func (b *BigInt) BigFloat() *big.Float {
	res := big.Float{}
	return res.SetInt(b.BigInt())
}

func (b *BigInt) Ratio() *big.Rat {
	res := big.Rat{}
	return res.SetInt(b.BigInt())
}

// BigFloat conversions

func (b *BigFloat) Int() Int {
	i, _ := b.BigFloat().Int64()
	return Int(i)
}

func (b *BigFloat) BigInt() *big.Int {
	bi, _ := b.BigFloat().Int(nil)
	return bi
}

func (b *BigFloat) Double() Double {
	f, _ := b.BigFloat().Float64()
	return Double(f)
}

func (b *BigFloat) BigFloat() *big.Float {
	return (*big.Float)(b)
}

func (b *BigFloat) Ratio() *big.Rat {
	res := big.Rat{}
	return res.SetFloat64(float64(b.Double()))
}

// Ratio conversions

func (r *Ratio) Int() Int {
	f, _ := r.Ratio().Float64()
	return Int(f)
}

func (r *Ratio) BigInt() *big.Int {
	f, _ := r.Ratio().Float64()
	return big.NewInt(int64(f))
}

func (r *Ratio) Double() Double {
	f, _ := r.Ratio().Float64()
	return Double(f)
}

func (r *Ratio) BigFloat() *big.Float {
	f, _ := r.Ratio().Float64()
	return big.NewFloat(f)
}

func (r *Ratio) Ratio() *big.Rat {
	return (*big.Rat)(r)
}

// Ops

// Add

func (ops IntOps) Add(x, y Number) Number {
	return x.Int() + y.Int()
}

func (ops DoubleOps) Add(x, y Number) Number {
	return x.Double() + y.Double()
}

func (ops BigIntOps) Add(x, y Number) Number {
	b := big.Int{}
	b.Add(x.BigInt(), y.BigInt())
	res := BigInt(b)
	return &res
}

func (ops BigFloatOps) Add(x, y Number) Number {
	b := big.Float{}
	b.Add(x.BigFloat(), y.BigFloat())
	res := BigFloat(b)
	return &res
}

func (ops RatioOps) Add(x, y Number) Number {
	r := big.Rat{}
	r.Add(x.Ratio(), y.Ratio())
	res := Ratio(r)
	return &res
}

// Subtract

func (ops IntOps) Subtract(x, y Number) Number {
	return x.Int() - y.Int()
}

func (ops DoubleOps) Subtract(x, y Number) Number {
	return x.Double() - y.Double()
}

func (ops BigIntOps) Subtract(x, y Number) Number {
	b := big.Int{}
	b.Sub(x.BigInt(), y.BigInt())
	res := BigInt(b)
	return &res
}

func (ops BigFloatOps) Subtract(x, y Number) Number {
	b := big.Float{}
	b.Sub(x.BigFloat(), y.BigFloat())
	res := BigFloat(b)
	return &res
}

func (ops RatioOps) Subtract(x, y Number) Number {
	r := big.Rat{}
	r.Sub(x.Ratio(), y.Ratio())
	res := Ratio(r)
	return &res
}

// IsZero

func (ops IntOps) IsZero(x Number) bool {
	return x.Int() == 0
}

func (ops DoubleOps) IsZero(x Number) bool {
	return x.Double() == 0
}

func (ops BigIntOps) IsZero(x Number) bool {
	return x.BigInt().Sign() == 0
}

func (ops BigFloatOps) IsZero(x Number) bool {
	return x.BigFloat().Sign() == 0
}

func (ops RatioOps) IsZero(x Number) bool {
	return x.Ratio().Sign() == 0
}
