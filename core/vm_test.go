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
	if v, ok := result.(Counted); ok {
		if v.Count() != 3 {
			t.Errorf("expected count 3, got %d", v.Count())
		}
	} else {
		t.Errorf("expected Counted, got %T", result)
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
