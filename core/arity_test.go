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
		{"variadic", `(fn [x & xs] nil)`, []int{1, 2, 3, 4}, "at least 1"},
		{"rest only", `(fn [& xs] nil)`, []int{0, 1, 2, 3, 4}, "at least 0"},
		{"mixed", `(fn ([] nil) ([a b & xs] nil))`, []int{0, 2, 3, 4}, "0 or at least 2"},
		{"contiguous variadic", `(fn ([a b] nil) ([a b c] nil) ([a b c d] nil) ([a b c d & xs] nil))`, []int{2, 3, 4}, "at least 2"},
		{"variadic with gap", `(fn ([a] nil) ([a b c] nil) ([a b c & xs] nil))`, []int{1, 3, 4}, "1 or at least 3"},
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

func TestArityExpectedString(t *testing.T) {
	for _, tt := range []struct {
		name     string
		fixed    []int
		variadic int
		want     string
	}{
		{"fixed", []int{2}, -1, "2"},
		{"fixed gaps", []int{3, 0}, -1, "0 or 3"},
		{"fixed duplicates", []int{2, 1, 2}, -1, "1 or 2"},
		{"variadic", nil, 2, "at least 2"},
		{"any count", nil, 0, "at least 0"},
		{"map", []int{2, 3, 4}, 4, "at least 2"},
		{"conj", []int{2}, 2, "at least 2"},
		{"unsorted contiguous", []int{4, 2, 3, 2}, 4, "at least 2"},
		{"contiguous from zero", []int{0, 2, 1}, 3, "at least 0"},
		{"gap", []int{0}, 2, "0 or at least 2"},
		{"gap before suffix", []int{1, 3, 4}, 4, "1 or at least 3"},
		{"multiple gaps", []int{3, 0, 1}, 5, "0 or 1 or 3 or at least 5"},
		{"covered fixed counts", []int{5, 2, 3}, 2, "at least 2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := &ArityError{Fixed: tt.fixed, VariadicMin: tt.variadic}
			original := append([]int(nil), err.Fixed...)
			if got := err.ExpectedString(); got != tt.want {
				t.Fatalf("ExpectedString() = %q; want %q", got, tt.want)
			}
			if err.VariadicMin != tt.variadic {
				t.Fatal("formatter changed the variadic minimum")
			}
			for i, n := range original {
				if err.Fixed[i] != n {
					t.Fatal("formatter changed the fixed arities")
				}
			}
		})
	}
}

func TestArglistArityError(t *testing.T) {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()

	for _, tt := range []struct{ arglist, expected string }{
		{`([x y])`, "2"},
		{`([x] [x y z])`, "1 or 3"},
		{`([x] [x & xs])`, "at least 1"},
		{`([] [x y & xs])`, "0 or at least 2"},
		{`([x y z & xs] [x])`, "1 or at least 3"},
		{`([x & xs] [x y & ys])`, "at least 1"},
	} {
		obj, err := TryRead(NewReader(strings.NewReader(tt.arglist), "<arglist>"))
		if err != nil {
			t.Fatal(err)
		}
		arity := newArglistArityError(obj.(Seq), 0, "declared")
		if got := arity.ExpectedString(); got != tt.expected {
			t.Fatalf("arglist %s: expected = %q; want %q", tt.arglist, got, tt.expected)
		}
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
