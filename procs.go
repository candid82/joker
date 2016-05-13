package main

import (
	"reflect"
)

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

func ensureSeq(obj Object, msg string) Seq {
	switch s := obj.(type) {
	case Seq:
		return s
	case Sequenceable:
		return s.Seq()
	default:
		panic(RT.newError(msg))
	}
}

var procCons Proc = func(args []Object) Object {
	checkArity(args, 2, 2)
	s := ensureSeq(args[1], "cons's second argument must be sequenceable")
	return s.Cons(args[0])
}

var procFirst Proc = func(args []Object) Object {
	checkArity(args, 1, 1)
	s := ensureSeq(args[0], "first's argument must be sequenceable")
	return s.First()
}

var procNext Proc = func(args []Object) Object {
	checkArity(args, 1, 1)
	s := ensureSeq(args[0], "next's argument must be sequenceable")
	res := s.Rest()
	if res.IsEmpty() {
		return NIL
	}
	return res
}

var procRest Proc = func(args []Object) Object {
	checkArity(args, 1, 1)
	s := ensureSeq(args[0], "rest's argument must be sequenceable")
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
	return ensureSeq(args[0], "Argument to seq must be sequenceable")
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
		s := ensureSeq(obj, "count not supported on this type: "+obj.GetType().ToString(false))
		c := 0
		for !s.IsEmpty() {
			c++
			s = s.Rest()
			switch obj := s.(type) {
			case Counted:
				return Int{i: c + obj.Count()}
			}
		}
		return Int{i: c}
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

var coreNamespace = GLOBAL_ENV.namespaces[MakeSymbol("gclojure.core").name]

func intern(name string, proc Proc) {
	coreNamespace.intern(MakeSymbol(name)).value = proc
}

func init() {
	intern("list*", procList)
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
	processFile("core.clj", EVAL)
	GLOBAL_ENV.currentNamespace = currentNamespace
}
