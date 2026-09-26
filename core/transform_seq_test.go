package core

import "testing"

func TestTransformSeqMemoization(t *testing.T) {
	for _, kind := range []transformSeqKind{transformMap, transformFilter, transformMapcat, transformConcat} {
		calls := 0
		source := NewArrayVectorFrom(NIL, Int{I: 1}, Int{I: 2})
		fn := Proc{Fn: func(args []Object) Object {
			calls++
			switch kind {
			case transformFilter:
				return Boolean{B: true}
			case transformMapcat:
				return NewArrayVectorFrom(args[0])
			default:
				return args[0]
			}
		}}
		seq := &TransformSeq{kind: kind, fn: fn, source: source, keep: true}
		if kind == transformConcat {
			seq.source = NewArrayVectorFrom(EmptyList, source, EmptyList)
		}
		if seq.IsRealized() || calls != 0 {
			t.Fatal("sequence realized eagerly")
		}
		if seq.IsEmpty() || seq.First() != NIL || !seq.IsRealized() {
			t.Fatal("nil element mistaken for empty sequence")
		}
		if kind != transformConcat && calls != 1 {
			t.Fatal("realized past the first element")
		}
		for pass := 0; pass < 2; pass++ {
			values := ToSlice(seq)
			if len(values) != 3 || values[0] != NIL || !values[2].Equals(Int{I: 2}) {
				t.Fatalf("incorrect repeated traversal: %v", values)
			}
		}
		if kind != transformConcat && calls != 3 {
			t.Fatalf("callback results not memoized: %d calls", calls)
		}
	}
}

func TestFilterCursorRetainsIndexedTails(t *testing.T) {
	values := []Object{Int{I: 0}, Int{I: 1}, Int{I: 2}, Int{I: 3}, Int{I: 4}}
	for _, input := range []Seq{
		&ArraySeq{arr: values, index: 1, step: 2},
		&VectorSeq{vector: NewArrayVectorFrom(values...), index: 1},
		MakeString("aλβ").Seq(),
	} {
		var expected []Object
		for s := input; !s.IsEmpty(); s = s.Rest() {
			expected = append(expected, s.First())
		}
		calls := 0
		seq := NewFilterSeq(Proc{Fn: func(args []Object) Object {
			calls++
			return Boolean{B: calls%2 == 0}
		}}, input, true)
		if seq.IsEmpty() || !seq.First().Equals(expected[1]) || calls != 2 {
			t.Fatalf("incorrect first match for %T", input)
		}
		firstRest := seq.Rest()
		var want []Object
		for i := 1; i < len(expected); i += 2 {
			want = append(want, expected[i])
		}
		for pass := 0; pass < 2; pass++ {
			got := ToSlice(seq)
			if len(got) != len(want) {
				t.Fatalf("wrong result length for %T", input)
			}
			for i := range want {
				if !got[i].Equals(want[i]) {
					t.Fatalf("wrong result for %T at %d", input, i)
				}
			}
		}
		if seq.Rest() != firstRest || calls != len(expected) || !input.First().Equals(expected[0]) {
			t.Fatalf("source or cached tail changed for %T", input)
		}
	}
}

func TestTransformSeqFilterEmptyAndRetry(t *testing.T) {
	calls := 0
	seq := NewFilterSeq(Proc{Fn: func(args []Object) Object {
		calls++
		if calls == 1 {
			panic("retry")
		}
		return Boolean{B: false}
	}}, NewArrayVectorFrom(Int{I: 1}), true).(*TransformSeq)
	func() {
		defer func() {
			if recover() != "retry" {
				t.Fatal("expected callback failure")
			}
		}()
		seq.First()
	}()
	if seq.IsRealized() {
		t.Fatal("failed realization cached")
	}
	if !seq.IsEmpty() || seq.First() != NIL || !seq.Rest().IsEmpty() || calls != 2 {
		t.Fatal("empty sequence or failed realization retry is incorrect")
	}
}
