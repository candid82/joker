package core

import (
	"os"
	"strings"
	"testing"
)

var coreFunctionsBenchmarkResult Object

// Exercise public functions and Joker callbacks through the VM. Setup and
// compilation are outside timing; every lazy result is consumed in the body.
func BenchmarkCoreFunctions(b *testing.B) {
	previous := GLOBAL_ENV.CurrentNamespace()
	defer GLOBAL_ENV.SetCurrentNamespace(previous)
	path := "../benchmarks/core_functions_inputs.joke"
	source, err := os.ReadFile(path)
	if err != nil {
		b.Fatal(err)
	}
	if err := ProcessReader(NewReader(strings.NewReader(string(source)), path), path, EVAL); err != nil {
		b.Fatal(err)
	}
	for _, tc := range []struct{ name, body string }{
		{"mapv-small", "(mapv inc small)"},
		{"mapv", "(mapv inc items)"},
		{"mapv-2", "(mapv add items items)"},
		{"mapv-3", "(mapv add3 items items items)"},
		{"mapv-4", "(mapv add4 items items items items)"},
		{"filterv", "(filterv even? items)"},
		{"frequencies", "(frequencies duplicates)"},
		{"zipmap-small", "(zipmap small small)"},
		{"zipmap", "(zipmap items items)"},
		{"select-keys-small", "(select-keys dictionary small)"},
		{"select-keys", "(select-keys dictionary items)"},
		{"range", "(reduce + (range 1024))"},
		{"range-step", "(reduce + (range 0 2048 2))"},
		{"range-infinite", "(reduce + (take 1024 (range)))"},
		{"repeat", "(reduce + (repeat 1024 1))"},
		{"keep", "(reduce + (keep keep-even items))"},
		{"map-indexed", "(reduce + (map-indexed add items))"},
		{"keep-indexed", "(reduce + (keep-indexed keep-even-index items))"},
		{"drop", "(first (drop 512 items))"},
		{"take-while", "(reduce + (take-while below-half items))"},
		{"drop-while", "(first (drop-while below-half items))"},
		{"distinct", "(reduce + (distinct duplicates))"},
		{"map-2", "(reduce + (map add items items))"},
		{"map-3", "(reduce + (map add3 items items items))"},
		{"map-4", "(reduce + (map add4 items items items items))"},
		{"get-in-small", "(get-in small-tree small-path)"},
		{"get-in-default", "(get-in deep-tree deep-path :missing)"},
		{"get-in", "(get-in deep-tree deep-path)"},
		{"assoc-in-small", "(assoc-in small-tree small-path 2)"},
		{"assoc-in", "(assoc-in deep-tree deep-path 2)"},
		{"update-in-small", "(update-in small-tree small-path inc)"},
		{"update-in", "(update-in deep-tree deep-path + 1 2 3 4)"},
		{"pipeline", "(pipeline)"},
		{"nested-hof", "(nested-hof)"},
		{"sum-loop-control", "(sum-loop)"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			obj, err := TryRead(NewReader(strings.NewReader("(fn [] "+tc.body+")"), "<core-benchmark>"))
			if err != nil {
				b.Fatal(err)
			}
			expr, err := TryParse(obj, &ParseContext{GlobalEnv: GLOBAL_ENV})
			if err != nil {
				b.Fatal(err)
			}
			value, err := TryEvaluate(expr)
			if err != nil {
				b.Fatal(err)
			}
			fn := value.(*Fn)
			fn.ensureCompiled()
			vm := NewVM()
			vm.Execute(fn, nil)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				coreFunctionsBenchmarkResult = vm.Execute(fn, nil)
			}
		})
	}
}
