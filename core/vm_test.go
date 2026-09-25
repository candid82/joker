package core

import (
	"bytes"
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
		t.Errorf("expected ArrayVector (like AST literals), got %T", result)
	}
}

func TestVMMapSetLiteralParity(t *testing.T) {
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
			ast, err := TryEval(expr)
			if err != nil {
				t.Fatalf("AST evaluation: %v", err)
			}
			if !IsVMCompatible(expr) {
				t.Fatal("literal should be VM-compatible")
			}
			proto, err := Compile(expr, "<test>")
			if err != nil {
				t.Fatal(err)
			}
			vmResult := NewVM().Execute(&Fn{proto: proto, isCompiled: true}, nil)
			if got := ast.ToString(true); got != tt.expected {
				t.Errorf("AST result: got %s, want %s", got, tt.expected)
			}
			if got := vmResult.ToString(true); got != ast.ToString(true) || vmResult.GetType() != ast.GetType() {
				t.Errorf("VM result %s (%v), AST result %s (%v)", got, vmResult.GetType(), ast.ToString(true), ast.GetType())
			}
		})
	}
}

func TestVMFallbackAttribution(t *testing.T) {
	reader := NewReader(strings.NewReader("(fn target [x] x)"), "<fallback-test>")
	form, err := TryRead(reader)
	if err != nil {
		t.Fatal(err)
	}
	expr, err := TryParse(form, &ParseContext{GlobalEnv: GLOBAL_ENV})
	if err != nil {
		t.Fatal(err)
	}
	stats := &vmFallbackStats{sites: make(map[vmFallbackSite]uint64)}
	vm := NewVM()
	vm.frameCount = 1
	vm.frames[0] = CallFrame{
		closure:    &Fn{proto: &FunctionProto{Name: "caller"}},
		arityProto: &ArityProto{Chunk: &Chunk{Lines: []int{9, 9}}}, ip: 2,
	}
	for i := 0; i < 2*vmFallbackSampleRate; i++ {
		stats.record(&Fn{fnExpr: expr.(*FnExpr)}, vm)
	}
	previous := vmFallbackAttribution.Swap(stats)
	defer vmFallbackAttribution.Store(previous)
	var out bytes.Buffer
	WriteVMFallbackAttribution(&out)
	if !strings.Contains(out.String(), "fallback calls: 512") || !strings.Contains(out.String(), "512  caller:9 -> target <fallback-test>:1") {
		t.Fatalf("unexpected fallback report: %s", out.String())
	}
}

func TestVMFinallyFallsBackToAST(t *testing.T) {
	reader := NewReader(strings.NewReader("(fn [] (try 1 (finally 2)))"), "<test>")
	form, err := TryRead(reader)
	if err != nil {
		t.Fatal(err)
	}
	expr, err := TryParse(form, &ParseContext{GlobalEnv: GLOBAL_ENV})
	if err != nil {
		t.Fatal(err)
	}
	fn, ok := expr.(*FnExpr)
	if !ok {
		t.Fatalf("expected FnExpr, got %T", expr)
	}
	if IsVMCompatibleFn(fn) {
		t.Fatal("try/finally is not yet safe to compile")
	}
}

// evalAndCompile parses, compiles, and executes code using the VM
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
		t.Skipf("expression not VM compatible: %s", code)
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
