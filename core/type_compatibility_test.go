package core

import (
	"reflect"
	"sync"
	"testing"
)

func TestTypeCompatibilityMatchesReflection(t *testing.T) {
	objectType := &Type{name: "Object", reflectType: reflect.TypeOf((*Object)(nil)).Elem()}
	types := []*Type{
		TYPE.Int, TYPE.String, TYPE.Nil, TYPE.Type, TYPE.Number, objectType,
		TYPE.Vec, TYPE.Vector, TYPE.ArrayVector, TYPE.Seq, TYPE.Map,
		TYPE.ArrayMap, TYPE.HashMap, TYPE.Set, TYPE.MapSet, TYPE.Editable,
		TYPE.TransientCollection, TYPE.TransientAssociative, TYPE.Callable,
	}
	for _, abstract := range types {
		for _, concrete := range types {
			want := concrete.reflectType == abstract.reflectType
			if abstract.reflectType.Kind() == reflect.Interface {
				want = concrete.reflectType.Implements(abstract.reflectType)
			}
			for attempt := 0; attempt < 2; attempt++ {
				if got := IsEqualOrImplements(abstract, concrete); got != want {
					t.Fatalf("%s / %s: got %v, want %v", abstract.name, concrete.name, got, want)
				}
			}
		}
	}
}

// Distinct Joker type descriptors can describe the same immutable Go type.
func TestTypeCompatibilityAliasesAndNil(t *testing.T) {
	alias := &Type{name: "vector-alias", reflectType: TYPE.Vector.reflectType}
	if !IsEqualOrImplements(TYPE.Vec, alias) || !IsEqualOrImplements(TYPE.Vector, alias) {
		t.Fatal("alias must have the original reflection-type compatibility")
	}
	objectType := &Type{name: "Object", reflectType: reflect.TypeOf((*Object)(nil)).Elem()}
	for _, abstract := range []*Type{TYPE.Nil, objectType, TYPE.Vec, TYPE.Int} {
		if IsInstance(abstract, NIL) {
			t.Fatalf("nil must not be an instance of %s", abstract.name)
		}
	}
	if !IsInstance(TYPE.Number, Int{I: 1}) || IsInstance(TYPE.Seq, Int{I: 1}) {
		t.Fatal("positive and negative interface predicates changed")
	}
}

type compatibilityTestInterface interface{ compatibilityMarker() }
type compatibilityTestMatch struct{}
type compatibilityTestMiss struct{}

func (compatibilityTestMatch) compatibilityMarker() {}

func TestTypeCompatibilityConcurrentMissesAndHits(t *testing.T) {
	abstract := &Type{reflectType: reflect.TypeOf((*compatibilityTestInterface)(nil)).Elem()}
	match := &Type{reflectType: reflect.TypeOf(compatibilityTestMatch{})}
	miss := &Type{reflectType: reflect.TypeOf(compatibilityTestMiss{})}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				if !IsEqualOrImplements(abstract, match) || IsEqualOrImplements(abstract, miss) {
					t.Error("concurrent cache lookup returned incorrect compatibility")
					return
				}
			}
		}()
	}
	wg.Wait()
}

var compatibilityBenchmarkResult bool

func BenchmarkTypeCompatibility(b *testing.B) {
	for _, tc := range []struct {
		name     string
		abstract *Type
		concrete *Type
	}{
		{"positive", TYPE.Vec, TYPE.Vector},
		{"negative", TYPE.Seq, TYPE.Vector},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.Run("reflection", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					compatibilityBenchmarkResult = tc.concrete.reflectType.Implements(tc.abstract.reflectType)
				}
			})
			b.Run("cached", func(b *testing.B) {
				IsEqualOrImplements(tc.abstract, tc.concrete)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					compatibilityBenchmarkResult = IsEqualOrImplements(tc.abstract, tc.concrete)
				}
			})
		})
	}
}
