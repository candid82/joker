package core

import "testing"

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
