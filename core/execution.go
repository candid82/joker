package core

// Evaluate is the only production entry point for executing parsed code. Macros
// use it too. Compilation errors are surfaced, never hidden by AST fallback.
func Evaluate(expr Expr) Object {
	if DISABLE_VM {
		return Eval(expr, nil)
	}
	proto, err := CompileTopLevel(expr)
	PanicOnErr(err)
	return VMExecute(&Fn{proto: proto, isCompiled: true}, nil)
}

func TryEvaluate(expr Expr) (obj Object, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(Error); ok {
				err = e
			} else {
				panic(r)
			}
		}
	}()
	return Evaluate(expr), nil
}
