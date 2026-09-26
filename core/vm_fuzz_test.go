package core

import (
	"fmt"
	"testing"
)

// Generate only bounded, side-effect-free programs. Arbitrary Joker source is
// unsuitable for fuzzing an interpreter with filesystem/network procedures.
func FuzzVMExpressionParity(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	f.Add([]byte{9, 9, 2, 3, 4, 0, 8, 2, 2})
	f.Fuzz(func(t *testing.T, data []byte) {
		index := 0
		next := func() byte {
			if index >= len(data) {
				return 0
			}
			b := data[index]
			index++
			return b
		}
		var expr func(int) string
		expr = func(depth int) string {
			b := next()
			if depth == 0 || b%10 == 0 {
				return fmt.Sprint(int(b) % 10)
			}
			a, c := expr(depth-1), expr(depth-1)
			switch b % 10 {
			case 1:
				return fmt.Sprintf("(+ %s %s)", a, c)
			case 2:
				return fmt.Sprintf("(if (< %s %s) %s %s)", a, c, a, c)
			case 3:
				return fmt.Sprintf("(let [x %s x (fn [] x)] (+ (x) %s))", a, c)
			case 4:
				return fmt.Sprintf("(let [x %s y %s f (fn ([] x) ([_] y))] (+ (f) (f nil)))", a, c)
			case 5:
				return fmt.Sprintf("(let [g (fn [] %s)] (letfn [(f [] (g)) (g [] %s)] (f)))", a, c)
			case 6:
				return fmt.Sprintf("(first [%s %s])", a, c)
			case 7:
				return fmt.Sprintf("(get {:a %s :b %s} :b)", a, c)
			case 8:
				return fmt.Sprintf("(try (throw (ex-info \"expected\" {})) (catch Error e %s) (finally %s))", a, c)
			default:
				return fmt.Sprintf("(loop [n 3 acc %s] (if (zero? n) acc (recur (dec n) (+ acc %s))))", a, c)
			}
		}
		code := expr(3)
		parsed := parseVMTest(t, code)
		ast, err := TryEval(parsed)
		if err != nil {
			t.Fatal(err)
		}
		proto, err := CompileTopLevel(parsed)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateFunctionProto(proto); err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		vm := NewVM().ExecuteTopLevel(proto)
		if !ast.Equals(vm) {
			t.Fatalf("%s: AST %s, VM %s", code, ast.ToString(true), vm.ToString(true))
		}
		env := NewPackEnv()
		packed := proto.Pack(nil, env)
		h, _ := UnpackHeader(env.Pack(nil), GLOBAL_ENV)
		unpacked, _ := UnpackFunctionProto(packed, h)
		result := NewVM().ExecuteTopLevel(unpacked)
		if !ast.Equals(result) {
			t.Fatalf("%s: packed result differs", code)
		}
	})
}
