package core

import (
	"math/rand"
	"testing"
)

// All these keys have the same hash, exercising the HAMT collision node's
// copy-on-write, growth, and deletion paths.
type collisionKey struct {
	InfoHolder
	id int
}

func (k *collisionKey) Equals(other interface{}) bool {
	o, ok := other.(*collisionKey)
	return ok && k.id == o.id
}
func (k *collisionKey) Hash() uint32                     { return 42 }
func (k *collisionKey) GetType() *Type                   { return TYPE.Int }
func (k *collisionKey) ToString(bool) string             { return "collision key" }
func (k *collisionKey) WithInfo(info *ObjectInfo) Object { k.info = info; return k }

func TestTransientHashMapCollisions(t *testing.T) {
	keys := make([]Object, 60)
	for i := range keys {
		keys[i] = &collisionKey{id: i}
	}
	original := NewHashMap(keys[0], Int{I: 0})
	builder := original.AsTransient().(*TransientHashMap)
	for i, key := range keys {
		builder.AssocBang(key, Int{I: i * 2})
	}
	if builder.Count() != len(keys) {
		t.Fatalf("count after insertion: %d", builder.Count())
	}
	for i := range keys {
		key := &collisionKey{id: i}
		ok, value := builder.Get(key)
		if !ok || !value.Equals(Int{I: i * 2}) {
			t.Fatalf("missing collision key %d", i)
		}
	}
	for i := 0; i < len(keys); i += 2 {
		builder.WithoutBang(&collisionKey{id: i})
	}
	if builder.Count() != len(keys)/2 {
		t.Fatalf("count after deletion: %d", builder.Count())
	}
	result := builder.Persistent().(*HashMap)
	for i := range keys {
		ok, value := result.Get(keys[i])
		if ok != (i%2 == 1) || (ok && !value.Equals(Int{I: i * 2})) {
			t.Fatalf("bad key %d", i)
		}
	}
	if ok, value := original.Get(keys[0]); !ok || !value.Equals(Int{I: 0}) {
		t.Fatal("mutated original")
	}
	defer func() {
		if recover() == nil {
			t.Error("transient remained usable after persistent!")
		}
	}()
	builder.Get(keys[0])
}

func TestTransientHashMapRandom(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	original := NewHashMap(Int{I: -1}, Int{I: 5})
	builder := original.AsTransient().(*TransientHashMap)
	want := map[int]int{-1: 5}
	for step := 0; step < 9000; step++ {
		key := r.Intn(450) - 1
		if r.Intn(3) == 0 {
			builder.WithoutBang(Int{I: key})
			delete(want, key)
		} else {
			val := r.Intn(10000)
			builder.AssocBang(Int{I: key}, Int{I: val})
			want[key] = val
		}
		if step%149 == 0 {
			if builder.Count() != len(want) {
				t.Fatalf("at %d: count %d != %d", step, builder.Count(), len(want))
			}
			for k, v := range want {
				ok, got := builder.Get(Int{I: k})
				if !ok || !got.Equals(Int{I: v}) {
					t.Fatalf("at %d: key %d", step, k)
				}
			}
		}
	}
	result := builder.Persistent().(*HashMap)
	if result.Count() != len(want) {
		t.Fatal("persistent count differs")
	}
	for k, v := range want {
		ok, got := result.Get(Int{I: k})
		if !ok || !got.Equals(Int{I: v}) {
			t.Fatalf("persistent key %d", k)
		}
	}
	if original.Count() != 1 {
		t.Fatal("mutated original")
	}
	// A second builder must copy nodes left editable by the first builder.
	other := result.AsTransient().(*TransientHashMap)
	other.AssocBang(Int{I: -99}, Int{I: 1})
	other.Persistent()
	if ok, _ := result.Get(Int{I: -99}); ok {
		t.Fatal("mutated persistent result")
	}
}

func TestTransientVectorRandom(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	objs := make([]Object, 1100)
	for i := range objs {
		objs[i] = Int{I: i}
	}
	original := NewVectorFrom(objs...)
	builder := original.AsTransient().(*TransientVector)
	want := append([]Object(nil), objs...)
	for step := 0; step < 5000; step++ {
		switch r.Intn(3) {
		case 0:
			v := Int{I: step}
			builder.ConjBang(v)
			want = append(want, v)
		case 1:
			if len(want) > 0 {
				builder.PopBang()
				want = want[:len(want)-1]
			}
		case 2:
			if len(want) > 0 {
				i := r.Intn(len(want))
				v := Int{I: -step}
				builder.AssocBang(Int{I: i}, v)
				want[i] = v
			}
		}
		if step%131 == 0 {
			if builder.Count() != len(want) {
				t.Fatalf("at %d: count %d != %d", step, builder.Count(), len(want))
			}
			for i := range want {
				if !builder.Nth(i).Equals(want[i]) {
					t.Fatalf("at %d: index %d", step, i)
				}
			}
		}
	}
	result := builder.Persistent().(*Vector)
	for i := range want {
		if !result.Nth(i).Equals(want[i]) {
			t.Fatalf("persistent index %d", i)
		}
	}
	for i := range objs {
		if !original.Nth(i).Equals(objs[i]) {
			t.Fatalf("original index %d changed", i)
		}
	}
	other := result.AsTransient().(*TransientVector)
	other.AssocBang(Int{I: 0}, Int{I: -99})
	other.Persistent()
	if !result.Nth(0).Equals(want[0]) {
		t.Fatal("mutated persistent result")
	}
}

func TestTransientVectorBoundaryCycles(t *testing.T) {
	builder := EmptyArrayVector().AsTransient().(*TransientVector)
	for i := 0; i < 2200; i++ {
		builder.ConjBang(Int{I: i})
		if !builder.Nth(i).Equals(Int{I: i}) {
			t.Fatalf("append at %d", i)
		}
	}
	for i := 2199; i >= 0; i-- {
		if !builder.Nth(i).Equals(Int{I: i}) {
			t.Fatalf("pop at %d", i)
		}
		builder.PopBang()
	}
	if builder.Count() != 0 {
		t.Fatal("nonempty vector")
	}
	builder.ConjBang(Int{I: 7})
	if !builder.Persistent().(*Vector).Nth(0).Equals(Int{I: 7}) {
		t.Fatal("reuse after pop")
	}
}

func BenchmarkVectorConjTransient(b *testing.B) {
	for run := 0; run < b.N; run++ {
		builder := EmptyArrayVector().AsTransient()
		for i := 0; i < 4096; i++ {
			builder = builder.ConjBang(Int{I: i})
		}
		builder.Persistent()
	}
}
func BenchmarkVectorConjPersistent(b *testing.B) {
	for run := 0; run < b.N; run++ {
		var v Conjable = EmptyArrayVector()
		for i := 0; i < 4096; i++ {
			v = v.Conj(Int{I: i})
		}
	}
}
func BenchmarkMapAssocTransient(b *testing.B) {
	for run := 0; run < b.N; run++ {
		var builder TransientAssociative = EmptyArrayMap().AsTransient().(TransientAssociative)
		for i := 0; i < 4096; i++ {
			builder = builder.AssocBang(Int{I: i}, Int{I: i})
		}
		builder.Persistent()
	}
}
func BenchmarkMapAssocPersistent(b *testing.B) {
	for run := 0; run < b.N; run++ {
		var m Associative = EmptyArrayMap()
		for i := 0; i < 4096; i++ {
			m = m.Assoc(Int{I: i}, Int{I: i})
		}
	}
}
