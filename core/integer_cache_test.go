package core

import "testing"

var integerCacheResult Object

func TestIntegerArithmeticCacheAllocations(t *testing.T) {
	for _, tt := range []struct {
		name string
		fn   ProcFn
		args []Object
	}{
		{"inc", procInc, []Object{Int{I: 2015}}},
		{"add", procAdd, []Object{Int{I: 2000}, Int{I: 16}}},
		{"dec", procDec, []Object{Int{I: 2017}}},
		{"subtract", procSubtract, []Object{Int{I: 2017}, Int{I: 1}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			allocs := testing.AllocsPerRun(20, func() { integerCacheResult = tt.fn(tt.args) })
			if !integerCacheResult.Equals(Int{I: 2016}) {
				t.Fatal("incorrect arithmetic result")
			}
			if allocs != 0 {
				t.Fatalf("cached arithmetic allocated %.0f objects; want zero", allocs)
			}
		})
	}
}

func TestIntegerArithmeticCacheBounds(t *testing.T) {
	for _, value := range []int{-1, 0, 1, 255, 256, 2016, 2047, 2048} {
		args := []Object{Int{I: value}, Int{I: 0}}
		allocs := testing.AllocsPerRun(20, func() { integerCacheResult = procAdd(args) })
		want := float64(1)
		if value >= 0 && value < 2048 {
			want = 0
		}
		if allocs != want || !integerCacheResult.Equals(Int{I: value}) {
			t.Fatalf("result %d: got %.0f allocations, want %.0f", value, allocs, want)
		}
	}
}

func TestIntegerArithmeticCacheSemantics(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	for _, tt := range []struct {
		x, y int
		want int
		sub  bool
	}{
		{-1, 1, 0, false},
		{254, 1, 255, false},
		{255, 1, 256, false},
		{2000, 16, 2016, false},
		{2046, 1, 2047, false},
		{2047, 1, 2048, false},
		{maxInt, 1, minInt, false},
		{minInt, -1, maxInt, false},
		{1, 1, 0, true},
		{0, 1, -1, true},
		{2017, 1, 2016, true},
		{2048, 1, 2047, true},
		{2049, 1, 2048, true},
		{minInt, 1, maxInt, true},
		{maxInt, -1, minInt, true},
	} {
		fn := procAdd
		if tt.sub {
			fn = procSubtract
		}
		info := &ObjectInfo{}
		x := Int{I: tt.x, Original: "original spelling"}.WithInfo(info)
		result := fn([]Object{x, Int{I: tt.y}})
		i, ok := result.(Int)
		if !ok || i.I != tt.want || i.GetInfo() != nil || i.Original != "" {
			t.Fatalf("%d, %d (subtract=%t): incorrect result %#v", tt.x, tt.y, tt.sub, result)
		}
		located := result.WithInfo(info)
		if located.GetInfo() != info || result.GetInfo() != nil || x.GetInfo() != info || x.(Int).Original != "original spelling" {
			t.Fatal("arithmetic changed operand info or contaminated a cached result")
		}
		again := fn([]Object{x, Int{I: tt.y}})
		if again.GetInfo() != nil || !again.Equals(result) || again.(Int).Original != "" {
			t.Fatal("cached arithmetic result changed on subsequent access")
		}
	}
}

func TestIntegerArithmeticCacheVMCompatibility(t *testing.T) {
	for _, tt := range []struct {
		code string
		want string
	}{
		{`(loop [n 0] (if (= n 2016) n (recur (inc n))))`, `2016`},
		{`(loop [n 2016] (if (= n 0) n (recur (dec n))))`, `0`},
		{`[(type (inc 2015)) (type (inc 2015N)) (type (inc 2015.0)) (type (+ 2000 1/2))]`, `[Int BigInt Double Ratio]`},
		{`[(inc 2015N) (dec 2017N) (+ 2015 1.5) (- 2017 1.5) (+ 2015 1/2) (- 2017 1/2)]`, `[2016N 2016N 2016.5 2015.5 4031/2 4033/2]`},
		{`[(inc' 2015) (dec' 2017) (+' 2015 1) (-' 2017 1)]`, `[2016N 2016N 2016N 2016N]`},
		{`[(identical? (inc 2015) (+ 2000 16)) (= (inc 2015) 2016N) (= (hash (inc 2015)) (hash 2016))]`, `[true true true]`},
		{`(let [f (fn [x] (inc x))] (with-redefs [inc (fn [x] :redefined)] (f 2015)))`, `:redefined`},
		{`(let [f (fn [x y] (+ x y))] (with-redefs [+ (fn [x y] :redefined)] (f 2000 16)))`, `:redefined`},
	} {
		t.Run(tt.code, func(t *testing.T) {
			if got := evalAndCompile(t, tt.code).ToString(true); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func BenchmarkIntegerCacheArithmetic(b *testing.B) {
	for _, tt := range []struct {
		name string
		fn   ProcFn
		args []Object
	}{
		{"inc-cached", procInc, []Object{Int{I: 2015}}},
		{"inc-uncached", procInc, []Object{Int{I: 4095}}},
		{"dec-cached", procDec, []Object{Int{I: 2017}}},
		{"add-cached", procAdd, []Object{Int{I: 2000}, Int{I: 16}}},
		{"add-uncached", procAdd, []Object{Int{I: 4096}, Int{I: 16}}},
		{"subtract-cached", procSubtract, []Object{Int{I: 2017}, Int{I: 1}}},
		{"count-existing-cache", procCount, []Object{NewArrayVectorFrom(NIL)}},
	} {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				integerCacheResult = tt.fn(tt.args)
			}
		})
	}
}
