package core

import "testing"

func TestVMContextAfterGILReacquisition(t *testing.T) {
	var aContext, bContext, afterResume, callbackContext *vmContext
	var aStack, bStack, afterResumeStack *Callstack
	var aExpr, afterResumeExpr Expr
	var resumedError *EvalError
	callback := Proc{Fn: func([]Object) Object {
		callbackContext = RT.vm
		return Int{I: 7}
	}}
	fnExpr := parseVMTest(t, `(fn [x] x)`).(*FnExpr)
	fnExpr.arities[0].body = []Expr{&CallExpr{callable: &LiteralExpr{obj: callback}}}
	callbackProto, err := CompileFnExpr(fnExpr, nil)
	if err != nil {
		t.Fatal(err)
	}
	lazy := NewMapSeq(&Fn{proto: callbackProto, isCompiled: true}, NewArrayVectorFrom(Int{I: 1}))
	bSuspended, aResumed, bDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
	bNative := Proc{Fn: func([]Object) Object {
		bContext, bStack = RT.vm, RT.callstack
		suspended := RT.Suspend()
		close(bSuspended)
		<-aResumed
		suspended.Resume()
		return NIL
	}}
	bProto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: bNative}})
	if err != nil {
		t.Fatal(err)
	}
	aNative := Proc{Fn: func([]Object) Object {
		aContext, aExpr, aStack = RT.vm, RT.currentExpr, RT.callstack
		suspended := RT.Suspend()
		<-bSuspended
		suspended.Resume()
		afterResume, afterResumeExpr, afterResumeStack = RT.vm, RT.currentExpr, RT.callstack
		resumedError = RT.NewError("after reacquiring GIL")
		value := lazy.First()
		close(aResumed)
		return value
	}}
	aProto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: aNative}})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		RT.LockIndependent()
		b := NewExecution()
		b.Call(&Fn{proto: bProto, isCompiled: true}, nil)
		b.Close()
		RT.GIL.Unlock()
		close(bDone)
	}()
	RT.GIL.Lock()
	a := NewExecution()
	result := a.Call(&Fn{proto: aProto, isCompiled: true}, nil)
	a.Close()
	RT.GIL.Unlock()
	<-bDone
	if result != (Int{I: 7}) || afterResume != aContext || callbackContext != aContext || bContext == aContext || afterResumeExpr != aExpr || resumedError.rt.currentExpr != aExpr || afterResumeStack != aStack || bStack == aStack {
		t.Fatalf("wrong context after reacquiring GIL: resume A=%v, callback A=%v, B differs=%v, expression A=%v, error A=%v", afterResume == aContext, callbackContext == aContext, bContext != aContext, afterResumeExpr == aExpr, resumedError.rt.currentExpr == aExpr)
	}
}

func TestGILIndependentEntryClearsAmbientContext(t *testing.T) {
	RT.GIL.Lock()
	previousVM, previousExpr, previousStack := RT.vm, RT.currentExpr, RT.callstack
	RT.vm = &vmContext{} // A native VM may be suspended when this host event arrives.
	RT.currentExpr = &CallExpr{}
	RT.GIL.Unlock()
	RT.LockIndependent()
	gotVM, gotExpr := RT.vm, RT.currentExpr
	RT.vm, RT.currentExpr, RT.callstack = previousVM, previousExpr, previousStack
	RT.GIL.Unlock()
	if gotVM != nil || gotExpr != nil {
		t.Fatal("independent GIL entry inherited another execution's context")
	}
}

func TestGILResumeRejectsExpiredContext(t *testing.T) {
	RT.GIL.Lock()
	previousVM, previousExpr, previousStack := RT.vm, RT.currentExpr, RT.callstack
	RT.GIL.Unlock()
	RT.LockIndependent()
	RT.vm = &vmContext{}
	suspended := RT.Suspend()
	func() {
		defer func() {
			RT.vm, RT.currentExpr, RT.callstack = previousVM, previousExpr, previousStack
			RT.GIL.Unlock()
		}()
		expectJokerPanic(t, func() { suspended.Resume() })
		if RT.vm != nil {
			t.Fatal("resumed an expired execution context")
		}
	}()
}

func TestVMExecutionContextExpiry(t *testing.T) {
	previous := RT.vm
	defer func() { RT.vm = previous }()
	var saved *vmContext
	proc := Proc{Fn: func([]Object) Object { saved = RT.vm; return NIL }}
	proto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: proc}})
	if err != nil {
		t.Fatal(err)
	}
	vm := NewVM()
	vm.ExecuteTopLevel(proto)
	if saved == nil || saved.vm != nil || saved.parent != nil {
		t.Fatal("finished execution retains pooled VM storage")
	}
	// A native caller may have saved a context while temporarily releasing the
	// GIL. Reusing its VM must not create a cycle or expose its new call frames.
	RT.vm = saved
	failing, err := CompileTopLevel(parseVMTest(t, `(nth [] 99)`))
	if err != nil {
		t.Fatal(err)
	}
	expectJokerPanic(t, func() { vm.ExecuteTopLevel(failing) })
	vm.Reset()
	for _, v := range vm.stack {
		if v != nil {
			t.Fatal("stack retained a value after failure")
		}
	}
	for _, f := range vm.frames {
		if f.closure != nil || f.arityProto != nil {
			t.Fatal("frame retained a function after failure")
		}
	}
	if vm.context != nil || vm.pendingCount != 0 || vm.handlerCount != 0 {
		t.Fatal("execution state leaked")
	}
}

func TestVMNativeCallbackReusesActiveVM(t *testing.T) {
	var caller *vmContext
	inner := Proc{Fn: func([]Object) Object {
		if RT.vm != caller || caller == nil || caller.vm == nil || caller.vm.frameCount < 2 {
			t.Fatal("callback did not execute on the native caller's VM")
		}
		return Int{I: 42}
	}}
	proto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: inner}})
	if err != nil {
		t.Fatal(err)
	}
	fn := &Fn{proto: proto, isCompiled: true}
	marker := Int{I: 99}
	outer := Proc{Fn: func(args []Object) Object {
		caller = RT.vm
		result := args[0].(*Fn).Call(nil)
		if RT.vm != caller || args[1] != marker {
			t.Fatal("native caller lost its context or borrowed arguments")
		}
		return result
	}}
	outerProto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: outer}, args: []Expr{
		&LiteralExpr{obj: fn}, &LiteralExpr{obj: marker},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result := NewVM().ExecuteTopLevel(outerProto)
	if result != (Int{I: 42}) || caller == nil || caller.vm != nil {
		t.Fatal("callback result or context expiry is incorrect")
	}
}

func TestNativeProcExplicitExecution(t *testing.T) {
	for _, name := range []string{"apply__", "eval__"} {
		vr := GLOBAL_ENV.CoreNamespace.Resolve(name)
		if vr == nil || vr.Value.(Proc).InExecution == nil {
			t.Fatalf("%s lacks an execution-aware native entry", name)
		}
	}
	owner := NewExecution()
	defer owner.Close()
	var caller *vmContext
	callback := Proc{Fn: func([]Object) Object {
		if RT.vm != caller || caller.vm != owner.vm {
			t.Fatal("nested callback used the ambient VM instead of its owner")
		}
		return Int{I: 42}
	}}
	callbackProto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: callback}})
	if err != nil {
		t.Fatal(err)
	}
	fn := &Fn{proto: callbackProto, isCompiled: true}
	native := Proc{
		Fn: func([]Object) Object {
			t.Fatal("VM bypassed the execution-aware entry")
			return NIL
		},
		InExecution: func(exec *Execution, _ []Object) Object {
			if exec != owner {
				t.Fatal("VM passed the wrong execution handle")
			}
			caller = RT.vm
			RT.vm = &vmContext{} // Simulate another execution changing ambient state.
			return exec.Call(fn, nil)
		},
	}
	proto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: native}})
	if err != nil {
		t.Fatal(err)
	}
	if result := owner.Call(&Fn{proto: proto, isCompiled: true}, nil); result != (Int{I: 42}) || caller.vm != nil {
		t.Fatal("execution-aware native call returned the wrong result or leaked its context")
	}
}

func TestRegisteredApplyAndEvalUseExecution(t *testing.T) {
	owner := NewExecution()
	defer owner.Close()
	var caller *vmContext
	check := Proc{Fn: func([]Object) Object {
		if RT.vm != caller || caller.vm != owner.vm {
			t.Fatal("callback did not use the explicit execution")
		}
		return Int{I: 42}
	}}
	callbackProto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: check}})
	if err != nil {
		t.Fatal(err)
	}
	callback := &Fn{proto: callbackProto, isCompiled: true}
	for _, tt := range []struct {
		name string
		args []Object
	}{
		{"apply__", []Object{callback, EmptyList}},
		{"eval__", []Object{NewListFrom(MakeSymbol("+"), Int{I: 40}, Int{I: 2})}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := GLOBAL_ENV.CoreNamespace.Resolve(tt.name).Value.(Proc)
			if p.InExecution == nil {
				t.Fatal("missing execution-aware entry")
			}
			original := p.InExecution
			p.InExecution = func(e *Execution, args []Object) Object {
				if e != owner {
					t.Fatal("VM did not pass the owning execution")
				}
				caller = RT.vm
				RT.vm = &vmContext{} // Native code may observe a different ambient context.
				return original(e, args)
			}
			exprs := make([]Expr, len(tt.args))
			for i, arg := range tt.args {
				exprs[i] = &LiteralExpr{obj: arg}
			}
			proto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: p}, args: exprs})
			if err != nil {
				t.Fatal(err)
			}
			if got := owner.Call(&Fn{proto: proto, isCompiled: true}, nil); !got.Equals(Int{I: 42}) {
				t.Fatalf("wrong result from %s: %s", tt.name, got)
			}
		})
	}
}

func TestExplicitExecutionInterleaving(t *testing.T) {
	firstExec, secondExec := NewExecution(), NewExecution()
	defer firstExec.Close()
	defer secondExec.Close()
	if firstExec.vm == secondExec.vm {
		t.Fatal("active executions share a VM")
	}
	var first, second *vmContext
	check := Proc{Fn: func([]Object) Object {
		if RT.vm != first || first.vm != firstExec.vm || firstExec.vm.frameCount < 2 {
			t.Fatal("explicit entry borrowed another execution's VM")
		}
		return Int{I: 42}
	}}
	checkProto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: check}})
	if err != nil {
		t.Fatal(err)
	}
	callback := &Fn{proto: checkProto, isCompiled: true}
	inSecond := Proc{Fn: func([]Object) Object {
		second = RT.vm
		result := firstExec.Call(callback, nil)
		if RT.vm != second || second.vm != secondExec.vm {
			t.Fatal("second execution's context was not restored")
		}
		return result
	}}
	secondProto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: inSecond}})
	if err != nil {
		t.Fatal(err)
	}
	secondFn := &Fn{proto: secondProto, isCompiled: true}
	inFirst := Proc{Fn: func([]Object) Object {
		first = RT.vm
		result := secondExec.Call(secondFn, nil)
		if RT.vm != first || first.vm != firstExec.vm {
			t.Fatal("first execution's context was not restored")
		}
		return result
	}}
	firstProto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: inFirst}})
	if err != nil {
		t.Fatal(err)
	}
	if result := firstExec.Call(&Fn{proto: firstProto, isCompiled: true}, nil); result != (Int{I: 42}) {
		t.Fatal("explicit execution returned the wrong result")
	}
	if first.vm != nil || second.vm != nil {
		t.Fatal("finished executions retained their published contexts")
	}
	if result := firstExec.Evaluate(&LiteralExpr{obj: Int{I: 7}}); result != (Int{I: 7}) {
		t.Fatal("execution could not be entered again")
	}
	uncompiled := &Fn{fnExpr: parseVMTest(t, `(fn [] 9)`).(*FnExpr)}
	if result := firstExec.Call(uncompiled, nil); !result.Equals(Int{I: 9}) {
		t.Fatal("execution did not compile a callable on entry")
	}
}

func TestVMNativeEntriesShareCaller(t *testing.T) {
	var caller *vmContext
	check := Proc{Fn: func([]Object) Object {
		if caller == nil || RT.vm != caller || caller.vm == nil || caller.vm.frameCount < 2 {
			t.Fatal("nested entry did not use the caller's VM")
		}
		return Int{I: 42}
	}}
	innerExpr := &CallExpr{callable: &LiteralExpr{obj: check}}
	innerProto, err := CompileTopLevel(innerExpr)
	if err != nil {
		t.Fatal(err)
	}
	fn := &Fn{proto: innerProto, isCompiled: true}
	for _, entry := range []struct {
		name string
		call func() Object
	}{
		{"Fn.Call", func() Object { return fn.Call(nil) }},
		{"Evaluate", func() Object { return Evaluate(innerExpr) }},
		{"VMExecute", func() Object { return VMExecute(fn, nil) }},
	} {
		t.Run(entry.name, func(t *testing.T) {
			foreign := Proc{Package: "test", Fn: func([]Object) Object {
				caller = RT.vm
				result := entry.call()
				if RT.vm != caller || caller.vm == nil {
					t.Fatal("native entry lost its caller's context")
				}
				return result
			}}
			outer, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: foreign}})
			if err != nil {
				t.Fatal(err)
			}
			if result := NewVM().ExecuteTopLevel(outer); result != (Int{I: 42}) || caller == nil || caller.vm != nil {
				t.Fatal("nested entry returned the wrong value or retained its VM")
			}
		})
	}
}

func TestVMIndependentCallbackDoesNotReusePausedVM(t *testing.T) {
	var parent, child *vmContext
	inner := Proc{Fn: func([]Object) Object {
		child = RT.vm
		if parent == nil || child == nil || child.vm == nil || child.vm == parent.vm || child.parent != nil {
			t.Fatal("independent callback borrowed the paused VM")
		}
		return Int{I: 42}
	}}
	innerProto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: inner}})
	if err != nil {
		t.Fatal(err)
	}
	fn := &Fn{proto: innerProto, isCompiled: true}
	outer := Proc{Fn: func([]Object) Object {
		parent = RT.vm
		result := CallIndependent(fn, nil)
		if RT.vm != parent || parent.vm == nil {
			t.Fatal("independent callback lost the paused context")
		}
		return result
	}}
	outerProto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: outer}})
	if err != nil {
		t.Fatal(err)
	}
	if result := NewVM().ExecuteTopLevel(outerProto); result != (Int{I: 42}) || child == nil || child.vm != nil || parent.vm != nil {
		t.Fatal("independent callback leaked a VM or returned the wrong result")
	}
}

func TestVMNestedEvaluateBoundaries(t *testing.T) {
	for _, tt := range []struct{ code, want string }{
		{`(let [f (fn [] (eval '(when true 42)))] (apply f []))`, `42`},
		{`(try (eval '(throw (ex-info "inner" {}))) (catch Error e :caught))`, `:caught`},
		{`(try (eval '(try (throw (ex-info "inner" {})) (finally (throw (ex-info "finally" {}))))) (catch Error e (ex-message e)))`, `"finally"`},
	} {
		result, err := TryEvaluate(parseVMTest(t, tt.code))
		if err != nil {
			t.Fatal(err)
		}
		if got := result.ToString(true); got != tt.want {
			t.Fatalf("got %s, want %s", got, tt.want)
		}
	}
}

func TestVMNestedEvaluateCaughtErrorPreservesCaller(t *testing.T) {
	failure := &CallExpr{callable: &LiteralExpr{obj: Proc{Fn: func([]Object) Object {
		panic(RT.NewError("nested error"))
	}}}}
	foreign := Proc{Package: "test", Fn: func([]Object) Object {
		ctx := RT.vm
		stackTop, frameCount := ctx.vm.stackTop, ctx.vm.frameCount
		if _, err := TryEvaluate(failure); err == nil {
			t.Fatal("nested evaluation did not fail")
		}
		if RT.vm != ctx || ctx.vm.stackTop != stackTop || ctx.vm.frameCount != frameCount {
			t.Fatal("caught nested error damaged the caller's VM")
		}
		return Evaluate(&LiteralExpr{obj: Int{I: 42}})
	}}
	proto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: foreign}})
	if err != nil {
		t.Fatal(err)
	}
	if result := NewVM().ExecuteTopLevel(proto); result != (Int{I: 42}) {
		t.Fatal("caller did not resume after the nested error")
	}
}

func TestVMNativeCallbackHostPanicCleanup(t *testing.T) {
	failure := Proc{Fn: func([]Object) Object { panic("callback failure") }}
	inner, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: failure}})
	if err != nil {
		t.Fatal(err)
	}
	fn := &Fn{proto: inner, isCompiled: true}
	outer := Proc{Fn: func([]Object) Object { return fn.Call(nil) }}
	proto, err := CompileTopLevel(&CallExpr{callable: &LiteralExpr{obj: outer}})
	if err != nil {
		t.Fatal(err)
	}
	vm := NewVM()
	func() {
		defer func() {
			if got := recover(); got != "callback failure" {
				t.Fatalf("unexpected panic: %v", got)
			}
		}()
		vm.ExecuteTopLevel(proto)
	}()
	if vm.nativeDepth != 0 || vm.frameCount > 1 || vm.handlerCount != 0 || vm.pendingCount != 0 || vm.context.vm != nil {
		t.Fatal("callback panic retained active VM state")
	}
	vm.Reset()
	for _, obj := range vm.stack {
		if obj != nil {
			t.Fatal("callback panic retained stack values")
		}
	}
}

func TestVMNativeCallbackFrameGrowthAndExceptions(t *testing.T) {
	for _, tt := range []struct{ code, want string }{
		{`(let [f (fn f [n] (if (zero? n) 0 (inc (f (dec n)))))] (apply f [130]))`, `130`},
		{`(try (apply (fn [] (throw (ex-info "callback" {}))) []) (catch Error e :caught))`, `:caught`},
		{`(let [a (atom [])] (try (apply (fn [] (try (throw (ex-info "callback" {})) (finally (swap! a conj :finally)))) []) (catch Error e (swap! a conj :caught))) @a)`, `[:finally :caught]`},
	} {
		result, err := TryEvaluate(parseVMTest(t, tt.code))
		if err != nil {
			t.Fatal(err)
		}
		if got := result.ToString(true); got != tt.want {
			t.Fatalf("got %s, want %s", got, tt.want)
		}
	}
}

func TestVMNativeCallbackTrace(t *testing.T) {
	code := `(let [f (fn [] (nth [] 9))] (apply f []))`
	_, err := TryEvaluate(parseVMTest(t, code))
	if err == nil {
		t.Fatal("expected callback error")
	}
	e := err.(*EvalError)
	if len(e.rt.callstack.frames) < 2 {
		t.Fatalf("callback caller frames lost: %v", err)
	}
	if e.rt.vm != nil {
		t.Fatal("error snapshot retained a live VM")
	}
}

func TestVMNativeContextInterleaving(t *testing.T) {
	for _, fail := range []bool{false, true} {
		var saved *vmContext
		checked := false
		interleave := Proc{Fn: func([]Object) Object {
			saved = RT.vm
			RT.vm = &vmContext{}
			if fail {
				panic("interleaved failure")
			}
			return NIL
		}}
		check := Proc{Fn: func([]Object) Object {
			if saved == nil || RT.vm != saved {
				t.Fatal("native call lost its caller's execution context")
			}
			checked = true
			return NIL
		}}
		expr := &TryExpr{
			body:        []Expr{&CallExpr{callable: &LiteralExpr{obj: interleave}}},
			finallyExpr: []Expr{&CallExpr{callable: &LiteralExpr{obj: check}}},
		}
		func() {
			defer func() {
				r := recover()
				if (fail && r != "interleaved failure") || (!fail && r != nil) {
					t.Fatalf("unexpected failure: %v", r)
				}
			}()
			Evaluate(expr)
		}()
		if !checked {
			t.Fatal("finally did not execute after the native call")
		}
	}
}
