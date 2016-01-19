package main

var procMeta Proc = func(args []Object) Object {
	switch obj := args[0].(type) {
	case Meta:
		return obj.GetMeta()
	default:
		return NIL
	}
}

func init() {
	GLOBAL_ENV.currentNamespace.intern(MakeSymbol("meta")).value = procMeta
}
