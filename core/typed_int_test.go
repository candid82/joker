package core

import (
	"fmt"
	"strings"
	"testing"
)

func TestTypedIntegerLoopCompilation(t *testing.T) {
	fn := evalAndCompile(t, `(fn [] (loop [n 0] (if (= n 10000) n (recur (inc n)))))`).(*Fn)
	code := DisassembleChunk(fn.proto.Arities[0].Chunk, "counter")
	for _, op := range []string{"CALL_INT_INC", "CALL_INT_EQ"} {
		if !strings.Contains(code, op) {
			t.Errorf("missing %s in:\n%s", op, code)
		}
	}
	if err := ValidateFunctionProto(fn.proto); err != nil {
		t.Fatal(err)
	}
	env := NewPackEnv()
	packed := fn.proto.Pack(nil, env)
	header, _ := UnpackHeader(env.Pack(nil), GLOBAL_ENV)
	proto, rest := UnpackFunctionProto(packed, header)
	if len(rest) != 0 || !VMExecute(&Fn{proto: proto, isCompiled: true}, nil).Equals(Int{I: 10000}) {
		t.Fatal("typed integer loop changed after packing")
	}
}

func TestTypedIntegerLoopAllocationScaling(t *testing.T) {
	allocs := make([]float64, 2)
	for i, limit := range []int{100, 10000} {
		fn := evalAndCompile(t, fmt.Sprintf(`(fn [] (loop [n 0] (if (= n %d) n (recur (inc n)))))`, limit)).(*Fn)
		allocs[i] = testing.AllocsPerRun(3, func() { integerCacheResult = fn.Call(nil) })
		if !integerCacheResult.Equals(Int{I: limit}) {
			t.Fatal("incorrect counter result")
		}
	}
	if allocs[1] > allocs[0]+2 {
		t.Fatalf("integer loop allocations scale with iterations: %.0f vs %.0f", allocs[0], allocs[1])
	}
}

func TestTypedIntegerLoopSemantics(t *testing.T) {
	for _, tt := range []struct {
		code string
		want string
	}{
		{`(loop [n 0] (if (= n 10000) n (recur (inc n))))`, `10000`},
		{`(loop [n 0 out []] (if (= n 3) out (recur (inc n) (conj out n))))`, `[0 1 2]`},
		{`(loop [n 0 out []] (if (= n 3) (mapv #(%) out) (recur (inc n) (conj out (fn [] n)))))`, `[0 1 2]`},
		{`(loop [n 2048 out []] (if (= n 2051) out (recur (inc n) (conj out [(inc n) (inc n)]))))`, `[[2049 2049] [2050 2050] [2051 2051]]`},
		{`(loop [n 0] (if (= n 3) (let [n 0] (loop [m n] (if (= m 2) m (recur (inc m))))) (recur (inc n))))`, `2`},
		{`(loop [n 0] (if (= n 3) n (let [next (inc n)] (recur next))))`, `3`},
		{`(loop [n 0] (if (= n 3) n (recur ((fn [x] (inc x)) n))))`, `3`},
		{`(loop [n 0] (if (>= n 7) n (recur (if (= n 4) 4.5 (inc n)))))`, `7.5`},
		{`(with-redefs [inc (fn [x] (+ x 0.5))] (loop [n 0] (if (>= n 3) [(= n 3) n] (recur (inc n)))))`, `[false 3.0]`},
		{`(with-redefs [= (fn [x y] (> x y))] (loop [n 0] (if (= n 3) n (recur (inc n)))))`, `4`},
		{`(with-redefs [joker.core/inc__ (fn [x] (+ x 2))] (loop [n 0] (if (= n 6) n (recur (inc n)))))`, `6`},
		{`(with-redefs [joker.core/=__ (fn [x y] (> x y))] (loop [n 0] (if (= n 3) n (recur (inc n)))))`, `4`},
		{`(with-redefs [inc inc] (loop [n 0] (if (= n 1) n
		    (recur (inc (do (var-set #'inc (fn [x] 99)) n))))))`, `1`},
		{`(with-redefs [joker.core/inc__ joker.core/inc__] (loop [n 0] (if (= n 99) n
		    (recur (inc (do (var-set #'joker.core/inc__ (fn [x] 99)) n))))))`, `99`},
		{`(let [calls (atom 0)] (with-redefs [inc 17]
		    (try (loop [n 0] (recur (inc (do (swap! calls (fn [x] (+ x 1))) n))))
		      (catch Error e @calls))))`, `0`},
		{`(let [inc (fn [x] (+ x 2))] (loop [n 0] (if (= n 6) n (recur (inc n)))))`, `6`},
		{`(loop [n 0N] (if (= n 3) [(type n) n] (recur (inc n))))`, `[BigInt 3N]`},
		{`(loop [n 0.0] (if (= n 3.0) [(type n) n] (recur (inc n))))`, `[Double 3.0]`},
		{`(loop [n 0] (if (= n 3) n (do (try (throw (ex-info "x" {})) (catch Error e (inc n))) (recur (inc n)))))`, `3`},
		{`(loop [n 0] (if (= n 3) n (do (try (inc n) (finally nil)) (recur (inc n)))))`, `3`},
		{`(loop [n 0] (if (= n 3) [(apply inc [n]) (reduce + [n 1])] (recur (inc n))))`, `[4 4]`},
	} {
		t.Run(tt.code, func(t *testing.T) {
			if got := evalAndCompile(t, tt.code).ToString(true); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTypedIntegerStorageMaterialization(t *testing.T) {
	vm := NewVM()
	info := &ObjectInfo{}
	original := Int{I: 1, Original: "01"}.WithInfo(info)
	vm.Push(original)
	vm.pushSlot(0)
	if got := vm.Pop(); got.GetInfo() != info || got.(Int).Original != "01" {
		t.Fatal("annotated integer lost information")
	}
	vm.pushInteger(10000)
	vm.pushSlot(1)
	if got := vm.Pop(); got.(Int).I != 10000 || got.GetInfo() != nil {
		t.Fatal("raw result did not materialize as a plain Int")
	}
	vm.Reset()
	for _, value := range vm.integers {
		if value != 0 {
			t.Fatal("reset retained integer payload")
		}
	}
	for _, value := range vm.stack {
		if value != nil {
			t.Fatal("reset retained object")
		}
	}
}

func TestTypedIntegerNativeReentry(t *testing.T) {
	fn := evalAndCompile(t, `(fn [sink] (loop [n 0] (if (= n 3000) (sink n) (recur (inc n)))))`).(*Fn)
	callback := evalAndCompile(t, `(fn [] (let [v [`+strings.Repeat("0 ", 600)+`]]
	    (loop [n 0] (if (= n 3000) n (recur (inc n))))))`).(*Fn)
	for _, independent := range []bool{false, true} {
		var retained []Object
		proc := Proc{Fn: func(args []Object) Object {
			before := args[0]
			retained = append([]Object(nil), args...)
			var result Object
			if independent {
				suspended := RT.Suspend()
				RT.LockIndependent()
				result = CallIndependent(callback, nil)
				RT.GIL.Unlock()
				suspended.Resume()
			} else {
				result = callback.Call(nil)
			}
			if !result.Equals(Int{I: 3000}) || !args[0].Equals(before) {
				t.Fatal("callback/growth overwrote native arguments")
			}
			return result
		}}
		func() {
			RT.GIL.Lock()
			defer RT.GIL.Unlock()
			if !fn.Call([]Object{proc}).Equals(Int{I: 3000}) {
				t.Fatal("incorrect reentry result")
			}
		}()
		if _, ok := retained[0].(Int); !ok || !retained[0].Equals(Int{I: 3000}) {
			t.Fatal("native code retained an internal marker")
		}
	}
}

func TestTypedIntegerPrivateProcedureGuards(t *testing.T) {
	variable := GLOBAL_ENV.CoreNamespace.Resolve("inc__")
	original := variable.Value
	defer func() { variable.Value = original }()
	fn := evalAndCompile(t, `(fn [] (loop [n 0] (if (= n 6) n (recur (inc n)))))`).(*Fn)
	proc := original.(Proc)
	proc.InExecution = func(_ *Execution, args []Object) Object { return Int{I: args[0].(Int).I + 2} }
	variable.Value = proc
	if !fn.Call(nil).Equals(Int{I: 6}) {
		t.Fatal("bypassed InExecution replacement")
	}
	proc.InExecution = nil
	proc.Fn = func(args []Object) Object { return Int{I: args[0].(Int).I + 2} }
	variable.Value = proc
	if !fn.Call(nil).Equals(Int{I: 6}) {
		t.Fatal("trusted spoofed Proc name")
	}
}

func TestTypedIntegerConservativeJoins(t *testing.T) {
	for _, code := range []string{
		`(fn [] (loop [n 0] (if (>= n 7) n (recur (if (= n 4) 4.5 (inc n))))))`,
		`(fn [] (loop [n 0N] (if (= n 3) n (recur (inc n)))))`,
	} {
		fn := evalAndCompile(t, code).(*Fn)
		if strings.Contains(DisassembleChunk(fn.proto.Arities[0].Chunk, "mixed"), "CALL_INT_") {
			t.Fatal("mixed/BigInt loop incorrectly classified as machine Int")
		}
	}
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	code := fmt.Sprintf(`(loop [n %d] (if (= n %d) n (recur (inc n))))`, maxInt, minInt)
	if !evalAndCompile(t, code).Equals(Int{I: minInt}) {
		t.Fatal("integer overflow behavior changed")
	}
}

func FuzzTypedIntegerLoop(f *testing.F) {
	f.Add(uint8(20), uint8(0))
	f.Add(uint8(10), uint8(1))
	f.Add(uint8(3), uint8(2))
	f.Fuzz(func(t *testing.T, limit, kind uint8) {
		RT.GIL.Lock()
		defer RT.GIL.Unlock()
		body := "n"
		switch kind % 3 {
		case 1:
			body = "[n (inc n)]"
		case 2:
			body = "((fn [x] x) n)"
		}
		code := fmt.Sprintf(`(loop [n 0] (if (= n %d) %s (recur (inc n))))`, limit%40, body)
		reader := NewReader(strings.NewReader(code), "<fuzz>")
		form, err := TryRead(reader)
		if err != nil {
			t.Fatal(err)
		}
		expr, err := TryParse(form, &ParseContext{GlobalEnv: GLOBAL_ENV})
		if err != nil {
			t.Fatal(err)
		}
		var values [2]Object
		for i, optimized := range []bool{false, true} {
			c := NewCompiler(nil, "<fuzz>")
			c.integerOptimization = optimized
			if err := c.compile(expr); err != nil {
				t.Fatal(err)
			}
			c.emitOp(OP_RETURN)
			if err := ValidateFunctionProto(c.function); err != nil {
				t.Fatal(err)
			}
			values[i] = VMExecute(&Fn{proto: c.function, isCompiled: true}, nil)
		}
		if !values[0].Equals(values[1]) {
			t.Fatalf("generic/typed results differ for %s", code)
		}
	})
}

func BenchmarkTypedIntegerLoop(b *testing.B) {
	reader := NewReader(strings.NewReader(`(fn [] (loop [n 0] (if (= n 10000) n (recur (inc n)))))`), "<bench>")
	form, err := TryRead(reader)
	if err != nil {
		b.Fatal(err)
	}
	expr, err := TryParse(form, &ParseContext{GlobalEnv: GLOBAL_ENV})
	if err != nil {
		b.Fatal(err)
	}
	proto, err := Compile(expr, "<bench>")
	if err != nil {
		b.Fatal(err)
	}
	fn := VMExecute(&Fn{proto: proto, isCompiled: true}, nil).(*Fn)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		integerCacheResult = fn.Call(nil)
	}
}
