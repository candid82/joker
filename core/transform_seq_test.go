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
