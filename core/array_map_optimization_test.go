package core

import "testing"

func TestArrayMapPersistentUpdatesDoNotShareBackingArray(t *testing.T) {
	original := EmptyArrayMap()
	original.Set(Int{I: 1}, Int{I: 10})
	original.Set(Int{I: 2}, Int{I: 20})

	removed := original.Without(Int{I: 1}).(*ArrayMap)
	added := original.Assoc(Int{I: 3}, Int{I: 30}).(*ArrayMap)
	unchanged := original.Without(Int{I: 9}).(*ArrayMap)

	original.Set(Int{I: 2}, Int{I: 99})
	for _, m := range []*ArrayMap{removed, added, unchanged} {
		if ok, value := m.Get(Int{I: 2}); !ok || !value.Equals(Int{I: 20}) {
			t.Fatalf("persistent result changed with original: %v", m)
		}
	}
	if ok, _ := removed.Get(Int{I: 1}); ok || removed.Count() != 1 {
		t.Fatalf("removed map retained deleted key: %v", removed)
	}
	if ok, value := added.Get(Int{I: 3}); !ok || !value.Equals(Int{I: 30}) {
		t.Fatalf("assoc lost new key: %v", added)
	}
}

func TestIsInstanceNilAndInt(t *testing.T) {
	if IsInstance(TYPE.Int, NIL) || !IsInstance(TYPE.Int, Int{I: 42}) {
		t.Fatal("type checks for nil and integers must retain their semantics")
	}
}
