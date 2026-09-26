package core

import (
	"fmt"
	"strings"
	"testing"
)

func callASTForTest(fn *Fn, args []Object) Object {
	old := DISABLE_VM
	DISABLE_VM = true
	defer func() { DISABLE_VM = old }()
	return fn.callAST(args)
}

func parseVMTest(t *testing.T, code string) Expr {
	t.Helper()
	obj, err := TryRead(NewReader(strings.NewReader(code), "<vm-regression>"))
	if err != nil {
		t.Fatal(err)
	}
	expr, err := TryParse(obj, &ParseContext{GlobalEnv: GLOBAL_ENV})
	if err != nil {
		t.Fatal(err)
	}
	return expr
}

func TestVMRegressionParity(t *testing.T) {
	cases := []struct{ name, code, want string }{
		{"metadata", `(meta ^{:a 1} [1 2])`, `{:a 1}`},
		{"metadata side effects", `(let [a (atom []) v ^{:a (swap! a conj :meta)} [(swap! a conj :body)]] @a)`, `[:meta :body]`},
		{"multi arity captures", `(let [x 10 y 20 f (fn ([] x) ([a] y))] [(f) (f 0)])`, `[10 20]`},
		{"second arity only capture", `(let [x 10 f (fn ([] 0) ([a] x))] (f 0))`, `10`},
		{"variadic captures", `(let [x 10 y 20 f (fn ([] x) ([a & xs] [y xs]))] [(f) (f 1 2)])`, `[10 [20 (2)]]`},
		{"nested captures in later arity", `(let [x 10 y 20 f (fn ([] x) ([a] (fn [] y)))] ((f 1)))`, `20`},
		{"letfn shadow", `(let [g (fn [] :outer)] (letfn [(f [] (g)) (g [] :inner)] (f)))`, `:inner`},
		{"letfn escaped", `(let [f (letfn [(f [n] (if (= n 0) :done (g (dec n)))) (g [n] (f n))] f)] (apply f [100]))`, `:done`},
		{"letfn in loop recur target", `(loop [n 3] (if (zero? n) n (letfn [(f [] 1)] (recur (dec n)))))`, `0`},
		{"var rebinding", `(with-redefs [+ (fn [a b] 99)] (+ 1 2))`, `99`},
		{"callable check order", `(let [a (atom 0)] (try (42 (reset! a 1)) (catch Error e nil)) @a)`, `0`},
		{"recur captures", `(loop [i 0 fs []] (if (= i 3) (mapv (fn [f] (f)) fs) (recur (inc i) (conj fs (fn [] i)))))`, `[0 1 2]`},
		{"function recur captures", `((fn [i fs] (if (= i 3) (mapv (fn [f] (f)) fs) (recur (inc i) (conj fs (fn [] i))))) 0 [])`, `[0 1 2]`},
		{"recursion depth", `((fn f [n] (if (= n 0) 0 (f (dec n)))) 1000)`, `0`},
		{"eval", `(eval '(letfn [(f [] (g)) (g [] :ok)] (f)))`, `:ok`},
		{"eval opaque constant", `(let [a (atom 1)] (identical? a (eval (list 'quote a))))`, `true`},
		{"binding shadow initializer", `(let [x 1 x (fn [] x)] (x))`, `1`},
		{"captured self shadow", `((fn f [n] (let [g (fn [] f) f 42] (identical? (g) (g)))) 0)`, `true`},
		{"native callback and letfn", `(letfn [(f [n] (if (zero? n) 0 (apply g [(dec n)]))) (g [n] (f n))] (f 20))`, `0`},
		{"load string", `(load-string "(meta ^{:a 1} [])")`, `{:a 1}`},
		{"quoted vector", `'[1 {:a #{2}}]`, `[1 {:a #{2}}]`},
		{"catch closure", `(let [f (try (throw (ex-info "hello" {})) (catch Error e (fn [] (ex-message e))))] (f))`, `"hello"`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			expr := parseVMTest(t, tt.code)
			ast, err := TryEval(expr)
			if err != nil {
				t.Fatal(err)
			}
			vm, err := TryEvaluate(expr)
			if err != nil {
				t.Fatal(err)
			}
			if got := vm.ToString(true); got != tt.want || !ast.Equals(vm) {
				t.Fatalf("AST %s, VM %s; want %s", ast.ToString(true), got, tt.want)
			}
		})
	}
}

func TestVMWideOperands(t *testing.T) {
	cases := []struct{ code, want string }{
		{`(count (list ` + strings.Repeat("1 ", 300) + `))`, `300`},
		{`(count [` + strings.Repeat("0 ", 20000) + `])`, `20000`},
		{strings.Repeat("(try ", 100) + `42` + strings.Repeat(" (finally nil))", 100), `42`},
		{`(last [` + strings.Repeat("0 ", 300) + `(let [x 42] x)])`, `42`},
		{`(let [` + func() string {
			var s strings.Builder
			for i := 0; i < 300; i++ {
				fmt.Fprintf(&s, "x%d %d ", i, i)
			}
			return s.String()
		}() + `] x299)`, `299`},
		{`(if false (do ` + strings.Repeat("1 ", 7000) + `) 42)`, `42`},
		{`(loop [n 1] (if (= n 0) 42 (do ` + strings.Repeat("1 ", 12000) + `(recur (dec n)))))`, `42`},
	}
	for _, tt := range cases {
		expr := parseVMTest(t, tt.code)
		result, err := TryEvaluate(expr)
		if err != nil {
			t.Fatal(err)
		}
		if result.ToString(true) != tt.want {
			t.Fatalf("got %s, want %s", result, tt.want)
		}
	}
}

func TestVMHostPanicFinally(t *testing.T) {
	events := EmptyArrayVector()
	panicProc := Proc{Fn: func([]Object) Object { panic("host failure") }}
	record := Proc{Fn: func([]Object) Object { events.Append(MakeKeyword("finally")); return NIL }}
	expr := &TryExpr{body: []Expr{&CallExpr{callable: &LiteralExpr{obj: panicProc}}}, finallyExpr: []Expr{&CallExpr{callable: &LiteralExpr{obj: record}}}}
	for _, ast := range []bool{true, false} {
		events.arr = nil
		func() {
			defer func() {
				if r := recover(); r != "host failure" {
					t.Errorf("panic: %v", r)
				}
			}()
			if ast {
				_, _ = TryEval(expr)
			} else {
				Evaluate(expr)
			}
		}()
		if events.Count() != 1 {
			t.Fatalf("finally did not run (AST=%v)", ast)
		}
	}
}

func TestVMMacroExecution(t *testing.T) {
	source := `(defmacro vm-regression-macro [] (if (:a (meta ^{:a true} [])) 1 2)) (def vm-regression-result (vm-regression-macro))`
	err := ProcessReader(NewReader(strings.NewReader(source), "<macro-test>"), "", EVAL)
	if err != nil {
		t.Fatal(err)
	}
	vr, ok := GLOBAL_ENV.Resolve(MakeSymbol("vm-regression-result"))
	if !ok || !vr.Value.Equals(Int{I: 1}) {
		t.Fatal("macro did not execute correctly")
	}
	macro, _ := GLOBAL_ENV.Resolve(MakeSymbol("vm-regression-macro"))
	if macro.Value.(*Fn).proto == nil {
		t.Fatal("macro was not compiled")
	}
}

func TestVMRuntimeObjectConstants(t *testing.T) {
	obj := &Atom{value: Int{I: 7}}
	expr := &LiteralExpr{obj: obj}
	if got := Evaluate(expr); got != obj {
		t.Fatal("runtime constant identity lost")
	}
}
