package core

import (
	"strings"
	"testing"
)

func expectJokerPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected Joker error")
		} else if _, ok := r.(Error); !ok {
			t.Errorf("host panic instead of Joker error: %v", r)
		}
	}()
	f()
}

func TestVMPackedValidation(t *testing.T) {
	expr := parseVMTest(t, `(let [x 1 y 2] (fn ([] x) ([a] (try y (finally nil)))))`)
	proto, err := CompileTopLevel(expr)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateFunctionProto(proto); err != nil {
		t.Fatal(err)
	}
	env := NewPackEnv()
	data := proto.Pack(nil, env)
	headerData := env.Pack(nil)
	for n := 0; n < len(headerData); n++ {
		expectJokerPanic(t, func() { UnpackHeader(headerData[:n], GLOBAL_ENV) })
	}
	for n := 0; n < len(data); n++ {
		h, _ := UnpackHeader(headerData, GLOBAL_ENV)
		expectJokerPanic(t, func() { UnpackFunctionProto(data[:n], h) })
	}
	h, _ := UnpackHeader(headerData, GLOBAL_ENV)
	unpacked, rest := UnpackFunctionProto(data, h)
	if len(rest) != 0 {
		t.Fatal("trailing packed data")
	}
	fn := NewVM().ExecuteTopLevel(unpacked).(*Fn)
	if !fn.Call(nil).Equals(Int{I: 1}) || !fn.Call([]Object{NIL}).Equals(Int{I: 2}) {
		t.Fatal("packed captures changed")
	}
	expectJokerPanic(t, func() { UnpackHeader([]byte("old-format"), GLOBAL_ENV) })
}

func TestVMRejectsInvalidBytecode(t *testing.T) {
	cases := []struct {
		name      string
		code      []byte
		constants []Object
	}{
		{"unknown", []byte{255}, nil},
		{"truncated operand", []byte{byte(OP_CONST)}, nil},
		{"constant index", []byte{byte(OP_CONST), 0, 0, 0, 0, byte(OP_RETURN)}, nil},
		{"local index", []byte{byte(OP_GET_LOCAL), 0, 0, 1, 0, byte(OP_RETURN)}, nil},
		{"jump into operand", []byte{byte(OP_JUMP), 0, 0, 0, 1, byte(OP_CONST), 0, 0, 0, 0, byte(OP_RETURN)}, []Object{NIL}},
		{"stack underflow", []byte{byte(OP_POP), byte(OP_RETURN)}, nil},
		{"fallthrough", []byte{byte(OP_NIL)}, nil},
		{"non-var", []byte{byte(OP_GET_VAR), 0, 0, 0, 0, byte(OP_RETURN)}, []Object{NIL}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			p := NewFunctionProto(tt.name)
			p.Chunk.Code = tt.code
			p.Chunk.Positions = make([]Position, len(tt.code))
			p.Chunk.Constants = tt.constants
			if err := ValidateFunctionProto(p); err == nil {
				t.Fatal("accepted invalid bytecode")
			}
			env := NewPackEnv()
			data := p.Pack(nil, env)
			h, _ := UnpackHeader(env.Pack(nil), GLOBAL_ENV)
			expectJokerPanic(t, func() { UnpackFunctionProto(data, h) })
		})
	}
}

func TestVMPackedObjectGraph(t *testing.T) {
	shared := NewListFrom(Int{I: 7})
	meta := EmptyArrayMap()
	meta.Add(MakeKeyword("a"), shared)
	vector := NewArrayVectorFrom(shared, shared, GLOBAL_ENV.CoreNamespace.Resolve("*out*"), TYPE.Fn).WithMeta(meta)
	proto, err := CompileTopLevel(&LiteralExpr{obj: vector})
	if err != nil {
		t.Fatal(err)
	}
	env := NewPackEnv()
	data := proto.Pack(nil, env)
	h, _ := UnpackHeader(env.Pack(nil), GLOBAL_ENV)
	unpacked, _ := UnpackFunctionProto(data, h)
	result := NewVM().ExecuteTopLevel(unpacked).(Vec)
	if result.At(0) != result.At(1) {
		t.Fatal("shared constant identity lost")
	}
	_, m := result.(Meta).GetMeta().Get(MakeKeyword("a"))
	if m != result.At(0) {
		t.Fatal("metadata sharing lost")
	}
	if result.At(2) != GLOBAL_ENV.CoreNamespace.Resolve("*out*") || result.At(3) != TYPE.Fn {
		t.Fatal("canonical Var/Type identity lost")
	}
	if NewVM().ExecuteTopLevel(unpacked) != result {
		t.Fatal("repeated constant identity lost")
	}
}

func TestVMStrictRuntime(t *testing.T) {
	expr := parseVMTest(t, `(fn [] 42)`)
	expectJokerPanic(t, func() { Eval(expr, nil) })
	fn := expr.(*FnExpr).Eval(nil).(*Fn)
	expectJokerPanic(t, func() { fn.callAST(nil) })
	// First invocation must compile even when a native caller, not OP_CALL,
	// encounters a generated/AST-backed function.
	if result := fn.Call(nil); !result.Equals(Int{I: 42}) || fn.proto == nil {
		t.Fatal("native call did not compile")
	}
	// Compiler failure cannot invoke the AST body.
	bad := &Fn{fnExpr: &FnExpr{arities: []FnArityExpr{{body: []Expr{&CatchExpr{}}}}}}
	expectJokerPanic(t, func() { bad.Call(nil) })
}

func TestVMErrorTrace(t *testing.T) {
	expr := parseVMTest(t, "(let [f (fn []\n (nth [] 3))]\n (f))")
	_, err := TryEvaluate(expr)
	if err == nil {
		t.Fatal("expected error")
	}
	message := err.Error()
	for _, site := range []string{"<vm-regression>:3:2", "<vm-regression>:2:2"} {
		if !strings.Contains(message, site) {
			t.Fatalf("missing caller %s: %s", site, message)
		}
	}
	if strings.Contains(message, "<file>:0:0") {
		t.Fatal(message)
	}
	// Every failure must release runtime/VM state before the next evaluation.
	if RT.vm != nil {
		t.Fatal("active VM leaked")
	}
	if got := Evaluate(parseVMTest(t, "(+ 1 2)")); !got.Equals(Int{I: 3}) {
		t.Fatal(got)
	}
}
