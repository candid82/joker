package core

import "testing"

var _ KVReduce = (*ArrayMap)(nil)

func TestArrayMapKVReduceOrderAndArguments(t *testing.T) {
	m := EmptyArrayMap()
	m.Set(NIL, Boolean{B: false})
	m.Set(Int{I: 1}, String{S: "one"})
	m.Set(Int{I: 2}, NIL)
	var calls [][]Object
	f := Proc{Package: "test", Fn: func(args []Object) Object {
		// Native callbacks may retain their argument slice. Each invocation
		// must retain the accumulator/key/value it actually received.
		calls = append(calls, args)
		return Int{I: args[0].(Int).I + 1}
	}}
	result := m.kvreduce(f, Int{I: 0})
	if !result.Equals(Int{I: 3}) || len(calls) != 3 {
		t.Fatalf("unexpected reduction: %v, %d calls", result, len(calls))
	}
	for i, args := range calls {
		if len(args) != 3 || !args[0].Equals(Int{I: i}) ||
			!args[1].Equals(m.arr[2*i]) || !args[2].Equals(m.arr[2*i+1]) {
			t.Fatalf("call %d changed arguments or sequence order: %v", i, args)
		}
	}
	if result := EmptyArrayMap().kvreduce(f, Int{I: 42}); !result.Equals(Int{I: 42}) || len(calls) != 3 {
		t.Fatal("empty reduction must return init without calling the callback")
	}
}

func BenchmarkArrayMapKVReduce(b *testing.B) {
	m := EmptyArrayMap()
	for i := 0; i < 4; i++ {
		m.Set(Int{I: i}, Int{I: i})
	}
	f := Proc{Package: "test", Fn: func(args []Object) Object { return args[0] }}
	b.Run("entry-sequence", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var res Object = NIL
			for s := m.Seq(); !s.IsEmpty(); s = s.Rest() {
				entry := s.First().(Vec)
				res = f.Call([]Object{res, entry.At(0), entry.At(1)})
			}
		}
	})
	b.Run("native-kv", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			m.kvreduce(f, NIL)
		}
	})
}
