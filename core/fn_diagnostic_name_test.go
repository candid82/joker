package core

import (
	"strings"
	"testing"
)

func TestRuntimeFnDiagnosticNames(t *testing.T) {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()

	ns := GLOBAL_ENV.CurrentNamespace().Name.ToString(false) + "/"
	for _, tt := range []struct {
		name, code, function, expected string
	}{
		{"core defn", `(map [])`, "joker.core/map", "at least 2"},
		{"explicit core name", `(conj [])`, "conj", "at least 2"},
		{"defn", `(do (defn diagnostic-fixed [x] x) (diagnostic-fixed))`, ns + "diagnostic-fixed", "1"},
		{"def", `(do (def diagnostic-def (fn [x] x)) (diagnostic-def))`, ns + "diagnostic-def", "1"},
		{"def metadata", `(do (def diagnostic-meta ^{:note true} (fn [x] x)) (diagnostic-meta))`, ns + "diagnostic-meta", "1"},
		{"explicit def name", `(do (def diagnostic-explicit (fn explicit-name [x] x)) (diagnostic-explicit))`, "explicit-name", "1"},
		{"multi arity", `(do (defn diagnostic-multi ([x] x) ([x y] y)) (diagnostic-multi))`, ns + "diagnostic-multi", "1 or 2"},
		{"variadic", `(do (defn diagnostic-variadic [x & xs] xs) (diagnostic-variadic))`, ns + "diagnostic-variadic", "at least 1"},
		{"def alias", `(do (defn diagnostic-original [x] x) (def diagnostic-alias diagnostic-original) (diagnostic-alias))`, ns + "diagnostic-original", "1"},
		{"local alias", `(let [original (fn [x] x) alias original] (alias))`, "original", "1"},
		{"core alias", `(let [f map] (f []))`, "joker.core/map", "at least 2"},
		{"apply", `(apply map [[]])`, "joker.core/map", "at least 2"},
		{"native callback", `(mapv (fn [] nil) [1])`, "<anonymous>", "0"},
		{"named native callback", `(let [callback (fn [] nil)] (mapv callback [1]))`, "callback", "0"},
		{"local", `(let [f (fn [x] x)] (f))`, "f", "1"},
		{"metadata", `(let [f ^{:note true} (fn [x] x)] (f))`, "f", "1"},
		{"loop initializer", `(loop [f (fn [x] x)] (f))`, "f", "1"},
		{"letfn", `(letfn [(f [x] x)] (f))`, "f", "1"},
		{"explicit name", `(let [f (fn explicit-name [x] x)] (f))`, "explicit-name", "1"},
		{"anonymous", `((fn [x] x))`, "<anonymous>", "1"},
		{"returned function", `(let [factory (fn [] (fn [x] x)) f (factory)] (f))`, "<anonymous>", "1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := TryEvaluate(parseVMTest(t, tt.code))
			evalErr, ok := err.(*EvalError)
			if !ok {
				t.Fatalf("expected EvalError, got %v", err)
			}
			want := "passed to " + tt.function + ", expected: " + tt.expected
			if !strings.Contains(evalErr.msg, want) {
				t.Fatalf("error = %s; want %s", evalErr.msg, want)
			}
		})
	}
}

func TestFnDiagnosticNamePreservesBindings(t *testing.T) {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()

	for _, tt := range []struct{ code, want string }{
		{`(let [f :outer f (fn [] f)] (f))`, ":outer"},
		{`(do
		     (defn diagnostic-redefinition [n]
		       (if (zero? n) :original (diagnostic-redefinition 0)))
		     (let [original diagnostic-redefinition]
		       (with-redefs [diagnostic-redefinition (fn [_] :replacement)]
		         (original 1))))`, ":replacement"},
	} {
		result, err := TryEvaluate(parseVMTest(t, tt.code))
		if err != nil {
			t.Fatal(err)
		}
		if got := result.ToString(true); got != tt.want {
			t.Fatalf("got %s; want %s", got, tt.want)
		}
	}
}

func TestFnDiagnosticNameParsedAndPacked(t *testing.T) {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()

	def := parseVMTest(t, `(def diagnostic-parsed (fn [x] x))`).(*DefExpr)
	expr := def.value.(*FnExpr)
	if expr.self.name != nil || expr.selfBinding != nil {
		t.Fatal("diagnostic name introduced a self-binding")
	}
	parsed := &Fn{fnExpr: expr}
	arity := parsed.CheckArity(0) // Also checks lazy compilation retains the name.
	if arity == nil || arity.name != def.vr.Name() {
		t.Fatalf("unexpected arity error: %v", arity)
	}

	env := NewPackEnv()
	packed := parsed.proto.Pack(nil, env)
	header, _ := UnpackHeader(env.Pack(nil), GLOBAL_ENV)
	proto, rest := UnpackFunctionProto(packed, header)
	if len(rest) != 0 {
		t.Fatalf("%d bytes remained after unpacking", len(rest))
	}
	for _, fn := range []*Fn{parsed, {proto: proto, isCompiled: true}} {
		func() {
			defer func() {
				r := recover()
				err, ok := r.(*EvalError)
				if !ok || err.msg != arity.Error() {
					t.Errorf("runtime error = %v; want %s", r, arity)
				}
			}()
			fn.Call(nil) // No Joker call site is necessary to identify the function.
		}()
	}
}
