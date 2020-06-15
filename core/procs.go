package core

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var coreNamespaces []string

type (
	Phase        int
	Dialect      int
	StringReader interface {
		ReadString(delim byte) (s string, e error)
	}
)

const (
	READ Phase = iota
	FORMAT
	PARSE
	EVAL
	PRINT_IF_NOT_NIL
)

const VERSION = "v0.15.5"

const (
	CLJ Dialect = iota
	CLJS
	JOKER
	EDN
	UNKNOWN
)

func ExtractCallable(args []Object, index int) Callable {
	return EnsureCallable(args, index)
}

func ExtractObject(args []Object, index int) Object {
	return args[index]
}

func ExtractString(args []Object, index int) string {
	return EnsureString(args, index).S
}

func ExtractKeyword(args []Object, index int) string {
	return EnsureKeyword(args, index).ToString(false)
}

func ExtractStringable(args []Object, index int) string {
	return EnsureStringable(args, index).S
}

func ExtractStrings(args []Object, index int) []string {
	strs := make([]string, 0)
	for i := index; i < len(args); i++ {
		strs = append(strs, EnsureString(args, i).S)
	}
	return strs
}

func ExtractInt(args []Object, index int) int {
	return EnsureInt(args, index).I
}

func ExtractInteger(args []Object, index int) int {
	switch c := args[index].(type) {
	case Number:
		return c.Int().I
	default:
		panic(RT.NewArgTypeError(index, c, "Number"))
	}
}

func ExtractBoolean(args []Object, index int) bool {
	return EnsureBoolean(args, index).B
}

func ExtractChar(args []Object, index int) rune {
	return EnsureChar(args, index).Ch
}

func ExtractTime(args []Object, index int) time.Time {
	return EnsureTime(args, index).T
}

func ExtractDouble(args []Object, index int) float64 {
	return EnsureDouble(args, index).D
}

func ExtractNumber(args []Object, index int) Number {
	return EnsureNumber(args, index)
}

func ExtractRegex(args []Object, index int) *regexp.Regexp {
	return EnsureRegex(args, index).R
}

func ExtractSeqable(args []Object, index int) Seqable {
	return EnsureSeqable(args, index)
}

func ExtractMap(args []Object, index int) Map {
	return EnsureMap(args, index)
}

func ExtractIOReader(args []Object, index int) io.Reader {
	return Ensureio_Reader(args, index)
}

func ExtractIOWriter(args []Object, index int) io.Writer {
	return Ensureio_Writer(args, index)
}

var procMeta = func(args []Object) Object {
	CheckArity(args, 1, 1)
	switch obj := args[0].(type) {
	case Meta:
		meta := obj.GetMeta()
		if meta != nil {
			return meta
		}
	case *Type:
		meta := obj.GetMeta()
		if meta != nil {
			return meta
		}
	}
	return NIL
}

var procWithMeta = func(args []Object) Object {
	CheckArity(args, 2, 2)
	m := EnsureMeta(args, 0)
	if args[1].Equals(NIL) {
		return args[0]
	}
	return m.WithMeta(EnsureMap(args, 1))
}

var procIsZero = func(args []Object) Object {
	n := EnsureNumber(args, 0)
	ops := GetOps(n)
	return Boolean{B: ops.IsZero(n)}
}

var procIsPos = func(args []Object) Object {
	n := EnsureNumber(args, 0)
	ops := GetOps(n)
	return Boolean{B: ops.Gt(n, Int{I: 0})}
}

var procIsNeg = func(args []Object) Object {
	n := EnsureNumber(args, 0)
	ops := GetOps(n)
	return Boolean{B: ops.Lt(n, Int{I: 0})}
}

var procAdd = func(args []Object) Object {
	x := AssertNumber(args[0], "")
	y := AssertNumber(args[1], "")
	ops := GetOps(x).Combine(GetOps(y))
	return ops.Add(x, y)
}

var procAddEx = func(args []Object) Object {
	x := AssertNumber(args[0], "")
	y := AssertNumber(args[1], "")
	ops := GetOps(x).Combine(GetOps(y)).Combine(BIGINT_OPS)
	return ops.Add(x, y)
}

var procMultiply = func(args []Object) Object {
	x := AssertNumber(args[0], "")
	y := AssertNumber(args[1], "")
	ops := GetOps(x).Combine(GetOps(y))
	return ops.Multiply(x, y)
}

var procMultiplyEx = func(args []Object) Object {
	x := AssertNumber(args[0], "")
	y := AssertNumber(args[1], "")
	ops := GetOps(x).Combine(GetOps(y)).Combine(BIGINT_OPS)
	return ops.Multiply(x, y)
}

var procSubtract = func(args []Object) Object {
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

var procSubtractEx = func(args []Object) Object {
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

var procDivide = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	y := EnsureNumber(args, 1)
	ops := GetOps(x).Combine(GetOps(y))
	return ops.Divide(x, y)
}

var procQuot = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	y := EnsureNumber(args, 1)
	ops := GetOps(x).Combine(GetOps(y))
	return ops.Quotient(x, y)
}

var procRem = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	y := EnsureNumber(args, 1)
	ops := GetOps(x).Combine(GetOps(y))
	return ops.Rem(x, y)
}

var procBitNot = func(args []Object) Object {
	x := AssertInt(args[0], "Bit operation not supported for "+args[0].GetType().ToString(false))
	return Int{I: ^x.I}
}

func AssertInts(args []Object) (Int, Int) {
	x := AssertInt(args[0], "Bit operation not supported for "+args[0].GetType().ToString(false))
	y := AssertInt(args[1], "Bit operation not supported for "+args[1].GetType().ToString(false))
	return x, y
}

var procBitAnd = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I & y.I}
}

var procBitOr = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I | y.I}
}

var procBitXor = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I ^ y.I}
}

var procBitAndNot = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I &^ y.I}
}

var procBitClear = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I &^ (1 << uint(y.I))}
}

var procBitSet = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I | (1 << uint(y.I))}
}

var procBitFlip = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I ^ (1 << uint(y.I))}
}

var procBitTest = func(args []Object) Object {
	x, y := AssertInts(args)
	return Boolean{B: x.I&(1<<uint(y.I)) != 0}
}

var procBitShiftLeft = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I << uint(y.I)}
}

var procBitShiftRight = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: x.I >> uint(y.I)}
}

var procUnsignedBitShiftRight = func(args []Object) Object {
	x, y := AssertInts(args)
	return Int{I: int(uint(x.I) >> uint(y.I))}
}

var procExInfo = func(args []Object) Object {
	CheckArity(args, 2, 3)
	res := &ExInfo{
		rt: RT.clone(),
	}
	res.Add(KEYWORDS.message, EnsureString(args, 0))
	res.Add(KEYWORDS.data, EnsureMap(args, 1))
	if len(args) == 3 {
		res.Add(KEYWORDS.cause, EnsureError(args, 2))
	}
	return res
}

var procExData = func(args []Object) Object {
	if ok, res := args[0].(*ExInfo).Get(KEYWORDS.data); ok {
		return res
	}
	return NIL
}

var procExCause = func(args []Object) Object {
	if ok, res := args[0].(*ExInfo).Get(KEYWORDS.cause); ok {
		return res
	}
	return NIL
}

var procExMessage = func(args []Object) Object {
	return args[0].(Error).Message()
}

var procRegex = func(args []Object) Object {
	r, err := regexp.Compile(EnsureString(args, 0).S)
	if err != nil {
		panic(RT.NewError("Invalid regex: " + err.Error()))
	}
	return &Regex{R: r}
}

func reGroups(s string, indexes []int) Object {
	if indexes == nil {
		return NIL
	} else if len(indexes) == 2 {
		if indexes[0] == -1 {
			return NIL
		} else {
			return String{S: s[indexes[0]:indexes[1]]}
		}
	} else {
		v := EmptyVector()
		for i := 0; i < len(indexes); i += 2 {
			if indexes[i] == -1 {
				v = v.Conjoin(NIL)
			} else {
				v = v.Conjoin(String{S: s[indexes[i]:indexes[i+1]]})
			}
		}
		return v
	}
}

var procReSeq = func(args []Object) Object {
	re := EnsureRegex(args, 0)
	s := EnsureString(args, 1)
	matches := re.R.FindAllStringSubmatchIndex(s.S, -1)
	if matches == nil {
		return NIL
	}
	res := make([]Object, len(matches))
	for i, match := range matches {
		res[i] = reGroups(s.S, match)
	}
	return &ArraySeq{arr: res}
}

var procReFind = func(args []Object) Object {
	re := EnsureRegex(args, 0)
	s := EnsureString(args, 1)
	match := re.R.FindStringSubmatchIndex(s.S)
	return reGroups(s.S, match)
}

var procRand = func(args []Object) Object {
	r := rand.Float64()
	return Double{D: r}
}

var procIsSpecialSymbol = func(args []Object) Object {
	return Boolean{B: IsSpecialSymbol(args[0])}
}

var procSubs = func(args []Object) Object {
	s := EnsureString(args, 0).S
	start := EnsureInt(args, 1).I
	slen := utf8.RuneCountInString(s)
	end := slen
	if len(args) > 2 {
		end = EnsureInt(args, 2).I
	}
	if start < 0 || start > slen {
		panic(RT.NewError(fmt.Sprintf("String index out of range: %d", start)))
	}
	if end < 0 || end > slen {
		panic(RT.NewError(fmt.Sprintf("String index out of range: %d", end)))
	}
	return String{S: string([]rune(s)[start:end])}
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
		if ok, v := m.Get(KEYWORDS.meta); ok {
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

var procSwapVals = func(args []Object) Object {
	a := EnsureAtom(args, 0)
	f := EnsureCallable(args, 1)
	fargs := append([]Object{a.value}, args[2:]...)
	oldValue := a.value
	a.value = f.Call(fargs)
	return NewVectorFrom(oldValue, a.value)
}

var procReset = func(args []Object) Object {
	a := EnsureAtom(args, 0)
	a.value = args[1]
	return a.value
}

var procResetVals = func(args []Object) Object {
	a := EnsureAtom(args, 0)
	oldValue := a.value
	a.value = args[1]
	return NewVectorFrom(oldValue, a.value)
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
	return Boolean{B: vr.Value != nil}
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

var procList = func(args []Object) Object {
	return NewListFrom(args...)
}

var procCons = func(args []Object) Object {
	CheckArity(args, 2, 2)
	s := EnsureSeqable(args, 1).Seq()
	return s.Cons(args[0])
}

var procFirst = func(args []Object) Object {
	CheckArity(args, 1, 1)
	s := EnsureSeqable(args, 0).Seq()
	return s.First()
}

var procNext = func(args []Object) Object {
	CheckArity(args, 1, 1)
	s := EnsureSeqable(args, 0).Seq()
	res := s.Rest()
	if res.IsEmpty() {
		return NIL
	}
	return res
}

var procRest = func(args []Object) Object {
	CheckArity(args, 1, 1)
	s := EnsureSeqable(args, 0).Seq()
	return s.Rest()
}

var procConj = func(args []Object) Object {
	switch c := args[0].(type) {
	case Conjable:
		return c.Conj(args[1])
	case Seq:
		return c.Cons(args[1])
	default:
		panic(RT.NewError("conj's first argument must be a collection, got " + c.GetType().ToString(false)))
	}
}

var procSeq = func(args []Object) Object {
	CheckArity(args, 1, 1)
	s := EnsureSeqable(args, 0).Seq()
	if s.IsEmpty() {
		return NIL
	}
	return s
}

var procIsInstance = func(args []Object) Object {
	CheckArity(args, 2, 2)
	t := EnsureType(args, 0)
	return Boolean{B: IsInstance(t, args[1])}
}

var procAssoc = func(args []Object) Object {
	return EnsureAssociative(args, 0).Assoc(args[1], args[2])
}

var procEquals = func(args []Object) Object {
	return Boolean{B: args[0].Equals(args[1])}
}

var procCount = func(args []Object) Object {
	switch obj := args[0].(type) {
	case Counted:
		return Int{I: obj.Count()}
	default:
		s := AssertSeqable(obj, "count not supported on this type: "+obj.GetType().ToString(false))
		return Int{I: SeqCount(s.Seq())}
	}
}

var procSubvec = func(args []Object) Object {
	// TODO: implement proper Subvector structure
	v := EnsureVector(args, 0)
	start := EnsureInt(args, 1).I
	end := EnsureInt(args, 2).I
	if start > end {
		panic(RT.NewError(fmt.Sprintf("subvec's start index (%d) is greater than end index (%d)", start, end)))
	}
	subv := make([]Object, 0, end-start)
	for i := start; i < end; i++ {
		subv = append(subv, v.at(i))
	}
	return NewVectorFrom(subv...)
}

var procCast = func(args []Object) Object {
	t := EnsureType(args, 0)
	if t.reflectType.Kind() == reflect.Interface &&
		args[1].GetType().reflectType.Implements(t.reflectType) ||
		args[1].GetType().reflectType == t.reflectType {
		return args[1]
	}
	panic(RT.NewError("Cannot cast " + args[1].GetType().ToString(false) + " to " + t.ToString(false)))
}

var procVec = func(args []Object) Object {
	return NewVectorFromSeq(EnsureSeqable(args, 0).Seq())
}

var procHashMap = func(args []Object) Object {
	if len(args)%2 != 0 {
		panic(RT.NewError("No value supplied for key " + args[len(args)-1].ToString(false)))
	}
	return NewHashMap(args...)
}

var procHashSet = func(args []Object) Object {
	res := EmptySet()
	for i := 0; i < len(args); i++ {
		res.Add(args[i])
	}
	return res
}

func str(args ...Object) string {
	var buffer bytes.Buffer
	for _, obj := range args {
		if !obj.Equals(NIL) {
			t := obj.GetType()
			// TODO: this is a hack. Rethink escape parameter in ToString
			escaped := (t == TYPE.String) || (t == TYPE.Char) || (t == TYPE.Regex)
			buffer.WriteString(obj.ToString(!escaped))
		}
	}
	return buffer.String()
}

var procStr = func(args []Object) Object {
	return String{S: str(args...)}
}

var procSymbol = func(args []Object) Object {
	if len(args) == 1 {
		return MakeSymbol(EnsureString(args, 0).S)
	}
	var ns *string = nil
	if !args[0].Equals(NIL) {
		ns = STRINGS.Intern(EnsureString(args, 0).S)
	}
	return Symbol{
		ns:   ns,
		name: STRINGS.Intern(EnsureString(args, 1).S),
	}
}

var procKeyword = func(args []Object) Object {
	if len(args) == 1 {
		switch obj := args[0].(type) {
		case String:
			return MakeKeyword(obj.S)
		case Symbol:
			return Keyword{
				ns:   obj.ns,
				name: obj.name,
				hash: hashSymbol(obj.ns, obj.name) ^ KeywordHashMask,
			}
		default:
			return NIL
		}
	}
	var ns *string = nil
	if !args[0].Equals(NIL) {
		ns = STRINGS.Intern(EnsureString(args, 0).S)
	}
	name := STRINGS.Intern(EnsureString(args, 1).S)
	return Keyword{
		ns:   ns,
		name: name,
		hash: hashSymbol(ns, name) ^ KeywordHashMask,
	}
}

var procGensym = func(args []Object) Object {
	return genSym(EnsureString(args, 0).S, "")
}

var procApply = func(args []Object) Object {
	// TODO:
	// Stacktrace is broken. Need to somehow know
	// the name of the function passed ...
	f := EnsureCallable(args, 0)
	return f.Call(ToSlice(EnsureSeqable(args, 1).Seq()))
}

var procLazySeq = func(args []Object) Object {
	return &LazySeq{
		fn: args[0].(*Fn),
	}
}

var procDelay = func(args []Object) Object {
	return &Delay{
		fn: args[0].(*Fn),
	}
}

var procForce = func(args []Object) Object {
	switch d := args[0].(type) {
	case *Delay:
		return d.Force()
	default:
		return d
	}
}

var procIdentical = func(args []Object) Object {
	return Boolean{B: args[0] == args[1]}
}

var procCompare = func(args []Object) Object {
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

var procInt = func(args []Object) Object {
	switch obj := args[0].(type) {
	case Char:
		return Int{I: int(obj.Ch)}
	case Number:
		return obj.Int()
	default:
		panic(RT.NewError(fmt.Sprintf("Cannot cast %s (type: %s) to Int", obj.ToString(true), obj.GetType().ToString(false))))
	}
}

var procNumber = func(args []Object) Object {
	return AssertNumber(args[0], fmt.Sprintf("Cannot cast %s (type: %s) to Number", args[0].ToString(true), args[0].GetType().ToString(false)))
}

var procDouble = func(args []Object) Object {
	n := AssertNumber(args[0], fmt.Sprintf("Cannot cast %s (type: %s) to Double", args[0].ToString(true), args[0].GetType().ToString(false)))
	return n.Double()
}

var procChar = func(args []Object) Object {
	switch c := args[0].(type) {
	case Char:
		return c
	case Number:
		i := c.Int().I
		if i < MIN_RUNE || i > MAX_RUNE {
			panic(RT.NewError(fmt.Sprintf("Value out of range for char: %d", i)))
		}
		return Char{Ch: rune(i)}
	default:
		panic(RT.NewError(fmt.Sprintf("Cannot cast %s (type: %s) to Char", c.ToString(true), c.GetType().ToString(false))))
	}
}

var procBoolean = func(args []Object) Object {
	return Boolean{B: ToBool(args[0])}
}

var procNumerator = func(args []Object) Object {
	bi := EnsureRatio(args, 0).r.Num()
	return &BigInt{b: *bi}
}

var procDenominator = func(args []Object) Object {
	bi := EnsureRatio(args, 0).r.Denom()
	return &BigInt{b: *bi}
}

var procBigInt = func(args []Object) Object {
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

var procBigFloat = func(args []Object) Object {
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

var procNth = func(args []Object) Object {
	n := EnsureNumber(args, 1).Int().I
	switch coll := args[0].(type) {
	case Indexed:
		if len(args) == 3 {
			return coll.TryNth(n, args[2])
		}
		return coll.Nth(n)
	case Nil:
		return NIL
	case Sequential:
		switch coll := args[0].(type) {
		case Seqable:
			if len(args) == 3 {
				return SeqTryNth(coll.Seq(), n, args[2])
			}
			return SeqNth(coll.Seq(), n)
		}
	}
	panic(RT.NewError("nth not supported on this type: " + args[0].GetType().ToString(false)))
}

var procLt = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Boolean{B: GetOps(a).Combine(GetOps(b)).Lt(a, b)}
}

var procLte = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Boolean{B: GetOps(a).Combine(GetOps(b)).Lte(a, b)}
}

var procGt = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Boolean{B: GetOps(a).Combine(GetOps(b)).Gt(a, b)}
}

var procGte = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Boolean{B: GetOps(a).Combine(GetOps(b)).Gte(a, b)}
}

var procEq = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return MakeBoolean(numbersEq(a, b))
}

var procMax = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Max(a, b)
}

var procMin = func(args []Object) Object {
	a := AssertNumber(args[0], "")
	b := AssertNumber(args[1], "")
	return Min(a, b)
}

var procIncEx = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	ops := GetOps(x).Combine(BIGINT_OPS)
	return ops.Add(x, Int{I: 1})
}

var procDecEx = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	ops := GetOps(x).Combine(BIGINT_OPS)
	return ops.Subtract(x, Int{I: 1})
}

var procInc = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	ops := GetOps(x).Combine(INT_OPS)
	return ops.Add(x, Int{I: 1})
}

var procDec = func(args []Object) Object {
	x := EnsureNumber(args, 0)
	ops := GetOps(x).Combine(INT_OPS)
	return ops.Subtract(x, Int{I: 1})
}

var procPeek = func(args []Object) Object {
	s := AssertStack(args[0], "")
	return s.Peek()
}

var procPop = func(args []Object) Object {
	s := AssertStack(args[0], "")
	return s.Pop().(Object)
}

var procContains = func(args []Object) Object {
	switch c := args[0].(type) {
	case Gettable:
		ok, _ := c.Get(args[1])
		if ok {
			return Boolean{B: true}
		}
		return Boolean{B: false}
	}
	panic(RT.NewError("contains? not supported on type " + args[0].GetType().ToString(false)))
}

var procGet = func(args []Object) Object {
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

var procDissoc = func(args []Object) Object {
	return EnsureMap(args, 0).Without(args[1])
}

var procDisj = func(args []Object) Object {
	return EnsureSet(args, 0).Disjoin(args[1])
}

var procFind = func(args []Object) Object {
	res := EnsureAssociative(args, 0).EntryAt(args[1])
	if res == nil {
		return NIL
	}
	return res
}

var procKeys = func(args []Object) Object {
	return EnsureMap(args, 0).Keys()
}

var procVals = func(args []Object) Object {
	return EnsureMap(args, 0).Vals()
}

var procRseq = func(args []Object) Object {
	return EnsureReversible(args, 0).Rseq()
}

var procName = func(args []Object) Object {
	return String{S: EnsureNamed(args, 0).Name()}
}

var procNamespace = func(args []Object) Object {
	ns := EnsureNamed(args, 0).Namespace()
	if ns == "" {
		return NIL
	}
	return String{S: ns}
}

var procFindVar = func(args []Object) Object {
	sym := EnsureSymbol(args, 0)
	if sym.ns == nil {
		panic(RT.NewError("find-var argument must be namespace-qualified symbol"))
	}
	if v, ok := GLOBAL_ENV.Resolve(sym); ok {
		return v
	}
	return NIL
}

var procSort = func(args []Object) Object {
	cmp := EnsureComparator(args, 0)
	coll := EnsureSeqable(args, 1)
	s := SortableSlice{
		s:   ToSlice(coll.Seq()),
		cmp: cmp,
	}
	sort.Sort(s)
	return &ArraySeq{arr: s.s}
}

var procEval = func(args []Object) Object {
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	expr := Parse(args[0], parseContext)
	return Eval(expr, nil)
}

var procType = func(args []Object) Object {
	return args[0].GetType()
}

var procPprint = func(args []Object) Object {
	obj := args[0]
	w := Assertio_Writer(GLOBAL_ENV.stdout.Value, "")
	pprintObject(obj, 0, w)
	fmt.Fprint(w, "\n")
	return NIL
}

func PrintObject(obj Object, w io.Writer) {
	printReadably := ToBool(GLOBAL_ENV.printReadably.Value)
	switch obj := obj.(type) {
	case Printer:
		obj.Print(w, printReadably)
	default:
		fmt.Fprint(w, obj.ToString(printReadably))
	}
}

var procPr = func(args []Object) Object {
	n := len(args)
	if n > 0 {
		f := Assertio_Writer(GLOBAL_ENV.stdout.Value, "")
		for _, arg := range args[:n-1] {
			PrintObject(arg, f)
			fmt.Fprint(f, " ")
		}
		PrintObject(args[n-1], f)
	}
	return NIL
}

var procNewline = func(args []Object) Object {
	f := Assertio_Writer(GLOBAL_ENV.stdout.Value, "")
	fmt.Fprintln(f)
	return NIL
}

var procFlush = func(args []Object) Object {
	switch f := args[0].(type) {
	case *File:
		f.Sync()
	}
	return NIL
}

func readFromReader(reader io.RuneReader) Object {
	r := NewReader(reader, "<>")
	obj, err := TryRead(r)
	PanicOnErr(err)
	return obj
}

var procRead = func(args []Object) Object {
	f := Ensureio_RuneReader(args, 0)
	return readFromReader(f)
}

var procReadString = func(args []Object) Object {
	CheckArity(args, 1, 1)
	return readFromReader(strings.NewReader(EnsureString(args, 0).S))
}

func readLine(r StringReader) (s string, e error) {
	s, e = r.ReadString('\n')
	if e == nil {
		l := len(s)
		if s[l-1] == '\n' {
			l -= 1
			if l > 0 && s[l-1] == '\r' {
				l -= 1
			}
		}
		s = s[0:l]
	} else if s != "" && e == io.EOF {
		e = nil
	}
	return
}

var procReadLine = func(args []Object) Object {
	CheckArity(args, 0, 0)
	f := AssertStringReader(GLOBAL_ENV.stdin.Value, "")
	line, err := readLine(f)
	if err != nil {
		return NIL
	}
	return String{S: line}
}

var procReaderReadLine = func(args []Object) Object {
	CheckArity(args, 1, 1)
	rdr := EnsureStringReader(args, 0)
	line, err := readLine(rdr)
	if err != nil {
		return NIL
	}
	return String{S: line}
}

var procNanoTime = func(args []Object) Object {
	return &BigInt{b: *big.NewInt(time.Now().UnixNano())}
}

var procMacroexpand1 = func(args []Object) Object {
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

var procLoadString = func(args []Object) Object {
	s := EnsureString(args, 0)
	obj, err := loadReader(NewReader(strings.NewReader(s.S), "<string>"))
	if err != nil {
		panic(err)
	}
	return obj
}

var procFindNamespace = func(args []Object) Object {
	ns := GLOBAL_ENV.FindNamespace(EnsureSymbol(args, 0))
	if ns == nil {
		return NIL
	}
	return ns
}

var procCreateNamespace = func(args []Object) Object {
	sym := EnsureSymbol(args, 0)
	res := GLOBAL_ENV.EnsureNamespace(sym)
	// In linter mode the latest create-ns call overrides position info.
	// This is for the cases when (ns ...) is called in .jokerd/linter.clj file and alike.
	// Also, isUsed needs to be reset in this case.
	if LINTER_MODE {
		res.Name = res.Name.WithInfo(sym.GetInfo()).(Symbol)
		res.isUsed = false
	}
	return res
}

var procInjectNamespace = func(args []Object) Object {
	sym := EnsureSymbol(args, 0)
	ns := GLOBAL_ENV.EnsureNamespace(sym)
	ns.isUsed = true
	ns.isGloballyUsed = true
	return ns
}

var procRemoveNamespace = func(args []Object) Object {
	ns := GLOBAL_ENV.RemoveNamespace(EnsureSymbol(args, 0))
	if ns == nil {
		return NIL
	}
	return ns
}

var procAllNamespaces = func(args []Object) Object {
	s := make([]Object, 0, len(GLOBAL_ENV.Namespaces))
	for _, ns := range GLOBAL_ENV.Namespaces {
		s = append(s, ns)
	}
	return &ArraySeq{arr: s}
}

var procNamespaceName = func(args []Object) Object {
	return EnsureNamespace(args, 0).Name
}

var procNamespaceMap = func(args []Object) Object {
	r := &ArrayMap{}
	for k, v := range EnsureNamespace(args, 0).mappings {
		r.Add(MakeSymbol(*k), v)
	}
	return r
}

var procNamespaceUnmap = func(args []Object) Object {
	ns := EnsureNamespace(args, 0)
	sym := EnsureSymbol(args, 1)
	if sym.ns != nil {
		panic(RT.NewError("Can't unintern namespace-qualified symbol"))
	}
	delete(ns.mappings, sym.name)
	return NIL
}

var procVarNamespace = func(args []Object) Object {
	v := EnsureVar(args, 0)
	return v.ns
}

var procRefer = func(args []Object) Object {
	ns := EnsureNamespace(args, 0)
	sym := EnsureSymbol(args, 1)
	v := EnsureVar(args, 2)
	return ns.Refer(sym, v)
}

var procAlias = func(args []Object) Object {
	EnsureNamespace(args, 0).AddAlias(EnsureSymbol(args, 1), EnsureNamespace(args, 2))
	return NIL
}

var procNamespaceAliases = func(args []Object) Object {
	r := &ArrayMap{}
	for k, v := range EnsureNamespace(args, 0).aliases {
		r.Add(MakeSymbol(*k), v)
	}
	return r
}

var procNamespaceUnalias = func(args []Object) Object {
	ns := EnsureNamespace(args, 0)
	sym := EnsureSymbol(args, 1)
	if sym.ns != nil {
		panic(RT.NewError("Alias can't be namespace-qualified"))
	}
	delete(ns.aliases, sym.name)
	return NIL
}

var procVarGet = func(args []Object) Object {
	return EnsureVar(args, 0).Resolve()
}

var procVarSet = func(args []Object) Object {
	EnsureVar(args, 0).Value = args[1]
	return args[1]
}

var procNsResolve = func(args []Object) Object {
	ns := EnsureNamespace(args, 0)
	sym := EnsureSymbol(args, 1)
	if sym.ns == nil && TYPES[sym.name] != nil {
		return TYPES[sym.name]
	}
	if vr, ok := GLOBAL_ENV.ResolveIn(ns, sym); ok {
		return vr
	}
	return NIL
}

var procArrayMap = func(args []Object) Object {
	if len(args)%2 == 1 {
		panic(RT.NewError("No value supplied for key " + args[len(args)-1].ToString(false)))
	}
	res := EmptyArrayMap()
	for i := 0; i < len(args); i += 2 {
		res.Set(args[i], args[i+1])
	}
	return res
}

const bufferHashMask uint32 = 0x5ed19e84

var procBuffer = func(args []Object) Object {
	if len(args) > 0 {
		s := EnsureString(args, 0)
		return MakeBuffer(bytes.NewBufferString(s.S))
	}
	return MakeBuffer(&bytes.Buffer{})
}

var procBufferedReader = func(args []Object) Object {
	switch rdr := args[0].(type) {
	case io.Reader:
		return MakeBufferedReader(rdr)
	default:
		panic(RT.NewArgTypeError(0, args[0], "IOReader"))
	}
}

var procSlurp = func(args []Object) Object {
	switch f := args[0].(type) {
	case String:
		b, err := ioutil.ReadFile(f.S)
		PanicOnErr(err)
		return String{S: string(b)}
	case io.Reader:
		b, err := ioutil.ReadAll(f)
		PanicOnErr(err)
		return String{S: string(b)}
	default:
		panic(RT.NewArgTypeError(0, args[0], "String or IOReader"))
	}
}

var procSpit = func(args []Object) Object {
	f := args[0]
	content := args[1]
	opts := EnsureMap(args, 2)
	appendFile := false
	if ok, append := opts.Get(MakeKeyword("append")); ok {
		appendFile = ToBool(append)
	}
	flags := os.O_CREATE | os.O_WRONLY
	if appendFile {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	switch f := f.(type) {
	case String:
		file, err := os.OpenFile(f.S, flags, 0644)
		PanicOnErr(err)
		defer file.Close()
		_, err = file.WriteString(str(content))
		PanicOnErr(err)
	case io.Writer:
		_, err := io.WriteString(f, str(content))
		PanicOnErr(err)
	default:
		panic(RT.NewArgTypeError(0, args[0], "String or IOWriter"))
	}
	return NIL
}

var procShuffle = func(args []Object) Object {
	s := ToSlice(EnsureSeqable(args, 0).Seq())
	for i := range s {
		j := rand.Intn(i + 1)
		s[i], s[j] = s[j], s[i]
	}
	return NewVectorFrom(s...)
}

var procIsRealized = func(args []Object) Object {
	return Boolean{B: EnsurePending(args, 0).IsRealized()}
}

var procDeriveInfo = func(args []Object) Object {
	dest := args[0]
	src := args[1]
	return dest.WithInfo(src.GetInfo())
}

var procJokerVersion = func(args []Object) Object {
	return String{S: VERSION[1:]}
}

var procHash = func(args []Object) Object {
	return Int{I: int(args[0].Hash())}
}

func loadFile(filename string) Object {
	var reader *Reader
	f, err := os.Open(filename)
	PanicOnErr(err)
	reader = NewReader(bufio.NewReader(f), filename)
	ProcessReaderFromEval(reader, filename)
	return NIL
}

var procLoadFile = func(args []Object) Object {
	filename := EnsureString(args, 0)
	return loadFile(filename.S)
}

var procLoadLibFromPath = func(args []Object) Object {
	libname := EnsureSymbol(args, 0).Name()
	pathname := EnsureString(args, 1).S
	cp := GLOBAL_ENV.classPath.Value
	cpvec := AssertVector(cp, "*classpath* must be a Vector, not a "+cp.GetType().ToString(false))
	count := cpvec.Count()
	var f *os.File
	var err error
	var canonicalErr error
	var filename string
	for i := 0; i < count; i++ {
		elem := cpvec.at(i)
		cpelem := AssertString(elem, "*classpath* must contain only Strings, not a "+elem.GetType().ToString(false)+" (at element "+strconv.Itoa(i)+")")
		s := cpelem.S
		if s == "" {
			filename = pathname
		} else {
			filename = filepath.Join(s, filepath.Join(strings.Split(libname, ".")...)) + ".joke" // could cache inner join....
		}
		f, err = os.Open(filename)
		if err == nil {
			canonicalErr = nil
			break
		}
		if s == "" {
			canonicalErr = err
		}
	}
	PanicOnErr(canonicalErr)
	PanicOnErr(err)
	reader := NewReader(bufio.NewReader(f), filename)
	ProcessReaderFromEval(reader, filename)
	return NIL
}

var procReduceKv = func(args []Object) Object {
	f := EnsureCallable(args, 0)
	init := args[1]
	coll := EnsureKVReduce(args, 2)
	return coll.kvreduce(f, init)
}

var procIndexOf = func(args []Object) Object {
	s := EnsureString(args, 0)
	ch := EnsureChar(args, 1)
	for i, r := range s.S {
		if r == ch.Ch {
			return Int{I: i}
		}
	}
	return Int{I: -1}
}

func libExternalPath(sym Symbol) (path string, ok bool) {
	nsSourcesVar, _ := GLOBAL_ENV.Resolve(MakeSymbol("joker.core/*ns-sources*"))
	nsSources := ToSlice(nsSourcesVar.Value.(*Vector).Seq())

	var sourceKey string
	var sourceMap Map
	for _, source := range nsSources {
		sourceKey = source.(*Vector).Nth(0).ToString(false)
		match, _ := regexp.MatchString(sourceKey, sym.Name())
		if match {
			sourceMap = source.(*Vector).Nth(1).(Map)
			break
		}
	}
	if sourceMap != nil {
		ok, url := sourceMap.Get(MakeKeyword("url"))
		if !ok {
			panic(RT.NewError("Key :url not found in ns-sources for: " + sourceKey))
		} else {
			return externalSourceToPath(sym.Name(), url.ToString(false)), true
		}
	}
	return
}

var procLibPath = func(args []Object) Object {
	sym := EnsureSymbol(args, 0)
	var path string

	path, ok := libExternalPath(sym)

	if !ok {
		var file string
		if GLOBAL_ENV.file.Value == nil {
			var err error
			file, err = filepath.Abs("user")
			PanicOnErr(err)
		} else {
			file = AssertString(GLOBAL_ENV.file.Value, "").S
			if linkDest, err := os.Readlink(file); err == nil {
				file = linkDest
			}
		}
		ns := GLOBAL_ENV.CurrentNamespace().Name

		parts := strings.Split(ns.Name(), ".")
		for _ = range parts {
			file, _ = filepath.Split(file)
			if len(file) == 0 {
				break
			}
			file = file[:len(file)-1]
		}
		path = filepath.Join(append([]string{file}, strings.Split(sym.Name(), ".")...)...) + ".joke"
	}
	return String{S: path}
}

var procInternFakeVar = func(args []Object) Object {
	nsSym := EnsureSymbol(args, 0)
	sym := EnsureSymbol(args, 1)
	isMacro := ToBool(args[2])
	res := InternFakeSymbol(GLOBAL_ENV.FindNamespace(nsSym), sym)
	res.isMacro = isMacro
	return res
}

var procParse = func(args []Object) Object {
	lm, _ := GLOBAL_ENV.Resolve(MakeSymbol("joker.core/*linter-mode*"))
	lm.Value = Boolean{B: true}
	LINTER_MODE = true
	defer func() {
		LINTER_MODE = false
		lm.Value = Boolean{B: false}
	}()
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	res := Parse(args[0], parseContext)
	return res.Dump(false)
}

var procTypes = func(args []Object) Object {
	CheckArity(args, 0, 0)
	res := EmptyArrayMap()
	for k, v := range TYPES {
		res.Add(String{S: *k}, v)
	}
	return res
}

var procCreateChan = func(args []Object) Object {
	CheckArity(args, 1, 1)
	n := EnsureInt(args, 0)
	ch := make(chan FutureResult, n.I)
	return MakeChannel(ch)
}

var procCloseChan = func(args []Object) Object {
	CheckArity(args, 1, 1)
	EnsureChannel(args, 0).Close()
	return NIL
}

var procSend = func(args []Object) (obj Object) {
	CheckArity(args, 2, 2)
	ch := EnsureChannel(args, 0)
	v := args[1]
	if v.Equals(NIL) {
		panic(RT.NewError("Can't put nil on channel"))
	}
	if ch.isClosed {
		return MakeBoolean(false)
	}
	obj = MakeBoolean(true)
	defer func() {
		if r := recover(); r != nil {
			RT.GIL.Lock()
			obj = MakeBoolean(false)
		}
	}()
	RT.GIL.Unlock()
	ch.ch <- MakeFutureResult(v, nil)
	RT.GIL.Lock()
	return
}

var procReceive = func(args []Object) Object {
	CheckArity(args, 1, 1)
	ch := EnsureChannel(args, 0)
	RT.GIL.Unlock()
	res, ok := <-ch.ch
	RT.GIL.Lock()
	if !ok {
		return NIL
	}
	if res.err != nil {
		panic(res.err)
	}
	return res.value
}

var procGo = func(args []Object) Object {
	CheckArity(args, 1, 1)
	f := EnsureCallable(args, 0)
	ch := MakeChannel(make(chan FutureResult, 1))
	go func() {

		defer func() {
			if r := recover(); r != nil {
				switch r := r.(type) {
				case Error:
					ch.ch <- MakeFutureResult(NIL, r)
					ch.Close()
				default:
					RT.GIL.Unlock()
					panic(r)
				}
			}
			RT.GIL.Unlock()
		}()

		RT.GIL.Lock()
		res := f.Call([]Object{})
		ch.ch <- MakeFutureResult(res, nil)
		ch.Close()
	}()
	return ch
}

var procVerbosityLevel = func(args []Object) Object {
	CheckArity(args, 0, 0)
	return MakeInt(VerbosityLevel)
}

var procExit = func(args []Object) Object {
	ExitJoker(EnsureInt(args, 0).I)
	return NIL
}

func PackReader(reader *Reader, filename string) ([]byte, error) {
	var p []byte
	packEnv := NewPackEnv()
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	if filename != "" {
		currentFilename := parseContext.GlobalEnv.file.Value
		defer func() {
			parseContext.GlobalEnv.SetFilename(currentFilename)
		}()
		s, err := filepath.Abs(filename)
		PanicOnErr(err)
		parseContext.GlobalEnv.SetFilename(MakeString(s))
	}
	for {
		obj, err := TryRead(reader)
		if err == io.EOF {
			var hp []byte
			hp = packEnv.Pack(hp)
			return append(hp, p...), nil
		}
		if err != nil {
			fmt.Fprintln(Stderr, err)
			return nil, err
		}
		expr, err := TryParse(obj, parseContext)
		if err != nil {
			fmt.Fprintln(Stderr, err)
			return nil, err
		}
		p = expr.Pack(p, packEnv)
		_, err = TryEval(expr)
		if err != nil {
			fmt.Fprintln(Stderr, err)
			return nil, err
		}
	}
}

var procIncProblemCount = func(args []Object) Object {
	PROBLEM_COUNT++
	return NIL
}

func ProcessReader(reader *Reader, filename string, phase Phase) error {
	if phase == FORMAT {
		FORMAT_MODE = true
		HASHMAP_THRESHOLD = 100000
	}
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	if filename != "" {
		currentFilename := parseContext.GlobalEnv.file.Value
		defer func() {
			parseContext.GlobalEnv.SetFilename(currentFilename)
		}()
		s, err := filepath.Abs(filename)
		PanicOnErr(err)
		parseContext.GlobalEnv.SetFilename(MakeString(s))
	}
	var prevObj Object
	for {
		obj, err := TryRead(reader)
		if err == io.EOF {
			if FORMAT_MODE && prevObj != nil {
				fmt.Fprint(Stdout, "\n")
			}
			return nil
		}
		if err != nil {
			fmt.Fprintln(Stderr, err)
			return err
		}
		if phase == READ {
			continue
		}
		if phase == FORMAT {
			if prevObj != nil {
				cnt := newLineCount(prevObj, obj)
				for i := 0; i < cnt; i++ {
					fmt.Fprint(Stdout, "\n")
				}
				if cnt == 0 {
					fmt.Fprint(Stdout, " ")
				}
			}
			formatObject(obj, 0, Stdout)
			prevObj = obj
			continue
		}
		expr, err := TryParse(obj, parseContext)
		if err != nil {
			fmt.Fprintln(Stderr, err)
			return err
		}
		if phase == PARSE {
			continue
		}
		obj, err = TryEval(expr)
		if err != nil {
			fmt.Fprintln(Stderr, err)
			return err
		}
		if phase == EVAL {
			continue
		}
		if _, ok := obj.(Nil); !ok {
			fmt.Fprintln(Stdout, obj.ToString(true))
		}
	}
}

func ProcessReaderFromEval(reader *Reader, filename string) {
	parseContext := &ParseContext{GlobalEnv: GLOBAL_ENV}
	if filename != "" {
		currentFilename := parseContext.GlobalEnv.file.Value
		defer func() {
			parseContext.GlobalEnv.SetFilename(currentFilename)
		}()
		s, err := filepath.Abs(filename)
		PanicOnErr(err)
		parseContext.GlobalEnv.SetFilename(MakeString(s))
	}
	for {
		obj, err := TryRead(reader)
		if err == io.EOF {
			return
		}
		PanicOnErr(err)
		expr, err := TryParse(obj, parseContext)
		PanicOnErr(err)
		obj, err = TryEval(expr)
		PanicOnErr(err)
	}
}

func processData(data []byte) {
	ns := GLOBAL_ENV.CurrentNamespace()
	GLOBAL_ENV.SetCurrentNamespace(GLOBAL_ENV.CoreNamespace)
	defer func() { GLOBAL_ENV.SetCurrentNamespace(ns) }()
	header, p := UnpackHeader(data, GLOBAL_ENV)
	for len(p) > 0 {
		var expr Expr
		expr, p = UnpackExpr(p, header)
		_, err := TryEval(expr)
		PanicOnErr(err)
	}
	if VerbosityLevel > 0 {
		fmt.Fprintf(Stderr, "processData: Evaluated code for %s\n", GLOBAL_ENV.CurrentNamespace().ToString(false))
	}
}

func setCoreNamespaces() {
	ns := GLOBAL_ENV.CoreNamespace
	ns.MaybeLazy("joker.core")

	vr := ns.Resolve("*core-namespaces*")
	set := vr.Value.(*MapSet)
	for _, ns := range coreNamespaces {
		set = set.Conj(MakeSymbol(ns)).(*MapSet)
	}
	vr.Value = set

	// Add 'joker.core to *loaded-libs*, now that it's loaded.
	vr = ns.Resolve("*loaded-libs*")
	set = vr.Value.(*MapSet).Conj(ns.Name).(*MapSet)
	vr.Value = set
}

var procIsNamespaceInitialized = func(args []Object) Object {
	sym := EnsureSymbol(args, 0)
	if sym.ns != nil {
		panic(RT.NewError("Can't ask for namespace info on namespace-qualified symbol"))
	}
	// First look for registered (e.g. std) libs
	ns, found := GLOBAL_ENV.Namespaces[sym.name]
	return MakeBoolean(found && ns.Lazy == nil)
}

func findConfigFile(filename string, workingDir string, findDir bool) string {
	var err error
	configName := ".joker"
	if findDir {
		configName = ".jokerd"
	}
	if filename != "" {
		filename, err = filepath.Abs(filename)
		if err != nil {
			fmt.Fprintln(Stderr, "Error reading config file "+filename+": ", err)
			return ""
		}
	}

	if workingDir != "" {
		workingDir, err := filepath.Abs(workingDir)
		if err != nil {
			fmt.Fprintln(Stderr, "Error resolving working directory"+workingDir+": ", err)
			return ""
		}
		filename = filepath.Join(workingDir, configName)
	}
	for {
		oldFilename := filename
		filename = filepath.Dir(filename)
		if filename == oldFilename {
			home := HomeDir()
			if home == "" {
				return ""
			}
			p := filepath.Join(home, configName)
			if info, err := os.Stat(p); err == nil {
				if !findDir || info.IsDir() {
					return p
				}
			}
			return ""
		}
		p := filepath.Join(filename, configName)
		if info, err := os.Stat(p); err == nil {
			if !findDir || info.IsDir() {
				return p
			}
		}
	}
}

func printConfigError(filename, msg string) {
	fmt.Fprintln(Stderr, "Error reading config file "+filename+": ", msg)
}

func knownMacrosToMap(km Object) (Map, error) {
	s := km.(Seqable).Seq()
	res := EmptyArrayMap()
	for !s.IsEmpty() {
		obj := s.First()
		switch obj := obj.(type) {
		case Symbol:
			res.Add(obj, NIL)
		case *Vector:
			if obj.Count() != 2 {
				return nil, errors.New(":known-macros item must be a symbol or a vector with two elements")
			}
			res.Add(obj.at(0), obj.at(1))
		default:
			return nil, errors.New(":known-macros item must be a symbol or a vector, got " + obj.GetType().ToString(false))
		}
		s = s.Rest()
	}
	return res, nil
}

func ReadConfig(filename string, workingDir string) {
	LINTER_CONFIG = GLOBAL_ENV.CoreNamespace.Intern(MakeSymbol("*linter-config*"))
	LINTER_CONFIG.Value = EmptyArrayMap()
	configFileName := findConfigFile(filename, workingDir, false)
	if configFileName == "" {
		return
	}
	f, err := os.Open(configFileName)
	if err != nil {
		printConfigError(configFileName, err.Error())
		return
	}
	r := NewReader(bufio.NewReader(f), configFileName)
	config, err := TryRead(r)
	if err != nil {
		printConfigError(configFileName, err.Error())
		return
	}
	configMap, ok := config.(Map)
	if !ok {
		printConfigError(configFileName, "config root object must be a map, got "+config.GetType().ToString(false))
		return
	}
	ok, ignoredUnusedNamespaces := configMap.Get(MakeKeyword("ignored-unused-namespaces"))
	if ok {
		seq, ok1 := ignoredUnusedNamespaces.(Seqable)
		if ok1 {
			WARNINGS.ignoredUnusedNamespaces = NewSetFromSeq(seq.Seq())
		} else {
			printConfigError(configFileName, ":ignored-unused-namespaces value must be a vector, got "+ignoredUnusedNamespaces.GetType().ToString(false))
			return
		}
	}
	ok, ignoredFileRegexes := configMap.Get(MakeKeyword("ignored-file-regexes"))
	if ok {
		seq, ok1 := ignoredFileRegexes.(Seqable)
		if ok1 {
			s := seq.Seq()
			for !s.IsEmpty() {
				regex, ok2 := s.First().(*Regex)
				if !ok2 {
					printConfigError(configFileName, ":ignored-file-regexes elements must be regexes, got "+s.First().GetType().ToString(false))
					return
				}
				WARNINGS.IgnoredFileRegexes = append(WARNINGS.IgnoredFileRegexes, regex.R)
				s = s.Rest()
			}
		} else {
			printConfigError(configFileName, ":ignored-file-regexes value must be a vector, got "+ignoredFileRegexes.GetType().ToString(false))
			return
		}
	}
	ok, entryPoints := configMap.Get(MakeKeyword("entry-points"))
	if ok {
		seq, ok1 := entryPoints.(Seqable)
		if ok1 {
			WARNINGS.entryPoints = NewSetFromSeq(seq.Seq())
		} else {
			printConfigError(configFileName, ":entry-points value must be a vector, got "+entryPoints.GetType().ToString(false))
			return
		}
	}
	ok, knownNamespaces := configMap.Get(MakeKeyword("known-namespaces"))
	if ok {
		if _, ok1 := knownNamespaces.(Seqable); !ok1 {
			printConfigError(configFileName, ":known-namespaces value must be a vector, got "+knownNamespaces.GetType().ToString(false))
			return
		}
	}
	ok, knownTags := configMap.Get(MakeKeyword("known-tags"))
	if ok {
		if _, ok1 := knownTags.(Seqable); !ok1 {
			printConfigError(configFileName, ":known-tags value must be a vector, got "+knownTags.GetType().ToString(false))
			return
		}
	}
	ok, knownMacros := configMap.Get(KEYWORDS.knownMacros)
	if ok {
		_, ok1 := knownMacros.(Seqable)
		if !ok1 {
			printConfigError(configFileName, ":known-macros value must be a vector, got "+knownMacros.GetType().ToString(false))
			return
		}
		m, err := knownMacrosToMap(knownMacros)
		if err != nil {
			printConfigError(configFileName, err.Error())
			return
		}
		configMap = configMap.Assoc(KEYWORDS.knownMacros, m).(Map)
	}
	ok, rules := configMap.Get(KEYWORDS.rules)
	if ok {
		m, ok := rules.(Map)
		if !ok {
			printConfigError(configFileName, ":rules value must be a map, got "+rules.GetType().ToString(false))
			return
		}
		if ok, v := m.Get(KEYWORDS.ifWithoutElse); ok {
			WARNINGS.ifWithoutElse = ToBool(v)
		}
		if ok, v := m.Get(KEYWORDS.unusedFnParameters); ok {
			WARNINGS.unusedFnParameters = ToBool(v)
		}
		if ok, v := m.Get(KEYWORDS.fnWithEmptyBody); ok {
			WARNINGS.fnWithEmptyBody = ToBool(v)
		}
	}
	LINTER_CONFIG.Value = configMap
}

func removeJokerNamespaces() {
	for k, ns := range GLOBAL_ENV.Namespaces {
		if ns != GLOBAL_ENV.CoreNamespace && strings.HasPrefix(*k, "joker.") {
			delete(GLOBAL_ENV.Namespaces, k)
		}
	}
}

func markJokerNamespacesAsUsed() {
	for k, ns := range GLOBAL_ENV.Namespaces {
		if ns != GLOBAL_ENV.CoreNamespace && strings.HasPrefix(*k, "joker.") {
			ns.isUsed = true
			ns.isGloballyUsed = true
		}
	}
}

func NewReaderFromFile(filename string) (*Reader, error) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(Stderr, "Error: ", err)
		return nil, err
	}
	return NewReader(bufio.NewReader(f), filename), nil
}

func ProcessLinterFile(configDir string, filename string) {
	linterFileName := filepath.Join(configDir, filename)
	if _, err := os.Stat(linterFileName); err == nil {
		if reader, err := NewReaderFromFile(linterFileName); err == nil {
			ProcessReader(reader, linterFileName, EVAL)
		}
	}
}

func ProcessLinterFiles(dialect Dialect, filename string, workingDir string) {
	if dialect == EDN {
		return
	}
	configDir := findConfigFile(filename, workingDir, true)
	if configDir == "" {
		return
	}
	if dialect == JOKER {
		ProcessLinterFile(configDir, "linter.joke")
		return
	}
	ProcessLinterFile(configDir, "linter.cljc")
	switch dialect {
	case CLJS:
		ProcessLinterFile(configDir, "linter.cljs")
	case CLJ:
		ProcessLinterFile(configDir, "linter.clj")
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
