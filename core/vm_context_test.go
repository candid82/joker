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
