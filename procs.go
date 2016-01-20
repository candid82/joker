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

func init() {
	GLOBAL_ENV.currentNamespace.intern(MakeSymbol("meta")).value = procMeta
}
