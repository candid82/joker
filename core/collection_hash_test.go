package core

import "testing"

func TestDirectCollectionHashCompatibility(t *testing.T) {
	for n := 0; n <= 32; n++ {
		array := EmptyArrayMap()
		var hashed Map = EmptyHashMap
		set := EmptySet()
		for i := 0; i < n; i++ {
			key := Int{I: i}
			value := NewArrayVectorFrom(key, MakeString("value"), NIL)
			array.Add(key, value)
			hashed = hashed.Assoc(key, value).(Map)
			set.Add(key)
		}
		if got, want := array.Hash(), hashUnordered(array.Seq(), 1); got != want || got != hashed.Hash() {
			t.Fatalf("map size %d: direct %d, sequence %d, hash map %d", n, got, want, hashed.Hash())
		}
		if got, want := set.Hash(), hashUnordered(set.Seq(), 2); got != want {
			t.Fatalf("set size %d: direct %d, sequence %d", n, got, want)
		}
		// Exercise internal mutation after hashing: no cached hash may survive.
		array.Set(Int{I: 0}, set)
		if array.Hash() != hashUnordered(array.Seq(), 1) {
			t.Fatal("hash after mutation or nested collection differs")
		}
	}
	for _, value := range []uint32{0, 1, 255, 256, 0x12345678, 0xffffffff} {
		h := getHash()
		h.Write(uint32ToBytes(value))
		if hashUint32(2166136261, value) != h.Sum32() {
			t.Fatalf("hash byte order differs for %x", value)
		}
	}
}
