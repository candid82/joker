package core

import (
	"fmt"
	"testing"
)

func vectorBuilderObjects(n int) []Object {
	objects := make([]Object, n)
	for i := range objects {
		objects[i] = Int{I: i}
	}
	return objects
}

func TestBulkVectorConstruction(t *testing.T) {
	for _, n := range []int{0, 1, 16, 31, 32, 33, 63, 64, 65, 1024, 1025, 1056, 1057, 32768, 32769, 32800, 32801, 65536} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			objects := vectorBuilderObjects(n)
			v := NewVectorFrom(objects...)
			want := EmptyVector()
			for _, o := range objects {
				want = want.Conjoin(o)
			}
			if v.Count() != n || v.shift != want.shift || !v.Equals(want) || v.Hash() != want.Hash() {
				t.Fatal("bulk construction differs from persistent construction")
			}
			for i := 0; i < n; i++ {
				if !v.At(i).Equals(Int{I: i}) {
					t.Fatalf("incorrect value at index %d", i)
				}
			}
			if !v.Conjoin(NIL).Equals(want.Conjoin(NIL)) {
				t.Fatal("conj after bulk construction failed")
			}
			transient := v.AsTransient().(*TransientVector)
			transient.ConjBang(NIL)
			if !transient.Persistent().Equals(want.Conjoin(NIL)) || !v.Equals(want) {
				t.Fatal("transient update changed the source or produced a wrong result")
			}
			if n == 0 {
				return
			}
			if !v.Pop().(*Vector).Equals(want.Pop()) || !v.Peek().Equals(want.Peek()) {
				t.Fatal("pop/peek after bulk construction failed")
			}
			for _, i := range []int{0, n / 2, n - 1} {
				changed := v.Assoc(Int{I: i}, NIL)
				if !changed.Equals(want.Assoc(Int{I: i}, NIL)) || !v.Equals(want) {
					t.Fatal("persistent assoc changed the source or produced a wrong result")
				}
			}
			objects[0], objects[n-1] = NIL, NIL
			if !v.Equals(want) {
				t.Fatal("bulk vector aliases the input slice")
			}
		})
	}
}

type vectorBuilderSeq interface{ Seq }
type uncountedVectorBuilderSeq struct{ vectorBuilderSeq }

func TestVectorFromUncountedSeq(t *testing.T) {
	for _, n := range []int{0, 4, 33, 1057} {
		want := NewVectorFrom(vectorBuilderObjects(n)...)
		got := NewVectorFromSeq(uncountedVectorBuilderSeq{want.Seq()})
		if !got.Equals(want) || got.GetType() != want.GetType() {
			t.Fatalf("incorrect uncounted seq conversion for size %d", n)
		}
	}
}

func TestPackedVectorBuilder(t *testing.T) {
	meta := EmptyArrayMap().Assoc(MakeKeyword("builder-test"), Boolean{B: true}).(Map)
	for _, n := range []int{0, 4, 16, 17, 33, 1057} {
		for _, array := range []bool{false, true} {
			objects := vectorBuilderObjects(n)
			var source Meta = NewVectorFrom(objects...)
			var want Vec = EmptyVector()
			if array {
				source = NewArrayVectorFrom(objects...)
				want = EmptyArrayVector()
			}
			for _, o := range objects {
				want = want.Conj(o).(Vec)
			}
			env := NewPackEnv()
			packed := packObject(source.WithMeta(meta), nil, env)
			header, _ := UnpackHeader(env.Pack(nil), GLOBAL_ENV)
			got, rest := unpackObject(packed, header)
			if len(rest) != 0 || !got.Equals(want) || got.GetType() != want.GetType() || !got.(Meta).GetMeta().Equals(meta) {
				t.Fatalf("packed vector lost values, type, or metadata (size %d, array=%v)", n, array)
			}
		}
	}
}

func TestExprDumpVectorBuilders(t *testing.T) {
	for _, n := range []int{0, 4, 33, 65} {
		objects := vectorBuilderObjects(n)
		body := make([]Expr, n)
		symbols := make([]Symbol, n)
		for i := range body {
			body[i] = &LiteralExpr{obj: objects[i]}
			symbols[i] = MakeSymbol(fmt.Sprintf("arg%d", i))
		}
		m := EmptyArrayMap()
		addVector(m, body, "items", false)
		_, items := m.Get(MakeKeyword("items"))
		v := items.(*Vector)
		if v.Count() != n {
			t.Fatal("incorrect expression dump count")
		}
		for i := 0; i < n; i++ {
			_, value := v.At(i).(Map).Get(KEYWORDS.object)
			if !value.Equals(objects[i]) {
				t.Fatal("expression dump order changed")
			}
		}
		macro := &MacroCallExpr{name: "test", args: objects}
		_, args := macro.Dump(false).Get(MakeKeyword("args"))
		if !args.Equals(NewVectorFrom(objects...)) || args.GetType() != TYPE.Vector {
			t.Fatal("macro argument dump changed")
		}
		arity := FnArityExpr{args: symbols, body: body}
		_, args = arity.Dump(false).Get(MakeKeyword("args"))
		if args.(*Vector).Count() != n {
			t.Fatal("incorrect arity argument count")
		}
		for i := 0; i < n; i++ {
			if !args.(*Vector).At(i).Equals(symbols[i]) {
				t.Fatal("arity argument order changed")
			}
		}
		fn := &FnExpr{arities: []FnArityExpr{arity}}
		_, arities := fn.Dump(false).Get(MakeKeyword("arities"))
		if arities.(*Vector).Count() != 1 {
			t.Fatal("incorrect function arity dump")
		}
		tryExpr := &TryExpr{catches: []*CatchExpr{{excType: TYPE.String, excSymbol: MakeSymbol("e"), body: body}}}
		_, catches := tryExpr.Dump(false).Get(MakeKeyword("catches"))
		if catches.(*Vector).Count() != 1 {
			t.Fatal("incorrect catch dump")
		}
	}
}

var vectorBuilderBenchmarkResult Object

func BenchmarkVectorBulkConstruction(b *testing.B) {
	for _, n := range []int{64, 1024, 32769} {
		objects := vectorBuilderObjects(n)
		b.Run(fmt.Sprintf("bulk/%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				vectorBuilderBenchmarkResult = NewVectorFrom(objects...)
			}
		})
		b.Run(fmt.Sprintf("previous/%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				tail := make([]interface{}, 32)
				for j := range tail {
					tail[j] = objects[j]
				}
				v := &Vector{count: 32, shift: 5, root: empty_node, tail: tail}
				for _, o := range objects[32:] {
					v = v.Conjoin(o)
				}
				vectorBuilderBenchmarkResult = v
			}
		})
	}
}
