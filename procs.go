package main

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

func intern(name string, proc Proc) {
	GLOBAL_ENV.currentNamespace.intern(MakeSymbol(name)).value = proc
}

func init() {
	intern("meta", procMeta)
	intern("zero?", procIsZero)
}
