package main

import (
	"fmt"
)

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
	n := ensureNumber(args[0])
	ops := GetOps(ensureNumber(args[0]))
	return Bool(ops.IsZero(n))
}

var procAdd Proc = func(args []Object) Object {
	var res Number = Int(0)
	for _, n := range args {
		ops := GetOps(res).Combine(GetOps(n))
		res = ops.Add(res, ensureNumber(n))
	}
	return res
}

var procMultiply Proc = func(args []Object) Object {
	var res Number = Int(1)
	for _, n := range args {
		ops := GetOps(res).Combine(GetOps(n))
		res = ops.Multiply(res, ensureNumber(n))
	}
	return res
}

func panicArity(n int, name string) {
	panic(&EvalError{msg: fmt.Sprintf("Wrong number of args (%d) passed to %s", n, name)})
}

var procSubtract Proc = func(args []Object) Object {
	if len(args) == 0 {
		panicArity(0, "-")
	}
	var res Number = Int(0)
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
		panicArity(0, "/")
	}
	var res Number = Int(1)
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

func intern(name string, proc Proc) {
	GLOBAL_ENV.currentNamespace.intern(MakeSymbol(name)).value = proc
}

func init() {
	intern("meta", procMeta)
	intern("zero?", procIsZero)
	intern("+", procAdd)
	intern("-", procSubtract)
	intern("*", procMultiply)
	intern("/", procDivide)
}
