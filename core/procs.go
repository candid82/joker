package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

type (
	Phase int
)

const (
	READ Phase = iota
	PARSE
	EVAL
)

func ensureArrayMap(args []Object, index int) *ArrayMap {
	switch obj := args[index].(type) {
	case *ArrayMap:
		return obj
	default:
		panic(RT.newArgTypeError(index, obj, "Map"))
	}
}

var procMeta Proc = func(args []Object) Object {
	switch obj := args[0].(type) {
	case Meta:
		meta := obj.GetMeta()
		if meta != nil {
			return meta
		}
	}
	return NIL
}

var procWithMeta Proc = func(args []Object) Object {
	CheckArity(args, 2, 2)
	m := EnsureMeta(args, 0)
	if args[1].Equals(NIL) {
		return args[0]
	}
	return m.WithMeta(EnsureMap(args, 1))
}

var procIsZero Proc = func(args []Object) Object {
	n := EnsureNumber(args, 0)
	ops := GetOps(n)
	return Bool{B: ops.IsZero(n)}
}

var procIsPos Proc = func(args []Object) Object {
	n := EnsureNumber(args, 0)
	ops := GetOps(n)
	return Bool{B: ops.Gt(n, Int{I: 0})}
}

var procIsNeg Proc = func(args []Object) Object {
	n := EnsureNumber(args, 0)
	ops := GetOps(n)
	return Bool{B: ops.Lt(n, Int{I: 0})}
}

var procAdd Proc = func(args []Object) Object {
	x := AssertNumber(args[0], "")
	y := AssertNumber(args[1], "")
	ops := GetOps(x).Combine(GetOps(y))
	return ops.Add(x, y)
}

var procAddEx Proc = func(args []Object) Object {
	x := AssertNumber(args[0], "")
	y := AssertNumber(args[1], "")
	ops := GetOps(x).Combine(GetOps(y)).Combine(BIGINT_OPS)
	return ops.Add(x, y)
}

var procMultiply Proc = func(args []Object) Object {
	x := AssertNumber(args[0], "")
	y := AssertNumber(args[1], "")
	ops := GetOps(x).Combine(GetOps(y))
	return ops.Multiply(x, y)
}

var procMultiplyEx Proc = func(args []Object) Object {
	x := AssertNumber(args[0], "")
	y := AssertNumber(args[1], "")
	ops := GetOps(x).Combine(GetOps(y)).Combine(BIGINT_OPS)
	return ops.Multiply(x, y)
}

var procSubtract Proc = func(args []Object) Object {
	var a, b Object
	if len(args) == 1 {
		a = Int{I: 0}
		b = args[0]
	} else {
		a = args[0]
		b = args[1]
	}
	ops := GetOps(a).Combine(GetOps(b))
	return ops.Subtract(AssertNumber(a, ""), AssertNumber(b, ""))
}

var procSubtractEx Proc = func(args []Object) Object {
	var a, b Object
	if len(args) == 1 {
		a = Int{I: 0}
		b = args[0]
	} else {
		a = args[0]
		b = args[1]
	}
	ops := GetOps(a).Combine(GetOps(b)).Combine(BIGINT_OPS)
	return ops.Subtract(AssertNumber(a, ""), AssertNumber(b, ""))
}

var procDivide Proc = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	y := EnsureNumber(args, 1)
	ops := GetOps(x).Combine(GetOps(y))
	return ops.Divide(x, y)
}

var procQuot Proc = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	y := EnsureNumber(args, 1)
	ops := GetOps(x).Combine(GetOps(y))
	return ops.Quotient(x, y)
}

var procRem Proc = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	y := EnsureNumber(args, 1)
	ops := GetOps(x).Combine(GetOps(y))
	return ops.Rem(x, y)
}

var procBitNot Proc = func(args []Object) Object {
	x := AssertInt(args[0], "Bit operation not supported for "+args[0].GetType().ToString(false))
	return Int{I: ^x.I}
}

func AssertInts(args []Object) (Int, Int) {
	x := AssertInt(args[0], "Bit operation not supported for "+args[0].GetType().ToString(false))
	y := AssertInt(args[1], "Bit operation not supported for "+args[1].GetType().ToString(false))
	return x, y
}

var procBitAnd Proc = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I & y.I}
}

var procBitOr Proc = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I | y.I}
}

var procBitXor Proc = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I ^ y.I}
}

var procBitAndNot Proc = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I &^ y.I}
}

var procBitClear Proc = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I &^ (1 << uint(y.I))}
}

var procBitSet Proc = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I | (1 << uint(y.I))}
}

var procBitFlip Proc = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I ^ (1 << uint(y.I))}
}

var procBitTest Proc = func(args []Object) Object {
	x, y := AssertInts(args)
	return Bool{B: x.I&(1<<uint(y.I)) != 0}
}

var procBitShiftLeft Proc = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I << uint(y.I)}
}

var procBitShiftRight Proc = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I >> uint(y.I)}
}

var procUnsignedBitShiftRight Proc = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: int(uint(x.I) >> uint(y.I))}
}

var procExInfo Proc = func(args []Object) Object {
	CheckArity(args, 2, 3)
	res := &ExInfo{
		msg:  EnsureString(args, 0),
		data: ensureArrayMap(args, 1),
		rt:   RT.clone(),
	}
	if len(args) == 3 {
		res.cause = EnsureError(args, 2)
	}
	return res
}

var procExData Proc = func(args []Object) Object {
	return args[0].(*ExInfo).data
}

var procRegex Proc = func(args []Object) Object {
	r, err := regexp.Compile(EnsureString(args, 0).S)
	if err != nil {
		panic(RT.NewError("Invalid regex: " + err.Error()))
	}
	return Regex{R: r}
}

var procReSeq Proc = func(args []Object) Object {
	re := EnsureRegex(args, 0)
	s := EnsureString(args, 1)
	matches := re.R.FindAllStringSubmatch(s.S, -1)
	res := make([]Object, len(matches))
	for i, match := range matches {
		if len(match) == 1 {
			res[i] = String{S: match[0]}
		} else {
			v := EmptyVector
			for _, str := range match {
				v = v.Conjoin(String{S: str})
			}
			res[i] = v
		}
	}
	return &ArraySeq{arr: res}
}

var procReFind Proc = func(args []Object) Object {
	re := EnsureRegex(args, 0)
	s := EnsureString(args, 1)
	match := re.R.FindStringSubmatch(s.S)
	if len(match) == 1 {
		return String{S: match[0]}
	}
	v := EmptyVector
	for _, str := range match {
		v = v.Conjoin(String{S: str})
	}
	return v
}

var procRand Proc = func(args []Object) Object {
	r := rand.Float64()
	return Double{D: r}
}

var procIsSpecialSymbol Proc = func(args []Object) Object {
	return Bool{B: IsSpecialSymbol(args[0])}
}

var procSubs Proc = func(args []Object) Object {
	s := EnsureString(args, 0).S
	start := EnsureInt(args, 1).I
	end := len(s)
	if len(args) > 2 {
		end = EnsureInt(args, 2).I
	}
	if start < 0 || start > len(s) {
		panic(RT.NewError(fmt.Sprintf("String index out of range: %d", start)))
	}
	if end < 0 || end > len(s) {
		panic(RT.NewError(fmt.Sprintf("String index out of range: %d", end)))
	}
	return String{S: s[start:end]}
}

var procIntern = func(args []Object) Object {
	ns := EnsureNamespace(args, 0)
	sym := EnsureSymbol(args, 1)
	vr := ns.Intern(sym)
	if len(args) == 3 {
		vr.Value = args[2]
	}
	return vr
}

var procSetMeta = func(args []Object) Object {
	vr := EnsureVar(args, 0)
	meta := EnsureMap(args, 1)
	vr.meta = meta
	return NIL
}

var procAtom = func(args []Object) Object {
	res := &Atom{
		value: args[0],
	}
	if len(args) > 1 {
		m := NewHashMap(args[1:]...)
		if ok, v := m.Get(MakeKeyword("meta")); ok {
			res.meta = AssertMap(v, "")
		}
	}
	return res
}

var procDeref = func(args []Object) Object {
	return EnsureDeref(args, 0).Deref()
}

var procSwap = func(args []Object) Object {
	a := EnsureAtom(args, 0)
	f := EnsureCallable(args, 1)
	fargs := append([]Object{a.value}, args[2:]...)
	a.value = f.Call(fargs)
	return a.value
}

var procReset = func(args []Object) Object {
	a := EnsureAtom(args, 0)
	a.value = args[1]
	return a.value
}

var procAlterMeta = func(args []Object) Object {
	r := EnsureRef(args, 0)
	f := EnsureFn(args, 1)
	return r.AlterMeta(f, args[2:])
}

var procResetMeta = func(args []Object) Object {
	r := EnsureRef(args, 0)
	m := EnsureMap(args, 1)
	return r.ResetMeta(m)
}

var procEmpty = func(args []Object) Object {
	switch c := args[0].(type) {
	case Collection:
		return c.Empty()
	default:
		return NIL
	}
}

var procIsBound = func(args []Object) Object {
	vr := EnsureVar(args, 0)
	return Bool{B: vr.Value != nil}
}

func toNative(obj Object) interface{} {
	switch obj := obj.(type) {
	case Native:
		return obj.Native()
	default:
		return obj.ToString(false)
	}
}

var procFormat = func(args []Object) Object {
	s := EnsureString(args, 0)
	objs := args[1:]
	fargs := make([]interface{}, len(objs))
	for i, v := range objs {
		fargs[i] = toNative(v)
	}
	res := fmt.Sprintf(s.S, fargs...)
	return String{S: res}
}

var procSetMacro Proc = func(args []Object) Object {
	vr := args[0].(*Var)
	vr.isMacro = true
	return vr
}

var procList Proc = func(args []Object) Object {
	return NewListFrom(args...)
}

var procCons Proc = func(args []Object) Object {
	CheckArity(args, 2, 2)
	s := EnsureSeqable(args, 1).Seq()
	return s.Cons(args[0])
}

var procFirst Proc = func(args []Object) Object {
	CheckArity(args, 1, 1)
	s := EnsureSeqable(args, 0).Seq()
	return s.First()
}

var procNext Proc = func(args []Object) Object {
	CheckArity(args, 1, 1)
	s := EnsureSeqable(args, 0).Seq()
	res := s.Rest()
	if res.IsEmpty() {
		return NIL
	}
	return res
}

var procRest Proc = func(args []Object) Object {
	CheckArity(args, 1, 1)
	s := EnsureSeqable(args, 0).Seq()
	return s.Rest()
}

var procConj Proc = func(args []Object) Object {
	switch c := args[0].(type) {
	case Conjable:
		return c.Conj(args[1])
	case Seq:
		return c.Cons(args[1])
	default:
		panic(RT.NewError("conj's first argument must be a collection, got " + c.GetType().ToString(false)))
	}
}

var procSeq Proc = func(args []Object) Object {
	CheckArity(args, 1, 1)
	s := EnsureSeqable(args, 0).Seq()
	if s.IsEmpty() {
		return NIL
	}
	return s
}

var procIsInstance Proc = func(args []Object) Object {
	CheckArity(args, 2, 2)
	switch t := args[0].(type) {
	case *Type:
		return Bool{B: IsInstance(t, args[1])}
	default:
		panic(RT.NewError("First argument to instance? must be a type"))
	}
}

var procAssoc Proc = func(args []Object) Object {
	return EnsureAssociative(args, 0).Assoc(args[1], args[2])
}

var procEquals Proc = func(args []Object) Object {
	return Bool{B: args[0].Equals(args[1])}
}

var procCount Proc = func(args []Object) Object {
	switch obj := args[0].(type) {
	case Counted:
		return Int{I: obj.Count()}
	default:
		s := AssertSeqable(obj, "count not supported on this type: "+obj.GetType().ToString(false))
		return Int{I: SeqCount(s.Seq())}
	}
}

var procSubvec Proc = func(args []Object) Object {
	// TODO: implement proper Subvector structure
	v := args[0].(*Vector)
	start := args[1].(Int).I
	end := args[2].(Int).I
	if start > end {
		panic(RT.NewError(fmt.Sprintf("subvec's start index (%d) is greater than end index (%d)", start, end)))
	}
	subv := make([]Object, 0, end-start)
	for i := start; i < end; i++ {
		subv = append(subv, v.at(i))
	}
	return NewVectorFrom(subv...)
}

var procCast Proc = func(args []Object) Object {
	t := EnsureType(args, 0)
	if t.reflectType.Kind() == reflect.Interface &&
		args[1].GetType().reflectType.Implements(t.reflectType) ||
		args[1].GetType().reflectType == t.reflectType {
		return args[1]
	}
	panic(RT.NewError("Cannot cast " + args[1].GetType().ToString(false) + " to " + t.ToString(false)))
}

var procVec Proc = func(args []Object) Object {
	return NewVectorFromSeq(EnsureSeqable(args, 0).Seq())
}

var procHashMap Proc = func(args []Object) Object {
	if len(args)%2 != 0 {
		panic(RT.NewError("No value supplied for key " + args[len(args)-1].ToString(false)))
	}
	return NewHashMap(args...)
}

var procHashSet Proc = func(args []Object) Object {
	res := EmptySet()
	for i := 0; i < len(args); i++ {
		res.Add(args[i])
	}
	return res
}

var procStr Proc = func(args []Object) Object {
	var buffer bytes.Buffer
	for _, obj := range args {
		if !obj.Equals(NIL) {
			buffer.WriteString(obj.ToString(false))
		}
	}
	return String{S: buffer.String()}
}

var procSymbol Proc = func(args []Object) Object {
	if len(args) == 1 {
		return MakeSymbol(EnsureString(args, 0).S)
	}
	return Symbol{
		ns:   STRINGS.Intern(EnsureString(args, 0).S),
		name: STRINGS.Intern(EnsureString(args, 1).S),
	}
}

var procKeyword Proc = func(args []Object) Object {
	if len(args) == 1 {
		switch obj := args[0].(type) {
		case String:
			return MakeKeyword(obj.S)
		case Symbol:
			return Keyword{
				ns:   obj.ns,
				name: obj.name,
				hash: hashSymbol(obj.ns, obj.name),
			}
		default:
			return NIL
		}
	}
	ns := STRINGS.Intern(EnsureString(args, 0).S)
	name := STRINGS.Intern(EnsureString(args, 1).S)
	return Keyword{
		ns:   ns,
		name: name,
		hash: hashSymbol(ns, name),
	}
}

var procGensym Proc = func(args []Object) Object {
	return genSym(EnsureString(args, 0).S, "")
}

var procApply Proc = func(args []Object) Object {
	// TODO:
	// Stacktrace is broken. Need to somehow know
	// the name of the function passed ...
	f := EnsureCallable(args, 0)
	return f.Call(ToSlice(EnsureSeqable(args, 1).Seq()))
}

var procLazySeq Proc = func(args []Object) Object {
	return &LazySeq{
		fn: args[0].(*Fn),
	}
}

var procDelay Proc = func(args []Object) Object {
	return &Delay{
		fn: args[0].(*Fn),
	}
}

var procForce Proc = func(args []Object) Object {
	switch d := args[0].(type) {
	case *Delay:
		return d.Force()
	default:
		return d
	}
}

var procIdentical Proc = func(args []Object) Object {
	return Bool{B: args[0] == args[1]}
}

var procCompare Proc = func(args []Object) Object {
	k1, k2 := args[0], args[1]
	if k1.Equals(k2) {
		return Int{I: 0}
	}
	switch k2.(type) {
	case Nil:
		return Int{I: 1}
	}
	switch k1 := k1.(type) {
	case Nil:
		return Int{I: -1}
	case Comparable:
		return Int{I: k1.Compare(k2)}
	}
	panic(RT.NewError(fmt.Sprintf("%s (type: %s) is not a Comparable", k1.ToString(true), k1.GetType().ToString(false))))
}

var procInt Proc = func(args []Object) Object {
	switch obj := args[0].(type) {
	case Char:
		return Int{I: int(obj.ch)}
	case Number:
		return obj.Int()
	default:
		panic(RT.NewError(fmt.Sprintf("Cannot cast %s (type: %s) to Int", obj.ToString(true), obj.GetType().ToString(false))))
	}
}

var procNumber Proc = func(args []Object) Object {
	return AssertNumber(args[0], fmt.Sprintf("Cannot cast %s (type: %s) to Number", args[0].ToString(true), args[0].GetType().ToString(false)))
}

var procDouble Proc = func(args []Object) Object {
	n := AssertNumber(args[0], fmt.Sprintf("Cannot cast %s (type: %s) to Double", args[0].ToString(true), args[0].GetType().ToString(false)))
	return n.Double()
}

var procChar Proc = func(args []Object) Object {
	switch c := args[0].(type) {
	case Char:
		return c
	case Number:
		i := c.Int().I
		if i < MIN_RUNE || i > MAX_RUNE {
			panic(RT.NewError(fmt.Sprintf("Value out of range for char: %d", i)))
		}
		return Char{ch: rune(i)}
	default:
		panic(RT.NewError(fmt.Sprintf("Cannot cast %s (type: %s) to Char", c.ToString(true), c.GetType().ToString(false))))
	}
}

var procBool Proc = func(args []Object) Object {
	return Bool{B: toBool(args[0])}
}

var procNumerator Proc = func(args []Object) Object {
	bi := EnsureRatio(args, 0).r.Num()
	return &BigInt{b: *bi}
}

var procDenominator Proc = func(args []Object) Object {
	bi := EnsureRatio(args, 0).r.Denom()
	return &BigInt{b: *bi}
}

var procBigInt Proc = func(args []Object) Object {
	switch n := args[0].(type) {
	case Number:
		return &BigInt{b: *n.BigInt()}
	case String:
		bi := big.Int{}
		if _, ok := bi.SetString(n.S, 10); ok {
			return &BigInt{b: bi}
		}
		panic(RT.NewError("Invalid number format " + n.S))
	default:
		panic(RT.NewError(fmt.Sprintf("Cannot cast %s (type: %s) to BigInt", n.ToString(true), n.GetType().ToString(false))))
	}
}

var procBigFloat Proc = func(args []Object) Object {
	switch n := args[0].(type) {
	case Number:
		return &BigFloat{b: *n.BigFloat()}
	case String:
		b := big.Float{}
		if _, ok := b.SetString(n.S); ok {
			return &BigFloat{b: b}
		}
		panic(RT.NewError("Invalid number format " + n.S))
	default:
		panic(RT.NewError(fmt.Sprintf("Cannot cast %s (type: %s) to BigFloat", n.ToString(true), n.GetType().ToString(false))))
	}
}

var procNth Proc = func(args []Object) Object {
	n := EnsureNumber(args, 1).Int().I
	switch coll := args[0].(type) {
	case Indexed:
		if len(args) == 3 {
			return coll.TryNth(n, args[2])
		}
		return coll.Nth(n)
	case Seqable:
		if len(args) == 3 {
			return SeqTryNth(coll.Seq(), n, args[2])
		}
		return SeqNth(coll.Seq(), n)
	default:
		panic(RT.NewError("nth not supported on this type: " + coll.GetType().ToString(false)))
	}
}

var procLt Proc = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Bool{B: GetOps(a).Combine(GetOps(b)).Lt(a, b)}
}

var procLte Proc = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Bool{B: GetOps(a).Combine(GetOps(b)).Lte(a, b)}
}

var procGt Proc = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Bool{B: GetOps(a).Combine(GetOps(b)).Gt(a, b)}
}

var procGte Proc = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Bool{B: GetOps(a).Combine(GetOps(b)).Gte(a, b)}
}

var procEq Proc = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Bool{B: GetOps(a).Combine(GetOps(b)).Eq(a, b)}
}

var procMax Proc = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Max(a, b)
}

var procMin Proc = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Min(a, b)
}

var procIncEx Proc = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	ops := GetOps(x).Combine(BIGINT_OPS)
	return ops.Add(x, Int{I: 1})
}

var procDecEx Proc = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	ops := GetOps(x).Combine(BIGINT_OPS)
	return ops.Subtract(x, Int{I: 1})
}

var procInc Proc = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	ops := GetOps(x).Combine(INT_OPS)
	return ops.Add(x, Int{I: 1})
}

var procDec Proc = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	ops := GetOps(x).Combine(INT_OPS)
	return ops.Subtract(x, Int{I: 1})
}

var procPeek Proc = func(args []Object) Object {
	s := AssertStack(args[0], "")
	return s.Peek()
}

var procPop Proc = func(args []Object) Object {
	s := AssertStack(args[0], "")
	return s.Pop().(Object)
}

var procContains Proc = func(args []Object) Object {
	switch c := args[0].(type) {
	case Gettable:
		ok, _ := c.Get(args[1])
		if ok {
			return Bool{B: true}
		}
		return Bool{B: false}
	}
	panic(RT.NewError("contains? not supported on type " + args[0].GetType().ToString(false)))
}

var procGet Proc = func(args []Object) Object {
	switch c := args[0].(type) {
	case Gettable:
		ok, v := c.Get(args[1])
		if ok {
			return v
		}
	}
	if len(args) == 3 {
		return args[2]
	}
	return NIL
}

var procDissoc Proc = func(args []Object) Object {
	return EnsureMap(args, 0).Without(args[1])
}

var procDisj Proc = func(args []Object) Object {
	return EnsureSet(args, 0).Disjoin(args[1])
}

var procFind Proc = func(args []Object) Object {
	res := EnsureAssociative(args, 0).EntryAt(args[1])
	if res == nil {
		return NIL
	}
	return res
}

var procKeys Proc = func(args []Object) Object {
	return EnsureMap(args, 0).Keys()
}

var procVals Proc = func(args []Object) Object {
	return EnsureMap(args, 0).Vals()
}

var procRseq Proc = func(args []Object) Object {
	return EnsureReversible(args, 0).Rseq()
}

var procName Proc = func(args []Object) Object {
	return String{S: EnsureNamed(args, 0).Name()}
}

var procNamespace Proc = func(args []Object) Object {
	ns := EnsureNamed(args, 0).Namespace()
	if ns == "" {
		return NIL
	}
	return String{S: ns}
}

var procFindVar Proc = func(args []Object) Object {
	sym := EnsureSymbol(args, 0)
	if sym.ns == nil {
		panic(RT.NewError("find-var argument must be namespace-qualified symbol"))
	}
	if v, ok := GLOBAL_ENV.Resolve(sym); ok {
		return v
	}
	return NIL
}

var procSort Proc = func(args []Object) Object {
	cmp := EnsureComparator(args, 0)
	coll := EnsureSeqable(args, 1)
	s := SortableSlice{
		s:   ToSlice(coll.Seq()),
		cmp: cmp,
	}
	sort.Sort(s)
	return &ArraySeq{arr: s.s}
}

var procEval Proc = func(args []Object) Object {
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	expr := Parse(args[0], parseContext)
	return Eval(expr, nil)
}

var procType Proc = func(args []Object) Object {
	return args[0].GetType()
}

func printObject(obj Object, w io.Writer) {
	printReadably := toBool(GLOBAL_ENV.printReadably.Value)
	switch obj := obj.(type) {
	case Printer:
		obj.Print(w, printReadably)
	default:
		fmt.Fprint(w, obj.ToString(printReadably))
	}
}

var procPr Proc = func(args []Object) Object {
	n := len(args)
	if n > 0 {
		f := AssertIOWriter(GLOBAL_ENV.stdout.Value, "")
		for _, arg := range args[:n-1] {
			printObject(arg, f)
			fmt.Fprint(f, " ")
		}
		printObject(args[n-1], f)
	}
	return NIL
}

var procNewline Proc = func(args []Object) Object {
	f := AssertIOWriter(GLOBAL_ENV.stdout.Value, "")
	fmt.Fprintln(f)
	return NIL
}

func readFromReader(reader io.RuneReader) Object {
	r := NewReader(reader)
	obj, err := TryRead(r)
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
	return obj
}

func EnsureIOReader(args []Object, index int) io.Reader {
	switch c := args[index].(type) {
	case io.Reader:
		return c
	default:
		panic(RT.newArgTypeError(index, c, "IOReader"))
	}
}

func AssertIOReader(obj Object, msg string) io.Reader {
	switch c := obj.(type) {
	case io.Reader:
		return c
	default:
		if msg == "" {
			msg = fmt.Sprintf("Expected %s, got %s", "IOReader", obj.GetType().ToString(false))
		}
		panic(RT.NewError(msg))
	}
}

func EnsureIOWriter(args []Object, index int) io.Writer {
	switch c := args[index].(type) {
	case io.Writer:
		return c
	default:
		panic(RT.newArgTypeError(index, c, "IOWriter"))
	}
}

func AssertIOWriter(obj Object, msg string) io.Writer {
	switch c := obj.(type) {
	case io.Writer:
		return c
	default:
		if msg == "" {
			msg = fmt.Sprintf("Expected %s, got %s", "IOWriter", obj.GetType().ToString(false))
		}
		panic(RT.NewError(msg))
	}
}

var procRead Proc = func(args []Object) Object {
	f := EnsureIOReader(args, 0)
	return readFromReader(bufio.NewReader(f))
}

var procReadString Proc = func(args []Object) Object {
	CheckArity(args, 1, 1)
	return readFromReader(strings.NewReader(EnsureString(args, 0).S))
}

var procReadLine Proc = func(args []Object) Object {
	CheckArity(args, 0, 0)
	var line string
	f := AssertIOReader(GLOBAL_ENV.stdin.Value, "")
	fmt.Fscanln(f, &line)
	return String{S: line}
}

var procNanoTime Proc = func(args []Object) Object {
	return &BigInt{b: *big.NewInt(time.Now().UnixNano())}
}

var procMacroexpand1 Proc = func(args []Object) Object {
	switch s := args[0].(type) {
	case Seq:
		parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
		return macroexpand1(s, parseContext)
	default:
		return s
	}
}

func loadReader(reader *Reader) (Object, error) {
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	var lastObj Object = NIL
	for {
		obj, err := TryRead(reader)
		if err == io.EOF {
			return lastObj, nil
		}
		if err != nil {
			return nil, err
		}
		expr, err := TryParse(obj, parseContext)
		if err != nil {
			return nil, err
		}
		lastObj, err = TryEval(expr)
		if err != nil {
			return nil, err
		}
	}
}

var procLoadString Proc = func(args []Object) Object {
	s := EnsureString(args, 0)
	obj, err := loadReader(NewReader(strings.NewReader(s.S)))
	if err != nil {
		panic(err)
	}
	return obj
}

var procFindNamespace Proc = func(args []Object) Object {
	ns := GLOBAL_ENV.FindNamespace(EnsureSymbol(args, 0))
	if ns == nil {
		return NIL
	}
	return ns
}

var procCreateNamespace Proc = func(args []Object) Object {
	return GLOBAL_ENV.EnsureNamespace(EnsureSymbol(args, 0))
}

var procRemoveNamespace Proc = func(args []Object) Object {
	ns := GLOBAL_ENV.RemoveNamespace(EnsureSymbol(args, 0))
	if ns == nil {
		return NIL
	}
	return ns
}

var procAllNamespaces Proc = func(args []Object) Object {
	s := make([]Object, 0, len(GLOBAL_ENV.Namespaces))
	for _, ns := range GLOBAL_ENV.Namespaces {
		s = append(s, ns)
	}
	return &ArraySeq{arr: s}
}

var procNamespaceName Proc = func(args []Object) Object {
	return EnsureNamespace(args, 0).Name
}

var procNamespaceMap Proc = func(args []Object) Object {
	r := &ArrayMap{}
	for k, v := range EnsureNamespace(args, 0).mappings {
		r.Add(MakeSymbol(*k), v)
	}
	return r
}

var procNamespaceUnmap Proc = func(args []Object) Object {
	ns := EnsureNamespace(args, 0)
	sym := EnsureSymbol(args, 1)
	if sym.ns != nil {
		panic(RT.NewError("Can't unintern namespace-qualified symbol"))
	}
	delete(ns.mappings, sym.name)
	return NIL
}

var procVarNamespace Proc = func(args []Object) Object {
	v := EnsureVar(args, 0)
	return v.ns
}

var procRefer Proc = func(args []Object) Object {
	ns := EnsureNamespace(args, 0)
	sym := EnsureSymbol(args, 1)
	v := EnsureVar(args, 2)
	return ns.Refer(sym, v)
}

var procAlias Proc = func(args []Object) Object {
	EnsureNamespace(args, 0).AddAlias(EnsureSymbol(args, 1), EnsureNamespace(args, 2))
	return NIL
}

var procNamespaceAliases Proc = func(args []Object) Object {
	r := &ArrayMap{}
	for k, v := range EnsureNamespace(args, 0).aliases {
		r.Add(MakeSymbol(*k), v)
	}
	return r
}

var procNamespaceUnalias Proc = func(args []Object) Object {
	ns := EnsureNamespace(args, 0)
	sym := EnsureSymbol(args, 1)
	if sym.ns != nil {
		panic(RT.NewError("Alias can't be namespace-qualified"))
	}
	delete(ns.aliases, sym.name)
	return NIL
}

var procVarGet Proc = func(args []Object) Object {
	return EnsureVar(args, 0).Value
}

var procVarSet Proc = func(args []Object) Object {
	EnsureVar(args, 0).Value = args[1]
	return args[1]
}

var procNsResolve Proc = func(args []Object) Object {
	ns := EnsureNamespace(args, 0)
	sym := EnsureSymbol(args, 1)
	if sym.ns == nil && TYPES[*sym.name] != nil {
		return TYPES[*sym.name]
	}
	if vr, ok := GLOBAL_ENV.ResolveIn(ns, sym); ok {
		return vr
	}
	return NIL
}

var procArrayMap Proc = func(args []Object) Object {
	if len(args)%2 == 1 {
		panic(RT.NewError("No value supplied for key " + args[len(args)-1].ToString(false)))
	}
	res := EmptyArrayMap()
	for i := 0; i < len(args); i += 2 {
		res.Set(args[i], args[i+1])
	}
	return res
}

var procBuffer Proc = func(args []Object) Object {
	if len(args) > 0 {
		s := EnsureString(args, 0)
		return &Buffer{bytes.NewBufferString(s.S)}
	}
	return &Buffer{&bytes.Buffer{}}
}

var procSlurp Proc = func(args []Object) Object {
	b, err := ioutil.ReadFile(EnsureString(args, 0).S)
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
	return String{S: string(b)}
}

var procHash Proc = func(args []Object) Object {
	return Int{I: int(args[0].Hash())}
}

var procSh Proc = func(args []Object) Object {
	strs := make([]string, len(args))
	for i := range args {
		strs[i] = EnsureString(args, i).S
	}
	cmd := exec.Command(strs[0], strs[1:]...)
	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
	if err = cmd.Start(); err != nil {
		panic(RT.NewError(err.Error()))
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(stdoutReader)
	stdoutString := buf.String()
	buf = new(bytes.Buffer)
	buf.ReadFrom(stderrReader)
	stderrString := buf.String()
	if err = cmd.Wait(); err != nil {
		EmptyArrayMap().Assoc(MakeKeyword("success"), Bool{B: false})
	}
	res := EmptyArrayMap()
	res.Add(MakeKeyword("success"), Bool{B: true})
	res.Add(MakeKeyword("out"), String{S: stdoutString})
	res.Add(MakeKeyword("err"), String{S: stderrString})
	return res
}

var procLoadFile Proc = func(args []Object) Object {
	filename := EnsureString(args, 0)
	var reader *Reader
	f, err := os.Open(filename.S)
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
	reader = NewReader(bufio.NewReader(f))
	ProcessReader(reader, filename.S, EVAL)
	return NIL
}

var procIndexOf Proc = func(args []Object) Object {
	s := EnsureString(args, 0)
	ch := EnsureChar(args, 1)
	for i, r := range s.S {
		if r == ch.ch {
			return Int{I: i}
		}
	}
	return Int{I: -1}
}

var procLibPath Proc = func(args []Object) Object {
	sym := EnsureSymbol(args, 0)
	var file string
	if GLOBAL_ENV.file.Value == nil {
		var err error
		file, err = filepath.Abs("user")
		if err != nil {
			panic(RT.NewError(err.Error()))
		}
	} else {
		file = AssertString(GLOBAL_ENV.file.Value, "").S
	}
	ns := GLOBAL_ENV.CurrentNamespace().Name
	parts := strings.Split(ns.Name(), ".")
	for _ = range parts {
		file, _ = filepath.Split(file)
		file = file[:len(file)-1]
	}
	path := filepath.Join(append([]string{file}, strings.Split(sym.Name(), ".")...)...)
	return String{S: path + ".joke"}
}

func ProcessReader(reader *Reader, filename string, phase Phase) {
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	if filename != "" {
		currentFilename := parseContext.GlobalEnv.file.Value
		defer func() {
			parseContext.GlobalEnv.file.Value = currentFilename
		}()
		s, err := filepath.Abs(filename)
		if err != nil {
			panic(RT.NewError(err.Error()))
		}
		parseContext.GlobalEnv.file.Value = String{S: s}
	}
	for {
		obj, err := TryRead(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		if phase == READ {
			continue
		}
		expr, err := TryParse(obj, parseContext)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		if phase == PARSE {
			continue
		}
		_, err = TryEval(expr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}
}

func intern(name string, proc Proc) {
	GLOBAL_ENV.CoreNamespace.Intern(MakeSymbol(name)).Value = proc
}

func init() {
	rand.Seed(time.Now().UnixNano())
	GLOBAL_ENV.CoreNamespace.Intern(MakeSymbol("*assert*")).Value = Bool{B: true}

	intern("list**", procList)
	intern("cons*", procCons)
	intern("first*", procFirst)
	intern("next*", procNext)
	intern("rest*", procRest)
	intern("conj*", procConj)
	intern("seq*", procSeq)
	intern("instance?*", procIsInstance)
	intern("assoc*", procAssoc)
	intern("meta*", procMeta)
	intern("with-meta*", procWithMeta)
	intern("=*", procEquals)
	intern("count*", procCount)
	intern("subvec*", procSubvec)
	intern("cast*", procCast)
	intern("vec*", procVec)
	intern("hash-map*", procHashMap)
	intern("hash-set*", procHashSet)
	intern("str*", procStr)
	intern("symbol*", procSymbol)
	intern("gensym*", procGensym)
	intern("keyword*", procKeyword)
	intern("apply*", procApply)
	intern("lazy-seq*", procLazySeq)
	intern("delay*", procDelay)
	intern("force*", procForce)
	intern("identical*", procIdentical)
	intern("compare*", procCompare)
	intern("zero?*", procIsZero)
	intern("int*", procInt)
	intern("nth*", procNth)
	intern("<*", procLt)
	intern("<=*", procLte)
	intern(">*", procGt)
	intern(">=*", procGte)
	intern("==*", procEq)
	intern("inc'*", procIncEx)
	intern("inc*", procInc)
	intern("dec'*", procDecEx)
	intern("dec*", procDec)
	intern("add'*", procAddEx)
	intern("add*", procAdd)
	intern("multiply'*", procMultiplyEx)
	intern("multiply*", procMultiply)
	intern("divide*", procDivide)
	intern("subtract'*", procSubtractEx)
	intern("subtract*", procSubtract)
	intern("max*", procMax)
	intern("min*", procMin)
	intern("pos*", procIsPos)
	intern("neg*", procIsNeg)
	intern("quot*", procQuot)
	intern("rem*", procRem)
	intern("bit-not*", procBitNot)
	intern("bit-and*", procBitAnd)
	intern("bit-or*", procBitOr)
	intern("bit-xor*", procBitXor)
	intern("bit-and-not*", procBitAndNot)
	intern("bit-clear*", procBitClear)
	intern("bit-set*", procBitSet)
	intern("bit-flip*", procBitFlip)
	intern("bit-test*", procBitTest)
	intern("bit-shift-left*", procBitShiftLeft)
	intern("bit-shift-right*", procBitShiftRight)
	intern("unsigned-bit-shift-right*", procUnsignedBitShiftRight)
	intern("peek*", procPeek)
	intern("pop*", procPop)
	intern("contains?*", procContains)
	intern("get*", procGet)
	intern("dissoc*", procDissoc)
	intern("disj*", procDisj)
	intern("find*", procFind)
	intern("keys*", procKeys)
	intern("vals*", procVals)
	intern("rseq*", procRseq)
	intern("name*", procName)
	intern("namespace*", procNamespace)
	intern("find-var*", procFindVar)
	intern("sort*", procSort)
	intern("eval*", procEval)
	intern("type*", procType)
	intern("num*", procNumber)
	intern("double*", procDouble)
	intern("char*", procChar)
	intern("bool*", procBool)
	intern("numerator*", procNumerator)
	intern("denominator*", procDenominator)
	intern("bigint*", procBigInt)
	intern("bigfloat*", procBigFloat)
	intern("pr*", procPr)
	intern("newline*", procNewline)
	intern("read*", procRead)
	intern("read-line*", procReadLine)
	intern("read-string*", procReadString)
	intern("nano-time*", procNanoTime)
	intern("macroexpand-1*", procMacroexpand1)
	intern("load-string*", procLoadString)
	intern("find-ns*", procFindNamespace)
	intern("create-ns*", procCreateNamespace)
	intern("remove-ns*", procRemoveNamespace)
	intern("all-ns*", procAllNamespaces)
	intern("ns-name*", procNamespaceName)
	intern("ns-map*", procNamespaceMap)
	intern("ns-unmap*", procNamespaceUnmap)
	intern("var-ns*", procVarNamespace)
	intern("refer*", procRefer)
	intern("alias*", procAlias)
	intern("ns-aliases*", procNamespaceAliases)
	intern("ns-unalias*", procNamespaceUnalias)
	intern("var-get*", procVarGet)
	intern("var-set*", procVarSet)
	intern("ns-resolve*", procNsResolve)
	intern("array-map*", procArrayMap)
	intern("buffer*", procBuffer)
	intern("ex-info*", procExInfo)
	intern("ex-data*", procExData)
	intern("regex*", procRegex)
	intern("re-seq*", procReSeq)
	intern("re-find*", procReFind)
	intern("rand*", procRand)
	intern("special-symbol?*", procIsSpecialSymbol)
	intern("subs*", procSubs)
	intern("intern*", procIntern)
	intern("set-meta*", procSetMeta)
	intern("atom*", procAtom)
	intern("deref*", procDeref)
	intern("swap*", procSwap)
	intern("reset*", procReset)
	intern("alter-meta*", procAlterMeta)
	intern("reset-meta*", procResetMeta)
	intern("empty*", procEmpty)
	intern("bound?*", procIsBound)
	intern("format*", procFormat)
	intern("load-file*", procLoadFile)

	intern("set-macro*", procSetMacro)
	intern("sh", procSh)
	intern("slurp*", procSlurp)
	intern("hash*", procHash)

	intern("index-of*", procIndexOf)
	intern("lib-path*", procLibPath)

	currentNamespace := GLOBAL_ENV.ns.Value
	GLOBAL_ENV.ns.Value = GLOBAL_ENV.CoreNamespace
	data, err := Asset("data/core.clj")
	if err != nil {
		panic(RT.NewError("Could not load core.clj"))
	}
	reader := bytes.NewReader(data)
	ProcessReader(NewReader(reader), "", EVAL)
	GLOBAL_ENV.ns.Value = currentNamespace
}
