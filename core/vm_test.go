package core

import (
	"strings"
	"testing"
)

func TestVMBasicArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{"add", "(+ 1 2)", 3},
		{"subtract", "(- 5 3)", 2},
		{"multiply", "(* 3 4)", 12},
		{"divide", "(/ 10 2)", 5},
		{"nested", "(+ (* 2 3) (- 10 4))", 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalAndCompile(t, tt.code)
			if i, ok := result.(Int); ok {
				if i.I != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, i.I)
				}
			} else {
				t.Errorf("expected Int, got %T", result)
			}
		})
	}
}

func TestVMConditionals(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{"if-true", "(if true 1 2)", 1},
		{"if-false", "(if false 1 2)", 2},
		{"if-nil", "(if nil 1 2)", 2},
		{"nested-if", "(if (< 1 2) (if (> 3 2) 10 20) 30)", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalAndCompile(t, tt.code)
			if i, ok := result.(Int); ok {
				if i.I != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, i.I)
				}
			} else {
				t.Errorf("expected Int, got %T", result)
			}
		})
	}
}

func TestVMLetBindings(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{"simple-let", "(let [x 5] x)", 5},
		{"let-with-expr", "(let [x (+ 1 2)] x)", 3},
		{"multiple-bindings", "(let [x 1 y 2] (+ x y))", 3},
		{"nested-let", "(let [x 1] (let [y 2] (+ x y)))", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalAndCompile(t, tt.code)
			if i, ok := result.(Int); ok {
				if i.I != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, i.I)
				}
			} else {
				t.Errorf("expected Int, got %T", result)
			}
		})
	}
}

func TestVMLoopRecur(t *testing.T) {
	code := `(loop [n 10 acc 0]
              (if (= n 0)
                acc
                (recur (- n 1) (+ acc n))))`

	result := evalAndCompile(t, code)
	if i, ok := result.(Int); ok {
		if i.I != 55 {
			t.Errorf("expected 55, got %d", i.I)
		}
	} else {
		t.Errorf("expected Int, got %T", result)
	}
}

func TestVMVectors(t *testing.T) {
	code := "[1 2 3]"
	result := evalAndCompile(t, code)
	if v, ok := result.(*ArrayVector); ok {
		if v.Count() != 3 {
			t.Errorf("expected count 3, got %d", v.Count())
		}
	} else {
		t.Errorf("expected ArrayVector, got %T", result)
	}
}

func TestVMMapSetLiterals(t *testing.T) {
	cases := []struct {
		name      string
		code      string
		threshold int64
		expected  string
	}{
		{"empty map", "{}", 16, "{}"},
		{"empty set", "#{}", 16, "#{}"},
		{"array map order", `(let [t (atom []) m {(do (swap! t conj :k1) :a) (do (swap! t conj :v1) 1)
			(do (swap! t conj :k2) :b) (do (swap! t conj :v2) 2)}] [m @t])`, 16, "[{:a 1, :b 2} [:k1 :v1 :k2 :v2]]"},
		{"hash map order", `(let [t (atom []) m {(do (swap! t conj :k1) :a) (do (swap! t conj :v1) 1)
			(do (swap! t conj :k2) :b) (do (swap! t conj :v2) 2)}] [(type m) @t])`, 2, "[HashMap [:k1 :v1 :k2 :v2]]"},
		{"array map duplicate", `(let [t (atom [])] (try
			{(do (swap! t conj :k1) :a) (do (swap! t conj :v1) 1)
			 (do (swap! t conj :k2) :a) (do (swap! t conj :v2) 2)
			 (do (swap! t conj :k3) :b) (do (swap! t conj :v3) 3)}
			(catch Error e [@t (ex-message e)])))`, 16, "[[:k1 :v1 :k2 :v2] \"Duplicate key: :a\"]"},
		{"hash map duplicate", `(let [t (atom [])] (try
			{(do (swap! t conj :k1) :a) (do (swap! t conj :v1) 1)
			 (do (swap! t conj :k2) :a) (do (swap! t conj :v2) 2)}
			(catch Error e [@t (ex-message e)])))`, 2, "[[:k1 :v1 :k2] \"Duplicate key: :a\"]"},
		{"set order", `(let [t (atom []) s #{(do (swap! t conj 1) 1) (do (swap! t conj 2) 2)}] [s @t])`, 16, "[#{1 2} [1 2]]"},
		{"set duplicate", `(let [t (atom [])] (try
			#{(do (swap! t conj 1) 1) (do (swap! t conj 2) 1) (do (swap! t conj 3) 3)}
			(catch Error e [@t (ex-message e)])))`, 16, "[[1 2] \"Duplicate set element: 1\"]"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewReader(strings.NewReader(tt.code), "<test>")
			form, err := TryRead(reader)
			if err != nil {
				t.Fatal(err)
			}
			oldThreshold := HASHMAP_THRESHOLD
			defer func() { HASHMAP_THRESHOLD = oldThreshold }()
			// Read with the default threshold; change it only for evaluation.
			HASHMAP_THRESHOLD = tt.threshold
			ctx := &ParseContext{GlobalEnv: GLOBAL_ENV}
			expr, err := TryParse(form, ctx)
			if err != nil {
				t.Fatal(err)
			}
			if !IsVMCompatible(expr) {
				t.Fatal("literal should be VM-compatible")
			}
			proto, err := Compile(expr, "<test>")
			if err != nil {
				t.Fatal(err)
			}
			vmResult := NewVM().Execute(&Fn{proto: proto, isCompiled: true}, nil)
			if got := vmResult.ToString(true); got != tt.expected {
				t.Errorf("got %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestVMPackedCollectionLiterals(t *testing.T) {
	cases := []struct {
		name, code string
	}{
		{"quoted empty list", `(fn [] '())`},
		{"quoted list", `(fn [] '(1 "text" :key))`},
		{"quoted set", `(fn [] '#{0 1 2})`},
		{"quoted empty set", `(fn [] '#{})`},
		{"quoted hash set", `(fn [] '#{0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16})`},
		{"nested quoted collection", `(fn [] '(#{:a 3} (4 5)))`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			form, err := TryRead(NewReader(strings.NewReader(tt.code), "<collection-test>"))
			if err != nil {
				t.Fatal(err)
			}
			expr, err := TryParse(form, &ParseContext{GlobalEnv: GLOBAL_ENV})
			if err != nil {
				t.Fatal(err)
			}
			fnExpr := expr.(*FnExpr)
			if !IsVMCompatibleFn(fnExpr) {
				t.Fatal("quoted collection should be VM compatible")
			}
			proto, err := CompileFnExpr(fnExpr, nil)
			if err != nil {
				t.Fatal(err)
			}
			compiled := &Fn{proto: proto, isCompiled: true}
			vmA, vmB := VMExecute(compiled, nil), VMExecute(compiled, nil)
			if vmA != vmB {
				t.Errorf("VM constant identity changed: %s", vmA)
			}

			env := NewPackEnv()
			packed := proto.Pack(nil, env)
			header, _ := UnpackHeader(env.Pack(nil), GLOBAL_ENV)
			unpacked, rest := UnpackFunctionProto(packed, header)
			if len(rest) != 0 {
				t.Fatalf("%d bytes remained after unpacking", len(rest))
			}
			packedFn := &Fn{proto: unpacked, isCompiled: true}
			packedA, packedB := VMExecute(packedFn, nil), VMExecute(packedFn, nil)
			if packedA != packedB || packedA.GetType() != vmA.GetType() || !packedA.Equals(vmA) || packedA.ToString(true) != vmA.ToString(true) {
				t.Errorf("packed VM %s (%T), direct VM %s (%T)", packedA, packedA, vmA, vmA)
			}
		})
	}
	meta := EmptyArrayMap().Assoc(MakeKeyword("tag"), Int{I: 1}).(Map)
	for _, obj := range []Object{
		NewListFrom(Int{I: 1}).WithMeta(meta),
		NewList(Int{I: 1}, NewListFrom(Int{I: 2}).WithMeta(meta).(*List)),
		EmptySet().WithMeta(meta),
		NewListFrom(TYPE.Fn), // Type objects need the packer's special encoding.
		EmptySet().Conj(EmptyArrayVector()).(Object),
	} {
		proto, err := Compile(&LiteralExpr{obj: obj}, "constant")
		if err != nil {
			t.Fatal(err)
		}
		if result := NewVM().ExecuteTopLevel(proto); result != obj {
			t.Fatal("runtime constant identity changed")
		}
		env := NewPackEnv()
		packed := proto.Pack(nil, env)
		header, _ := UnpackHeader(env.Pack(nil), GLOBAL_ENV)
		unpacked, _ := UnpackFunctionProto(packed, header)
		result := NewVM().ExecuteTopLevel(unpacked)
		if !result.Equals(obj) || result.GetType() != obj.GetType() {
			t.Fatalf("packed constant changed: %s", obj)
		}
	}
}

func TestVMVarLiteralIdentity(t *testing.T) {
	code := `(fn [] (var *out*))`
	form, err := TryRead(NewReader(strings.NewReader(code), "<var-test>"))
	if err != nil {
		t.Fatal(err)
	}
	expr, err := TryParse(form, &ParseContext{GlobalEnv: GLOBAL_ENV})
	if err != nil {
		t.Fatal(err)
	}
	fnExpr := expr.(*FnExpr)
	if !IsVMCompatibleFn(fnExpr) {
		t.Fatal("Var literals should be VM compatible")
	}
	want := GLOBAL_ENV.CoreNamespace.Resolve("*out*")
	proto, err := CompileFnExpr(fnExpr, nil)
	if err != nil {
		t.Fatal(err)
	}
	vmFn := &Fn{proto: proto, isCompiled: true}
	if vm := VMExecute(vmFn, nil); vm != want || VMExecute(vmFn, nil) != vm {
		t.Fatal("Var identity differs between calls")
	}
	env := NewPackEnv()
	packed := proto.Pack(nil, env)
	header, _ := UnpackHeader(env.Pack(nil), GLOBAL_ENV)
	unpacked, rest := UnpackFunctionProto(packed, header)
	if len(rest) != 0 || VMExecute(&Fn{proto: unpacked, isCompiled: true}, nil) != want {
		t.Fatal("Var identity differs after packed-bytecode round trip")
	}
}

func TestVMCaseExpansion(t *testing.T) {
	fn := evalAndCompile(t, `(fn [floor] (case floor 0 [1] 1 [0 2] 2 [1 3] 3 [2]))`).(*Fn)
	for floor, want := range []string{"[1]", "[0 2]", "[1 3]", "[2]"} {
		if got := fn.Call([]Object{Int{I: floor}}).ToString(true); got != want {
			t.Errorf("floor %d: got %s, want %s", floor, got, want)
		}
	}
}

func TestVMClosedEnvironment(t *testing.T) {
	for _, tt := range []struct {
		code string
		want int
	}{
		{`(let [x 5] (fn [y] (+ x y)))`, 7},
		{`(let [x 5] (fn [y] ((fn [] (+ x y)))))`, 7},
		{`(let [x 5] (fn [x] x))`, 2},
	} {
		fn := evalAndCompile(t, tt.code).(*Fn)
		if got := fn.Call([]Object{Int{I: 2}}); !got.Equals(Int{I: tt.want}) {
			t.Errorf("%s: got %s, want %d", tt.code, got, tt.want)
		}
	}
}

func TestVMNamedFunctionAndArityRecur(t *testing.T) {
	cases := []struct {
		code string
		args []Object
		want int
	}{
		{`(fn step [n] (if (= n 0) 0 (step (- n 1))))`, []Object{Int{I: 7}}, 0},
		{`(fn step ([n] (step n 0)) ([n acc] (if (= n 0) acc (recur (- n 1) (+ acc n)))))`, []Object{Int{I: 1000}}, 500500},
		{`(fn step [n] (if (= n 0) 0 ((fn [] (step (- n 1))))))`, []Object{Int{I: 4}}, 0},
		{`(fn step [n & xs] (if (seq xs) (recur (+ n (first xs)) (next xs)) n))`, []Object{Int{I: 1}, Int{I: 2}, Int{I: 3}}, 6},
		{`(fn step [n] (loop [i n] (if (= i 0) n (recur (- i 1)))))`, []Object{Int{I: 4}}, 4},
	}
	for _, tt := range cases {
		t.Run(tt.code, func(t *testing.T) {
			reader := NewReader(strings.NewReader(tt.code), "<test>")
			form, err := TryRead(reader)
			if err != nil {
				t.Fatal(err)
			}
			expr, err := TryParse(form, &ParseContext{GlobalEnv: GLOBAL_ENV})
			if err != nil {
				t.Fatal(err)
			}
			fnExpr := expr.(*FnExpr)
			proto, err := CompileFnExpr(fnExpr, nil)
			if err != nil {
				t.Fatal(err)
			}
			actual := VMExecute(&Fn{proto: proto, isCompiled: true}, tt.args)
			if !actual.Equals(Int{I: tt.want}) {
				t.Errorf("got %s, want %d", actual, tt.want)
			}
		})
	}
}

func TestVMTypeLiteralInClosedFunction(t *testing.T) {
	fn := evalAndCompile(t, `(fn [coll] (instance? Reduce coll))`).(*Fn)
	for _, tt := range []struct {
		coll Object
		want bool
	}{
		{NewListFrom(Int{I: 1}), false},
		{EmptyArrayVector(), true},
		{NIL, false},
	} {
		if got := fn.Call([]Object{tt.coll}); !got.Equals(Boolean{B: tt.want}) {
			t.Errorf("instance? Reduce %s: got %s, want %t", tt.coll, got, tt.want)
		}
	}
}

func TestVMPackedSourcePositions(t *testing.T) {
	filename := "packed.joke"
	pos := Position{filename: &filename, startLine: 7, startColumn: 3}
	chunk := NewChunk()
	chunk.appendAt(byte(OP_CALL), pos)
	chunk.appendAt(0, pos)
	chunk.callSites[0] = &CallExpr{Position: pos, callName: "packed-call"}
	env := NewPackEnv()
	packed := chunk.Pack(nil, env)
	header, _ := UnpackHeader(env.Pack(nil), GLOBAL_ENV)
	unpacked, remaining := unpackChunk(packed, header)
	if len(remaining) != 0 || len(unpacked.Positions) != 2 || unpacked.positionAt(0).Filename() != filename ||
		unpacked.positionAt(0).startLine != 7 || unpacked.positionAt(1).startColumn != 3 ||
		unpacked.callSites[0] == nil || unpacked.callSites[0].Pos().Filename() != filename ||
		unpacked.callSites[0].callName != "packed-call" || unpacked.callSiteAt(1) != nil {
		t.Fatalf("lost bytecode source positions or native call site during packing")
	}
}

func TestVMCallSiteAbsent(t *testing.T) {
	for _, chunk := range []*Chunk{NewChunk(), {Code: []byte{byte(OP_RETURN)}}} {
		for _, ip := range []int{-1, 0, 1} {
			if chunk.callSiteAt(ip) != nil {
				t.Fatal("unexpected call descriptor")
			}
		}
	}
}

func TestVMSourcePositions(t *testing.T) {
	code := "(let [f (fn []\n  (nth [] 4))] (f))"
	reader := NewReader(strings.NewReader(code), "<vm-position>")
	form, err := TryRead(reader)
	if err != nil {
		t.Fatal(err)
	}
	expr, err := TryParse(form, &ParseContext{GlobalEnv: GLOBAL_ENV})
	if err != nil {
		t.Fatal(err)
	}
	proto, err := Compile(expr, "<vm-position>")
	if err != nil {
		t.Fatal(err)
	}
	var vmErr Error
	func() {
		defer func() {
			if r := recover(); r != nil {
				vmErr = r.(Error)
			}
		}()
		VMExecute(&Fn{proto: proto, isCompiled: true}, nil)
	}()
	if vmErr == nil {
		t.Fatal("expected VM error")
	}
	if got := vmErr.Error(); !strings.Contains(got, "<vm-position>:2:3") || strings.Contains(got, "<file>:0:0") {
		t.Errorf("VM error lacks the failing source location: %s", got)
	}
}

func TestVMFinally(t *testing.T) {
	for _, tt := range []struct{ code, want string }{
		{`(let [events (atom [])] [(try (swap! events conj :body) :value (finally (swap! events conj :finally))) @events])`, `[:value [:body :finally]]`},
		{`(let [events (atom [])] [(try (throw (ex-info "body" {})) (catch Error e (swap! events conj :caught) :value) (finally (swap! events conj :finally))) @events])`, `[:value [:caught :finally]]`},
		{`(try (try (throw (ex-info "body" {})) (finally (throw (ex-info "finally" {})))) (catch Error e (ex-message e)))`, `"finally"`},
	} {
		if got := evalAndCompile(t, tt.code).ToString(true); got != tt.want {
			t.Errorf("got %s, want %s", got, tt.want)
		}
	}
}

func evalAndCompile(t *testing.T, code string) Object {
	t.Helper()

	// Parse the code using the standard reader
	reader := NewReader(strings.NewReader(code), "<test>")
	obj, err := TryRead(reader)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	// Parse to AST
	ctx := &ParseContext{GlobalEnv: GLOBAL_ENV}
	expr, err := TryParse(obj, ctx)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Check if VM compatible
	if !IsVMCompatible(expr) {
		t.Fatalf("expression not VM compatible: %s", code)
	}

	// Compile to bytecode
	proto, compileErr := Compile(expr, "<test>")
	if compileErr != nil {
		t.Fatalf("compile error: %v", compileErr)
	}

	// Create a function and execute
	fn := &Fn{
		proto:      proto,
		isCompiled: true,
	}

	vm := NewVM()
	return vm.Execute(fn, nil)
}

func TestVMNativeCallArguments(t *testing.T) {
	vm := NewVM()
	coreProc := Proc{Fn: procConcatSeq, Name: "procConcatSeq"}
	vm.Push(coreProc)
	vm.Push(NewVectorFrom(Int{I: 1}, Int{I: 2}))
	vm.Push(NewVectorFrom(Int{I: 3}))
	if vm.callValue(coreProc, 2) {
		t.Fatal("native call pushed a frame")
	}
	result := vm.Pop().(Seq)
	vm.Push(Int{I: 99}) // Reuse the call's stack slot after returning.
	if got := SeqToString(result, false); got != "(1 2 3)" {
		t.Fatalf("lazy native result changed after stack reuse: %s", got)
	}

	// Non-core procs can retain their arguments; they must receive an
	// independently owned slice rather than the VM stack view.
	retainingProc := Proc{Fn: func(args []Object) Object { return &ArraySeq{arr: args} }, Name: "retain", Package: "test"}
	vm.Reset()
	vm.Push(retainingProc)
	vm.Push(Int{I: 7})
	vm.callValue(retainingProc, 1)
	retained := vm.Pop().(Seq)
	vm.Push(Int{I: 8})
	if got := retained.First().(Int).I; got != 7 {
		t.Fatalf("retained argument changed after stack reuse: %d", got)
	}
}

func TestDisassemble(t *testing.T) {
	code := "(+ 1 2)"

	reader := NewReader(strings.NewReader(code), "<test>")
	obj, err := TryRead(reader)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	ctx := &ParseContext{GlobalEnv: GLOBAL_ENV}
	expr, err := TryParse(obj, ctx)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	proto, compileErr := Compile(expr, "<test>")
	if compileErr != nil {
		t.Fatalf("compile error: %v", compileErr)
	}

	disasm := DisassembleChunk(proto.Chunk, "<test>")
	t.Logf("Disassembly:\n%s", disasm)

	// Verify we have some bytecode
	if len(proto.Chunk.Code) == 0 {
		t.Error("expected non-empty bytecode")
	}
}
