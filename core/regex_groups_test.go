package core

import (
	"regexp"
	"strings"
	"testing"
)

func TestRegexGroups(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		input   string
		want    Object
	}{
		{"no match", `(a)`, "b", NIL},
		{"no captures", `a+`, "baa", String{S: "aa"}},
		{"empty match", `a*`, "b", String{S: ""}},
		{"capture", `(a)`, "a", NewVectorFrom(String{S: "a"}, String{S: "a"})},
		{"optional capture", `(a)?(b)`, "b", NewVectorFrom(String{S: "b"}, NIL, String{S: "b"})},
		{"empty capture", `(a*)`, "b", NewVectorFrom(String{S: ""}, String{S: ""})},
		{"plain Hiccup tag", `([^\s\.#]+)(?:#([^\s\.#]+))?(?:\.([^\s#]+))?`, "td",
			NewVectorFrom(String{S: "td"}, String{S: "td"}, NIL, NIL)},
		{"Hiccup tag shorthand", `([^\s\.#]+)(?:#([^\s\.#]+))?(?:\.([^\s#]+))?`, "td#court.reserved",
			NewVectorFrom(String{S: "td#court.reserved"}, String{S: "td"}, String{S: "court"}, String{S: "reserved"})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			indexes := regexp.MustCompile(tc.pattern).FindStringSubmatchIndex(tc.input)
			got := reGroups(tc.input, indexes)
			if !got.Equals(tc.want) || got.GetType() != tc.want.GetType() {
				t.Fatalf("got %s (%T), want %s (%T)", got.ToString(true), got, tc.want.ToString(true), tc.want)
			}
		})
	}
}

func TestRegexGroupsLargeResultsAndPersistence(t *testing.T) {
	// Include the full match: exercise 32-element construction and the >32
	// fallback, plus a result spanning more than one trie tail.
	for _, captures := range []int{31, 32, 64} {
		input := strings.Repeat("a", captures)
		indexes := regexp.MustCompile(strings.Repeat("(a)", captures)).FindStringSubmatchIndex(input)
		v := reGroups(input, indexes).(*Vector)
		if v.Count() != captures+1 || !v.At(0).Equals(String{S: input}) {
			t.Fatalf("incorrect full match or result size for %d captures", captures)
		}
		for i := 1; i < v.Count(); i++ {
			if !v.At(i).Equals(String{S: "a"}) {
				t.Fatalf("incorrect group %d for %d captures", i, captures)
			}
		}
		updated := v.Assoc(Int{I: 0}, String{S: "changed"})
		grown := v.Conjoin(NIL)
		if !v.At(0).Equals(String{S: input}) || v.Count() != captures+1 ||
			!updated.(*Vector).At(0).Equals(String{S: "changed"}) || grown.Count() != captures+2 {
			t.Fatal("persistent vector updates changed the original regex result")
		}
	}
}

var regexGroupsBenchmarkResult Object

func BenchmarkRegexGroups(b *testing.B) {
	indexes := []int{0, 2, 0, 2, -1, -1, -1, -1}
	b.Run("conjoin-baseline", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			v := EmptyVector()
			for i := 0; i < len(indexes); i += 2 {
				if indexes[i] == -1 {
					v = v.Conjoin(NIL)
				} else {
					v = v.Conjoin(String{S: "td"[indexes[i]:indexes[i+1]]})
				}
			}
			regexGroupsBenchmarkResult = v
		}
	})
	b.Run("construct-once", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			regexGroupsBenchmarkResult = reGroups("td", indexes)
		}
	})
}
