package core

import "testing"

func TestFirstCollectionSemantics(t *testing.T) {
	located := Int{I: 42}.WithInfo(&ObjectInfo{})
	for _, input := range []Object{
		NIL, EmptyArrayVector(), EmptyVector(), MakeString(""),
		NewArrayVectorFrom(located), NewVectorFrom(located),
		MakeString("abc"), MakeString("λx"), MakeString("\xffx"),
	} {
		want := input.(Seqable).Seq().First()
		got := procFirst([]Object{input})
		if !got.Equals(want) || got.GetInfo() != want.GetInfo() {
			t.Fatalf("first changed for %T: got %v, want %v", input, got, want)
		}
	}
	calls := 0
	lazy := NewMapSeq(Proc{Fn: func(args []Object) Object {
		calls++
		return args[0]
	}}, NewArrayVectorFrom(located, NIL))
	if procFirst([]Object{lazy}) != located || calls != 1 {
		t.Fatal("first realized past the first element")
	}
	expectJokerPanic(t, func() {
		Evaluate(&CallExpr{
			callable: &LiteralExpr{obj: Proc{Fn: procFirst}},
			args:     []Expr{&LiteralExpr{obj: EmptyArrayVector().AsTransient()}},
		})
	})
}

func TestToSliceCountedCollections(t *testing.T) {
	values := make([]Object, 80)
	for i := range values {
		values[i] = Int{I: i}
	}
	inputs := []Seq{EmptyList, MakeString("aλ").Seq(), NewListFrom(values[:3]...)}
	for _, start := range []int{0, 1, 31, 32, 79, 80, 81} {
		for _, step := range []int{0, 1, 2} {
			inputs = append(inputs, &ArraySeq{arr: values, index: start, step: step})
		}
		inputs = append(inputs, &VectorSeq{vector: NewVectorFrom(values...), index: start})
		inputs = append(inputs, &VectorSeq{vector: NewArrayVectorFrom(values...), index: start})
	}
	for _, input := range inputs {
		want := []Object{}
		for seq := input; !seq.IsEmpty(); seq = seq.Rest() {
			want = append(want, seq.First())
		}
		got := ToSlice(input)
		if got == nil || len(got) != len(want) {
			t.Fatalf("incorrect result length for %T", input)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("incorrect element %d for %T", i, input)
			}
		}
		if len(got) > 0 {
			first := input.First()
			got[0] = NIL
			if input.First() != first {
				t.Fatalf("result aliases %T backing storage", input)
			}
		}
	}
}
