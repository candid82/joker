package hiccup

import (
	"sync"
	"testing"

	. "github.com/candid82/joker/core"
)

func TestParseTag(t *testing.T) {
	for _, tc := range []struct {
		input, tag, id, class string
	}{
		{"div", "div", "", ""},
		{"div#id.a.b", "div", "id", "a b"},
		{"div..", "div", "", " "},
		{"div\vname", "div\vname", "", ""}, // RE2's \s excludes vertical tab.
		{"", "", "", ""},
		{"div invalid", "", "", ""},
		{"div#", "", "", ""},
		{"div#a#b", "", "", ""},
	} {
		tag, id, class := parseTag(tc.input)
		if tag != tc.tag || id != tc.id || class != tc.class {
			t.Fatalf("parseTag(%q) = %q, %q, %q", tc.input, tag, id, class)
		}
	}
}

func TestRendererIndependentBuffers(t *testing.T) {
	attrs := EmptyArrayMap()
	attrs.Add(MakeKeyword("title"), MakeString("<&>"))
	element := NewArrayVectorFrom(MakeKeyword("div#id.a"), attrs,
		NewArrayVectorFrom(MakeKeyword("br")), rawString(MakeString("<b>raw</b>")))
	want := "<div class=\"a\" id=\"id\" title=\"&lt;&amp;&gt;\"><br /><b>raw</b></div>"
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if got := renderHTML([]Object{element}); got != want {
					t.Errorf("render got %q, want %q", got, want)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestNilIsContentNotOptionsOrAttributes(t *testing.T) {
	options := EmptyArrayMap()
	options.Add(modeKey, xmlMode)
	element := NewArrayVectorFrom(MakeKeyword("div"), NIL)
	if got := renderHTML([]Object{options, element}); got != "<div></div>" {
		t.Fatalf("nil content must force a container: %q", got)
	}
	if _, ok := attributeMap(NIL); ok {
		t.Fatal("Joker nil is not an attribute map")
	}
}

func BenchmarkNativeRenderer(b *testing.B) {
	element := NewArrayVectorFrom(MakeKeyword("div"),
		NewArrayVectorFrom(MakeKeyword("span"), MakeString("hello <&>")))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		renderHTML([]Object{element})
	}
}
