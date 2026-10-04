package core

import (
	"strings"
	"testing"
)

func TestFnCheckArity(t *testing.T) {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()

	for _, tt := range []struct {
		name     string
		code     string
		accepted []int
		expected string
	}{
		{"zero", `(fn [] nil)`, []int{0}, "0"},
		{"fixed", `(fn [x] nil)`, []int{1}, "1"},
		{"multiple", `(fn ([] nil) ([a b c] nil))`, []int{0, 3}, "0 or 3"},
		{"variadic", `(fn [x & xs] nil)`, []int{1, 2, 3, 4}, "1+"},
		{"rest only", `(fn [& xs] nil)`, []int{0, 1, 2, 3, 4}, "0+"},
		{"mixed", `(fn ([] nil) ([a b & xs] nil))`, []int{0, 2, 3, 4}, "0 or 2+"},
		{"misleading metadata", `(with-meta (fn [] nil) {:arglists '([x])})`, []int{0}, "0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fn := evalAndCompile(t, tt.code).(*Fn)
			for argc := 0; argc <= 4; argc++ {
				wantAccepted := false
				for _, n := range tt.accepted {
					if n == argc {
						wantAccepted = true
					}
				}
				arityErr := fn.CheckArity(argc)
				if (arityErr == nil) != wantAccepted {
					t.Fatalf("CheckArity(%d) = %v; accepted = %v", argc, arityErr, wantAccepted)
				}
				args := make([]Object, argc)
				for i := range args {
					args[i] = NIL
				}
				if wantAccepted {
					fn.Call(args)
					continue
				}
				if arityErr.Actual != argc || arityErr.ExpectedString() != tt.expected {
					t.Fatalf("incorrect arity details: %+v", arityErr)
				}
				func() {
					defer func() {
						r := recover()
						err, ok := r.(*EvalError)
						if !ok || err.msg != arityErr.Error() {
							t.Errorf("VM error = %v; want %s", r, arityErr)
						}
					}()
					fn.Call(args)
				}()
			}
		})
	}
}

func TestFnCheckArityCompilesParsedFunction(t *testing.T) {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()

	expr := parseVMTest(t, `(fn [x] x)`).(*FnExpr)
	fn := &Fn{fnExpr: expr}
	if err := fn.CheckArity(1); err != nil || fn.proto == nil {
		t.Fatalf("CheckArity did not compile the function: %v", err)
	}
	if err := fn.CheckArity(0); err == nil || err.ExpectedString() != "1" {
		t.Fatalf("unexpected arity result: %v", err)
	}
}

func TestFnCheckArityTopLevel(t *testing.T) {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()

	fn := &Fn{proto: NewFunctionProto("<top-level>"), isCompiled: true}
	if err := fn.CheckArity(0); err != nil {
		t.Fatal(err)
	}
	if err := fn.CheckArity(1); err == nil || err.ExpectedString() != "0" {
		t.Fatalf("unexpected top-level arity: %v", err)
	}
}

func TestNativeErrorsWithoutCallSite(t *testing.T) {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()
	previousVM, previousExpr, previousStack := RT.vm, RT.currentExpr, RT.callstack
	RT.vm, RT.currentExpr, RT.callstack = nil, nil, &Callstack{}
	defer func() { RT.vm, RT.currentExpr, RT.callstack = previousVM, previousExpr, previousStack }()

	for _, tt := range []struct {
		name string
		call func()
		want string
	}{
		{"arity", func() { PanicArity(2) }, "Wrong number of args (2) passed to <native>"},
		{"arity range", func() { PanicArityMinMax(0, 1, 2) }, "Wrong number of args (0) passed to <native>; expects 1 or 2"},
		{"argument type", func() { panic(RT.NewArgTypeError(0, NIL, "String")) }, "Arg[0] of <native> must have type String, got Nil"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				err, ok := r.(*EvalError)
				if !ok {
					t.Fatalf("expected EvalError, got %v", r)
				}
				if err.msg != tt.want || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("error = %v; want %s", err, tt.want)
				}
			}()
			tt.call()
		})
	}
}
