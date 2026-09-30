package core

import "testing"

func TestTakeSequenceSemantics(t *testing.T) {
	for _, tt := range []struct {
		code string
		want string
	}{
		{`(vec (take 2 [nil false 3]))`, `[nil false]`},
		{`(vec (take 10 [1 2]))`, `[1 2]`},
		{`(vec (take 2 nil))`, `[]`},
		{`(vec (take 0 42))`, `[]`},
		{`(vec (take -1 42))`, `[]`},
		{`(vec (take 3/2 [1 2 3]))`, `[1 2]`},
		{`(vec (take 1.5 [1 2 3]))`, `[1 2]`},
		{`(vec (take 2N [1 2 3]))`, `[1 2]`},
		{`(vec (take 2M [1 2 3]))`, `[1 2]`},
		{`(vec (take 2 "aλβ"))`, `[\a \λ]`},
		{`(vec (take 3 (range)))`, `[0 1 2]`},
		{`(let [calls (atom []) s (take 2 (map #(do (swap! calls conj %) %) [nil false 3]))
		        before [(realized? s) @calls]
		        head (first s) tail (rest s) a (vec s) b (vec s)]
		    [before head (identical? tail (rest s)) a b @calls (realized? s)])`,
			`[[false []] nil true [nil false] [nil false] [nil false] true]`},
		{`(let [calls (atom 0) s (take 0 (lazy-seq (swap! calls inc) [1]))]
		    [(vec s) @calls])`, `[[] 0]`},
		{`(let [calls (atom 0) s (take 1 (lazy-seq
		        (if (= 1 (swap! calls inc)) (throw (ex-info "retry" {})) [7])))
		        failed (try (first s) (catch Error e (realized? s)))]
		    [failed (vec s) @calls])`, `[false [7] 2]`},
		{`(let [s (with-meta (take 2 [1 2 3]) {:tag :take})]
		    [(meta s) (vec s) (= s '(1 2)) (= (hash s) (hash '(1 2))) (type s)])`,
			`[{:tag :take} [1 2] true true LazySeq]`},
		{`(let [s (take 1 42)] (try (first s) (catch Error e (realized? s))))`, `false`},
		{`(let [s (take :invalid [1])] (try (first s) (catch Error e (realized? s))))`, `false`},
		{`(let [calls (atom 0) input (cons 1 (lazy-seq
		        (if (= 1 (swap! calls inc)) (throw (ex-info "retry-tail" {})) [2 3])))
		        s (take 2 input) tail (rest s)
		        failed (try (first tail) (catch Error e (realized? tail)))]
		    [failed (vec s) @calls (realized? tail)])`, `[false [1 2] 2 true]`},
	} {
		t.Run(tt.code, func(t *testing.T) {
			if got := evalAndCompile(t, tt.code).ToString(true); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

// Exercise public take on an indexed source, isolating its per-element cost
// from allocations made by source callbacks. The old closure/cons implementation
// uses over 2,000 allocations here; leave headroom for representation changes.
func TestTakeAllocationBudget(t *testing.T) {
	take := GLOBAL_ENV.CoreNamespace.Intern(MakeSymbol("take")).Resolve().(Callable)
	values := make([]Object, 256)
	for i := range values {
		values[i] = NIL
	}
	args := []Object{Int{I: len(values)}, NewArrayVectorFrom(values...)}
	allocs := testing.AllocsPerRun(5, func() {
		if got := SeqCount(take.Call(args).(Seq)); got != len(values) {
			t.Fatalf("got %d items", got)
		}
	})
	t.Logf("take allocated %.0f objects for %d elements", allocs, len(values))
	if allocs > 1200 {
		t.Fatalf("take allocated %.0f objects; budget is 1200", allocs)
	}
}

func BenchmarkTakeIndexed(b *testing.B) {
	take := GLOBAL_ENV.CoreNamespace.Intern(MakeSymbol("take")).Resolve().(Callable)
	values := make([]Object, 1000)
	for i := range values {
		values[i] = NIL
	}
	args := []Object{Int{I: len(values)}, NewArrayVectorFrom(values...)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := SeqCount(take.Call(args).(Seq)); got != len(values) {
			b.Fatalf("got %d items", got)
		}
	}
}
