package core

import (
	"os"
	"strings"
	"testing"
)

func TestNativeWalkOwnsCallbackArguments(t *testing.T) {
	var retained [][]Object
	callback := Proc{Fn: func(args []Object) Object {
		retained = append(retained, args)
		return args[0]
	}}
	form := NewArrayVectorFrom(Int{I: 1}, Int{I: 2})
	result := procPostwalk([]Object{callback, form})
	if !result.Equals(form) || len(retained) != 3 ||
		!retained[0][0].Equals(Int{I: 1}) || !retained[1][0].Equals(Int{I: 2}) ||
		!retained[2][0].Equals(form) {
		t.Fatal("native walker reused callback arguments")
	}
}

var walkBenchmarkResult Object

// Run the public walkers through the VM, with interpreted user callbacks.
// Parsing, compilation, fixture construction and warm-up are outside the
// timed loop.
func BenchmarkWalk(b *testing.B) {
	previous := GLOBAL_ENV.CurrentNamespace()
	defer GLOBAL_ENV.SetCurrentNamespace(previous)
	path := "../benchmarks/walk_inputs.joke"
	source, err := os.ReadFile(path)
	if err != nil {
		b.Fatal(err)
	}
	if err := ProcessReader(NewReader(strings.NewReader(string(source)), path), path, EVAL); err != nil {
		b.Fatal(err)
	}
	GLOBAL_ENV.EnsureSymbolIsNamespace(MakeSymbol("joker.walk"))

	compile := func(code string) *Fn {
		b.Helper()
		obj, err := TryRead(NewReader(strings.NewReader(code), "<walk-benchmark>"))
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
		return fn
	}
	for _, tc := range []struct {
		name string
		call string
	}{
		{"walk-vector", "walk identity identity joker.benchmark-walk-inputs/wide-vector"},
		{"postwalk-vector", "postwalk identity joker.benchmark-walk-inputs/wide-vector"},
		{"postwalk-list", "postwalk identity joker.benchmark-walk-inputs/wide-list"},
		{"prewalk-deep", "prewalk identity joker.benchmark-walk-inputs/deep-tree"},
		{"postwalk-mixed", "postwalk identity joker.benchmark-walk-inputs/mixed-tree"},
		{"postwalk-lazy", "postwalk identity (joker.benchmark-walk-inputs/lazy-input)"},
		{"keywordize-keys", "keywordize-keys joker.benchmark-walk-inputs/mixed-tree"},
		{"postwalk-replace", "postwalk-replace joker.benchmark-walk-inputs/replacements joker.benchmark-walk-inputs/mixed-tree"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			fn := compile("(fn [] (joker.walk/" + tc.call + "))")
			vm := NewVM()
			vm.Execute(fn, nil) // Warm bytecode, stack and caches.
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				walkBenchmarkResult = vm.Execute(fn, nil)
			}
		})
	}
}
