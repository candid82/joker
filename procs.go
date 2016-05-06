package main

func ensureNumber(obj Object) Number {
	switch n := obj.(type) {
	case Number:
		return n
	default:
		panic(RT.newError(obj.ToString(false) + " is not a Number"))
	}
}

func ensureString(obj Object) String {
	switch n := obj.(type) {
	case String:
		return n
	default:
		panic(RT.newError(obj.ToString(false) + " is not a String"))
	}
}

func ensureMap(obj Object) *ArrayMap {
	switch n := obj.(type) {
	case *ArrayMap:
		return n
	default:
		panic(RT.newError(obj.ToString(false) + " is not a Map"))
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

var procIsZero Proc = func(args []Object) Object {
	// checkArity(args, 1, "zero?")
	n := ensureNumber(args[0])
	ops := GetOps(ensureNumber(args[0]))
	return Bool{b: ops.IsZero(n)}
}

var procAdd Proc = func(args []Object) Object {
	var res Number = Int{i: 0}
	for _, n := range args {
		ops := GetOps(res).Combine(GetOps(n))
		res = ops.Add(res, ensureNumber(n))
	}
	return res
}

var procMultiply Proc = func(args []Object) Object {
	var res Number = Int{i: 1}
	for _, n := range args {
		ops := GetOps(res).Combine(GetOps(n))
		res = ops.Multiply(res, ensureNumber(n))
	}
	return res
}

var procSubtract Proc = func(args []Object) Object {
	if len(args) == 0 {
		panicArity(0)
	}
	var res Number = Int{i: 0}
	numbers := args
	if len(args) > 1 {
		res = ensureNumber(args[0])
		numbers = args[1:]
	}
	for _, n := range numbers {
		ops := GetOps(res).Combine(GetOps(n))
		res = ops.Subtract(res, ensureNumber(n))
	}
	return res
}

var procDivide Proc = func(args []Object) Object {
	if len(args) == 0 {
		panicArity(0)
	}
	var res Number = Int{i: 1}
	numbers := args
	if len(args) > 1 {
		res = ensureNumber(args[0])
		numbers = args[1:]
	}
	for _, n := range numbers {
		ops := GetOps(res).Combine(GetOps(n))
		res = ops.Divide(res, ensureNumber(n))
	}
	return res
}

var procExInfo Proc = func(args []Object) Object {
	checkArity(args, 2, 2)
	return &ExInfo{
		msg:  ensureString(args[0]),
		data: ensureMap(args[1]),
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
	checkArity(args, 1, 1)
	switch vr := args[0].(type) {
	case *Var:
		vr.isMacro = true
		return vr
	default:
		panic(RT.newError("set-macro argument must be a Var"))
	}
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
		return Bool{b: args[1].GetType() == t}
	default:
		panic(RT.newError("First argument to instance? must be a type"))
	}
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

	intern("meta", procMeta)
	intern("zero?", procIsZero)
	intern("+", procAdd)
	intern("-", procSubtract)
	intern("*", procMultiply)
	intern("/", procDivide)
	intern("ex-info", procExInfo)
	intern("print", procPrint)
	intern("set-macro", procSetMacro)

	currentNamespace := GLOBAL_ENV.currentNamespace
	GLOBAL_ENV.currentNamespace = coreNamespace
	processFile("core.clj", EVAL)
	GLOBAL_ENV.currentNamespace = currentNamespace
}
