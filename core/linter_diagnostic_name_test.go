package core

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestLinterFunctionNameMatchesRuntime(t *testing.T) {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()
	previousStderr, previousMode, previousProblems := Stderr, LINTER_MODE, PROBLEM_COUNT
	defer func() {
		Stderr, LINTER_MODE, PROBLEM_COUNT = previousStderr, previousMode, previousProblems
	}()

	for _, code := range []string{
		`(map [])`,
		`(conj [])`,
		`(first)`, // Native callables keep their call-site names.
		`((fn [x] x))`,
		`((fn ([a] a) ([a b c] a)) 1 2)`,
		`((fn ([] nil) ([a b & xs] a)) 1)`,
		`((fn ([a b] a) ([a b c] a) ([a b c d & xs] a)) 1)`,
		`(let [original (fn [x] x) renamed original] (renamed))`,
		`(let [renamed map] (renamed []))`,
		`(let [renamed (fn self-name [x] x)] (renamed))`,
		`(let [renamed ^{:note true} (fn [x] x)] (renamed))`,
	} {
		t.Run(code, func(t *testing.T) {
			LINTER_MODE = false
			expr := parseVMTest(t, code)
			_, runtimeErr := TryEvaluate(expr)
			evalErr, ok := runtimeErr.(*EvalError)
			if !ok {
				t.Fatalf("expected runtime EvalError, got %v", runtimeErr)
			}
			callExpr := expr
			if let, ok := expr.(*LetExpr); ok {
				callExpr = let.body[0]
			}
			call := callExpr.(*CallExpr)
			name := call.diagnosticFunctionName()
			if !strings.Contains(evalErr.msg, "passed to "+name) {
				t.Fatalf("runtime error %q does not use linter function name %q", evalErr.msg, name)
			}

			var output bytes.Buffer
			Stderr, LINTER_MODE = &output, true
			checkLinterCall(call, &ParseContext{GlobalEnv: GLOBAL_ENV}, call.Pos())
			if !strings.Contains(output.String(), "passed to "+name+", expected: ") {
				t.Fatalf("linter warning %q does not include expected arities for %q", output.String(), name)
			}
			if strings.Contains(evalErr.msg, ", expected: ") && !strings.Contains(output.String(), evalErr.msg+"\n") {
				t.Fatalf("linter warning %q does not match runtime message %q", output.String(), evalErr.msg)
			}
		})
	}
}

func TestFnSummaryDiagnosticNames(t *testing.T) {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()

	for _, code := range []string{`(fn [x] x)`, `(fn self-name [x] x)`} {
		expr := parseVMTest(t, code).(*FnExpr)
		summary := getFnSummary(expr)
		// The parser can assign an initializer's name after inference has
		// already cached a summary while parsing the body.
		inferFnDiagnosticName(expr, "inferred-name")
		proto, err := CompileFnExpr(expr, nil)
		if err != nil {
			t.Fatal(err)
		}
		compact := compactFnSummary(expr)
		if compact.fn != nil {
			t.Fatal("compact summary retained the AST")
		}
		for _, s := range []*FnSummary{summary, compact} {
			if name := s.functionName(); name != proto.Name {
				t.Fatalf("summary name = %q; runtime name = %q", name, proto.Name)
			}
		}
	}
}

func TestLinterMacroArityMessages(t *testing.T) {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()
	previousStderr, previousProblems := Stderr, PROBLEM_COUNT
	defer func() { Stderr, PROBLEM_COUNT = previousStderr, previousProblems }()

	for _, tt := range []struct {
		code     string
		argc     int
		expected string
	}{
		{`(fn [&form &env x] x)`, 0, "1"},
		{`(fn [&form &env x & xs] x)`, 0, "at least 1"},
		{`(fn ([&form &env] nil) ([&form &env x y & xs] x))`, 1, "0 or at least 2"},
	} {
		expr := parseVMTest(t, tt.code).(*FnExpr)
		inferFnDiagnosticName(expr, "macro-name")
		proto, err := CompileFnExpr(expr, nil)
		if err != nil {
			t.Fatal(err)
		}
		call := &CallExpr{args: make([]Expr, tt.argc)}
		want := fmt.Sprintf("Wrong number of args (%d) passed to macro-name, expected: %s", tt.argc, tt.expected)
		for _, report := range []func(){
			func() { reportWrongArity(expr, true, call, expr.Pos()) },
			func() { reportWrongSummaryArity(compactFnSummary(expr), true, call, expr.Pos()) },
			func() { printArityWarning(expr.Pos(), newArityError(proto, tt.argc+2), true) },
		} {
			var output bytes.Buffer
			Stderr = &output
			report()
			if !strings.Contains(output.String(), want+"\n") {
				t.Fatalf("macro warning = %q; want %q", output.String(), want)
			}
		}
	}
}

func TestCallableDiagnosticNameCycle(t *testing.T) {
	vr := &Var{}
	ref := &VarRefExpr{vr: vr}
	vr.expr = ref
	if name := callableFunctionName(ref); name != "" {
		t.Fatalf("cyclic alias has function name %q", name)
	}
}
