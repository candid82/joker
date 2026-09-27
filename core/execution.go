package core

// Execution owns one VM for a logical execution. Unlike RT.vm, this handle
// remains associated with its VM while other executions interleave. Call and
// Evaluate can enter it while its VM is paused in a native call.
type Execution struct {
	vm *VM
}

func NewExecution() *Execution {
	return &Execution{vm: vmPool.Get().(*VM)}
}

func (e *Execution) Close() {
	if e.vm == nil {
		return
	}
	if ctx := e.vm.context; ctx != nil && ctx.vm == e.vm {
		panic(RT.NewError("Cannot close an active execution"))
	}
	e.vm.Reset()
	vmPool.Put(e.vm)
	e.vm = nil
}

// Call always uses this execution's VM, not the ambient RT.vm. The caller
// must hold the GIL; independent Go callbacks must use their own Execution.
func (e *Execution) Call(fn *Fn, args []Object) Object {
	if e == nil || e.vm == nil {
		panic(RT.NewError("Execution is closed"))
	}
	previousVM, previousExpr := RT.vm, RT.currentExpr
	previousStack := RT.callstack
	defer func() { RT.vm, RT.currentExpr, RT.callstack = previousVM, previousExpr, previousStack }()
	if ctx := e.vm.context; ctx != nil && ctx.vm == e.vm {
		if e.vm.nativeDepth == 0 {
			panic(RT.NewError("Cannot re-enter an execution outside a native call"))
		}
		RT.vm = ctx
		fn.ensureCompiled()
		return e.vm.callCallback(fn, args)
	}
	RT.vm, RT.currentExpr = nil, nil
	RT.callstack = &Callstack{}
	fn.ensureCompiled()
	return e.vm.execute(fn, args, e)
}

// Evaluate compiles and executes a form on this execution. Macro evaluation
// during compilation still uses the legacy callable API until it is migrated.
func (e *Execution) Evaluate(expr Expr) Object {
	if e == nil || e.vm == nil {
		panic(RT.NewError("Execution is closed"))
	}
	if DISABLE_VM {
		return Eval(expr, nil)
	}
	previousVM, previousStack := RT.vm, RT.callstack
	if ctx := e.vm.context; ctx != nil && ctx.vm == e.vm {
		RT.vm = ctx
	} else {
		RT.vm = nil
		RT.callstack = &Callstack{}
	}
	defer func() { RT.vm, RT.callstack = previousVM, previousStack }()
	proto, err := CompileTopLevel(expr)
	PanicOnErr(err)
	return e.Call(&Fn{proto: proto, isCompiled: true}, nil)
}

// Evaluate is the legacy entry point for executing parsed code without an
// explicit Execution. Compilation errors are never hidden by AST fallback.
func Evaluate(expr Expr) Object {
	if DISABLE_VM {
		return Eval(expr, nil)
	}
	proto, err := CompileTopLevel(expr)
	PanicOnErr(err)
	return VMExecute(&Fn{proto: proto, isCompiled: true}, nil)
}

// CallIndependent starts a callback as a new logical execution, even if an
// unrelated VM is paused in a native call. The caller must hold the GIL.
// Synchronous callbacks from a VM should use Callable.Call directly instead.
func CallIndependent(fn Callable, args []Object) Object {
	previousVM, previousExpr, previousStack := RT.vm, RT.currentExpr, RT.callstack
	RT.vm, RT.currentExpr, RT.callstack = nil, nil, &Callstack{}
	defer func() { RT.vm, RT.currentExpr, RT.callstack = previousVM, previousExpr, previousStack }()
	if f, ok := fn.(*Fn); ok && !DISABLE_VM {
		exec := NewExecution()
		defer exec.Close()
		return exec.Call(f, args)
	}
	return fn.Call(args)
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
