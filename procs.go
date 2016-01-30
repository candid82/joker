package main

func ensureNumber(obj Object) Number {
	switch n := obj.(type) {
	case Number:
		return n
	default:
		panic(&EvalError{msg: obj.ToString(false) + " is not a Number"})
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
	return Bool(args[0].(Number).IsZero())
}

var procPlus Proc = func(args []Object) Object {
	var res Number = Int(0)
	for _, n := range args {
		res = res.Plus(ensureNumber(n))
	}
	return res
}

func intern(name string, proc Proc) {
	GLOBAL_ENV.currentNamespace.intern(MakeSymbol(name)).value = proc
}

func init() {
	intern("meta", procMeta)
	intern("zero?", procIsZero)
	intern("+", procPlus)
}
