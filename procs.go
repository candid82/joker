package main

import (
	"bytes"
	"fmt"
	"reflect"
)

func assertCallable(obj Object, msg string) Callable {
	switch s := obj.(type) {
	case Callable:
		return s
	default:
		panic(RT.newError(msg))
	}
}

func assertChar(obj Object, msg string) Char {
	switch c := obj.(type) {
	case Char:
		return c
	default:
		panic(RT.newError(msg))
	}
}

func assertString(obj Object, msg string) String {
	switch s := obj.(type) {
	case String:
		return s
	default:
		panic(RT.newError(msg))
	}
}

func assertSymbol(obj Object, msg string) Symbol {
	switch s := obj.(type) {
	case Symbol:
		return s
	default:
		panic(RT.newError(msg))
	}
}

func assertVector(obj Object, msg string) *Vector {
	switch v := obj.(type) {
	case *Vector:
		return v
	default:
		panic(RT.newError(msg))
	}
}

func assertComparable(obj Object, msg string) Comparable {
	switch c := obj.(type) {
	case Comparable:
		return c
	default:
		if msg == "" {
			msg = fmt.Sprintf("Expected %s, got %s", "Comparable", obj.GetType().ToString(false))
		}
		panic(RT.newError(msg))
	}
}

func assertKeyword(obj Object, msg string) Keyword {
	switch k := obj.(type) {
	case Keyword:
		return k
	default:
		panic(RT.newError(msg))
	}
}

func assertBool(obj Object, msg string) Bool {
	switch b := obj.(type) {
	case Bool:
		return b
	default:
		panic(RT.newError(msg))
	}
}

func assertNumber(obj Object, msg string) Number {
	switch s := obj.(type) {
	case Number:
		return s
	default:
		panic(RT.newError(msg))
	}
}

func assertSeq(obj Object, msg string) Seq {
	switch s := obj.(type) {
	case Seqable:
		return s.Seq()
	default:
		if msg == "" {
			msg = fmt.Sprintf("Expected %s, got %s", "Seqable", obj.GetType().ToString(false))
		}
		panic(RT.newError(msg))
	}
}

func ensureCallable(args []Object, index int) Callable {
	switch c := args[index].(type) {
	case Callable:
		return c
	default:
		panic(RT.newArgTypeError(index, "Fn"))
	}
}

func ensureSeq(args []Object, index int) Seq {
	switch s := args[index].(type) {
	case Seqable:
		return s.Seq()
	default:
		panic(RT.newArgTypeError(index, "Seqable"))
	}
}

func ensureNumber(args []Object, index int) Number {
	switch obj := args[index].(type) {
	case Number:
		return obj
	default:
		panic(RT.newArgTypeError(index, "Number"))
	}
}

func ensureString(args []Object, index int) String {
	switch obj := args[index].(type) {
	case String:
		return obj
	default:
		panic(RT.newArgTypeError(index, "String"))
	}
}

func ensureType(args []Object, index int) *Type {
	switch obj := args[index].(type) {
	case *Type:
		return obj
	default:
		panic(RT.newArgTypeError(index, "Type"))
	}
}

func ensureMap(args []Object, index int) *ArrayMap {
	switch obj := args[index].(type) {
	case *ArrayMap:
		return obj
	default:
		panic(RT.newArgTypeError(index, "Map"))
	}
}

func ensureMeta(args []Object, index int) Meta {
	switch obj := args[index].(type) {
	case Meta:
		return obj
	default:
		panic(RT.newArgTypeError(index, "Meta"))
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
	checkArity(args, 2, 2)
	return ensureMeta(args, 0).WithMeta(ensureMap(args, 1))
}

var procIsZero Proc = func(args []Object) Object {
	// checkArity(args, 1, "zero?")
	n := ensureNumber(args, 0)
	ops := GetOps(ensureNumber(args, 0))
	return Bool{b: ops.IsZero(n)}
}

var procAdd Proc = func(args []Object) Object {
	var res Number = Int{i: 0}
	for i, n := range args {
		ops := GetOps(res).Combine(GetOps(n))
		res = ops.Add(res, ensureNumber(args, i))
	}
	return res
}

var procMultiply Proc = func(args []Object) Object {
	var res Number = Int{i: 1}
	for i, n := range args {
		ops := GetOps(res).Combine(GetOps(n))
		res = ops.Multiply(res, ensureNumber(args, i))
	}
	return res
}

var procSubtract Proc = func(args []Object) Object {
	if len(args) == 0 {
		panicArity(0)
	}
	var res Number = Int{i: 0}
	start := 0
	if len(args) > 1 {
		res = ensureNumber(args, 0)
		start = 1
	}
	for i := start; i < len(args); i++ {
		ops := GetOps(res).Combine(GetOps(args[i]))
		res = ops.Subtract(res, ensureNumber(args, i))
	}
	return res
}

var procDivide Proc = func(args []Object) Object {
	if len(args) == 0 {
		panicArity(0)
	}
	var res Number = Int{i: 1}
	start := 0
	if len(args) > 1 {
		res = ensureNumber(args, 0)
		start = 1
	}
	for i := start; i < len(args); i++ {
		ops := GetOps(res).Combine(GetOps(args[i]))
		res = ops.Divide(res, ensureNumber(args, i))
	}
	return res
}

var procExInfo Proc = func(args []Object) Object {
	checkArity(args, 2, 2)
	return &ExInfo{
		msg:  ensureString(args, 0),
		data: ensureMap(args, 1),
		rt:   RT.clone(),
	}
}

var procPrint Proc = func(args []Object) Object {
	n := len(args)
	if n > 0 {
		for _, arg := range args[:n-1] {
			print(arg.ToString(false))
			print(" ")
		}
		print(args[n-1].ToString(false))
	}
	return NIL
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
	checkArity(args, 2, 2)
	s := ensureSeq(args, 1)
	return s.Cons(args[0])
}

var procFirst Proc = func(args []Object) Object {
	checkArity(args, 1, 1)
	s := ensureSeq(args, 0)
	return s.First()
}

var procNext Proc = func(args []Object) Object {
	checkArity(args, 1, 1)
	s := ensureSeq(args, 0)
	res := s.Rest()
	if res.IsEmpty() {
		return NIL
	}
	return res
}

var procRest Proc = func(args []Object) Object {
	checkArity(args, 1, 1)
	s := ensureSeq(args, 0)
	return s.Rest()
}

var procConj Proc = func(args []Object) Object {
	switch c := args[0].(type) {
	case Nil:
		return NewListFrom(args[1])
	case Conjable:
		return c.Conj(args[1])
	case Seq:
		return c.Cons(args[1])
	default:
		panic(RT.newError("conj's first argument must be a collection"))
	}
}

var procSeq Proc = func(args []Object) Object {
	checkArity(args, 1, 1)
	s := ensureSeq(args, 0)
	if s.IsEmpty() {
		return NIL
	}
	return s
}

var procIsInstance Proc = func(args []Object) Object {
	checkArity(args, 2, 2)
	switch t := args[0].(type) {
	case *Type:
		if t.reflectType.Kind() == reflect.Interface {
			return Bool{b: args[1].GetType().reflectType.Implements(t.reflectType)}
		} else {
			return Bool{b: args[1].GetType().reflectType == t.reflectType}
		}
	default:
		panic(RT.newError("First argument to instance? must be a type"))
	}
}

var procAssoc Proc = func(args []Object) Object {
	if args[0].Equals(NIL) {
		return EmptyArrayMap().Assoc(args[1], args[2])
	}
	return ensureMap(args, 0).Assoc(args[1], args[2])
}

var procEquals Proc = func(args []Object) Object {
	return Bool{b: args[0].Equals(args[1])}
}

var procCount Proc = func(args []Object) Object {
	switch obj := args[0].(type) {
	case Counted:
		return Int{i: obj.Count()}
	default:
		s := assertSeq(obj, "count not supported on this type: "+obj.GetType().ToString(false))
		return Int{i: SeqCount(s)}
	}
}

var procSubvec Proc = func(args []Object) Object {
	// TODO: implement proper Subvector structure
	v := args[0].(*Vector)
	start := args[1].(Int).i
	end := args[2].(Int).i
	subv := make([]Object, 0, end-start)
	for i := start; i < end; i++ {
		subv = append(subv, v.at(i))
	}
	return NewVectorFrom(subv...)
}

var procCast Proc = func(args []Object) Object {
	t := ensureType(args, 0)
	if t.reflectType.Kind() == reflect.Interface &&
		args[1].GetType().reflectType.Implements(t.reflectType) ||
		args[1].GetType().reflectType == t.reflectType {
		return args[1]
	}
	panic(RT.newError("Cannot cast " + args[1].GetType().ToString(false) + " to " + t.ToString(false)))
}

var procVec Proc = func(args []Object) Object {
	return NewVectorFromSeq(ensureSeq(args, 0))
}

var procHashMap Proc = func(args []Object) Object {
	if len(args)%2 != 0 {
		panic(RT.newError("No value supplied for key " + args[len(args)-1].ToString(false)))
	}
	res := EmptyArrayMap()
	for i := 0; i < len(args); i += 2 {
		res.Set(args[i], args[i+1])
	}
	return res
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
	return String{s: buffer.String()}
}

var procSymbol Proc = func(args []Object) Object {
	if len(args) == 1 {
		return MakeSymbol(ensureString(args, 0).s)
	}
	return Symbol{
		ns:   STRINGS.Intern(ensureString(args, 0).s),
		name: STRINGS.Intern(ensureString(args, 1).s),
	}
}

var procKeyword Proc = func(args []Object) Object {
	if len(args) == 1 {
		switch obj := args[0].(type) {
		case String:
			return MakeKeyword(obj.s)
		case Symbol:
			return Keyword{
				ns:   obj.ns,
				name: obj.name,
			}
		default:
			return NIL
		}
	}
	return Keyword{
		ns:   STRINGS.Intern(ensureString(args, 0).s),
		name: STRINGS.Intern(ensureString(args, 1).s),
	}
}

var procGensym Proc = func(args []Object) Object {
	return genSym(ensureString(args, 0).s, "")
}

var procApply Proc = func(args []Object) Object {
	// TODO:
	// Stacktrace is broken. Need to somehow know
	// the name of the function passed ...
	f := ensureCallable(args, 0)
	return f.Call(ToSlice(ensureSeq(args, 1)))
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
	return Bool{b: args[0] == args[1]}
}

var procCompare Proc = func(args []Object) Object {
	k1, k2 := args[0], args[1]
	if k1.Equals(k2) {
		return Int{i: 0}
	}
	switch k2.(type) {
	case Nil:
		return Int{i: 1}
	}
	switch k1 := k1.(type) {
	case Nil:
		return Int{i: -1}
	case Comparable:
		return Int{i: k1.Compare(k2)}
	}
	panic(RT.newError(fmt.Sprintf("%s (type: %s) is not a Comparable", k1.ToString(true), k1.GetType().ToString(false))))
}

var coreNamespace = GLOBAL_ENV.namespaces[MakeSymbol("gclojure.core").name]

func intern(name string, proc Proc) {
	coreNamespace.intern(MakeSymbol(name)).value = proc
}

func init() {
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

	intern("zero?", procIsZero)
	intern("+", procAdd)
	intern("-", procSubtract)
	intern("*", procMultiply)
	intern("/", procDivide)
	intern("ex-info", procExInfo)
	intern("print", procPrint)
	intern("set-macro*", procSetMacro)

	currentNamespace := GLOBAL_ENV.currentNamespace
	GLOBAL_ENV.currentNamespace = coreNamespace
	data, err := Asset("data/core.clj")
	if err != nil {
		panic(RT.newError("Could not load core.clj"))
	}
	reader := bytes.NewReader(data)
	processReader(NewReader(reader), EVAL)
	GLOBAL_ENV.currentNamespace = currentNamespace
}
