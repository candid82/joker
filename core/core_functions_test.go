package core

import "testing"

// A host reducer is allowed to support Reduce without supporting Seqable.
type nativeReduceProbe struct {
	Object
	items []Object
	calls int
}

func (source *nativeReduceProbe) reduceInit(step Callable, initial Object) Object {
	source.calls++
	result := initial
	for _, item := range source.items {
		result = step.Call([]Object{result, item})
	}
	return result
}

func (source *nativeReduceProbe) reduce(step Callable) Object {
	if len(source.items) == 0 {
		return step.Call(nil)
	}
	return source.reduceInit(step, source.items[0])
}

func TestNativeCoreReduceProtocol(t *testing.T) {
	items := []Object{boxInt(1), boxInt(2), boxInt(1)}
	identity := Proc{Fn: func(args []Object) Object { return args[0] }}
	for _, tc := range []struct {
		name string
		fn   ProcFn
		args func(Object) []Object
		want Object
	}{
		{"mapv", procMapv, func(source Object) []Object { return []Object{identity, source} }, NewArrayVectorFrom(items...)},
		{"filterv", procFilterv, func(source Object) []Object { return []Object{identity, source} }, NewArrayVectorFrom(items...)},
		{"frequencies", procFrequencies, func(source Object) []Object { return []Object{source} }, NewHashMap(boxInt(1), boxInt(2), boxInt(2), boxInt(1))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &nativeReduceProbe{Object: NIL, items: items}
			result := tc.fn(tc.args(source))
			if source.calls != 1 || !result.Equals(tc.want) {
				t.Fatal("native builder bypassed Reduce")
			}
		})
	}
	keys := &nativeReduceProbe{Object: NIL, items: []Object{MakeKeyword("a"), MakeKeyword("b")}}
	m := NewHashMap(MakeKeyword("a"), NewHashMap(MakeKeyword("b"), boxInt(7)))
	if got := procGetIn([]Object{m, keys}); keys.calls != 1 || !got.Equals(boxInt(7)) {
		t.Fatal("two-argument get-in bypassed Reduce")
	}
}

func TestNativeCoreOwnedCallbackArguments(t *testing.T) {
	for _, operation := range []string{"mapv", "filterv", "mapv-multi", "map-multi", "keep", "map-indexed", "keep-indexed", "take-while", "drop-while"} {
		t.Run(operation, func(t *testing.T) {
			var history [][]Object
			callback := Proc{Fn: func(args []Object) Object {
				history = append(history, args)
				return args[len(args)-1]
			}}
			source := NewArrayVectorFrom(boxInt(1), boxInt(2), boxInt(3))
			var result Object
			switch operation {
			case "mapv":
				result = procMapv([]Object{callback, source})
			case "filterv":
				result = procFilterv([]Object{callback, source})
			case "mapv-multi":
				result = procMapv([]Object{callback, source, source, source, source})
			case "map-multi":
				result = procMapMulti([]Object{callback, source, source, source, source})
			case "keep":
				result = procKeep([]Object{callback, source})
			case "map-indexed":
				result = procMapIndexed([]Object{callback, source})
			case "keep-indexed":
				result = procKeepIndexed([]Object{callback, source})
			case "take-while":
				result = procTakeWhile([]Object{callback, source})
			case "drop-while":
				result = procDropWhile([]Object{callback, source})
			}
			ToSlice(result.(Seqable).Seq())
			if len(history) != 3 {
				t.Fatalf("got %d callbacks", len(history))
			}
			for i, args := range history {
				if !args[len(args)-1].Equals(boxInt(i + 1)) {
					t.Fatal("host callback's retained argument slice was overwritten")
				}
			}
		})
	}
}

func TestNativeCallbackVarResolution(t *testing.T) {
	value := &Var{Value: Proc{Fn: func([]Object) Object { return boxInt(1) }}}
	callback := newNativeCallback(value, 1)
	callback.args[0] = NIL
	if !callback.call().Equals(boxInt(1)) {
		t.Fatal("incorrect initial Var value")
	}
	value.Value = Proc{Fn: func([]Object) Object { return boxInt(2) }}
	if !callback.call().Equals(boxInt(2)) {
		t.Fatal("native callback cached a Var's resolved value")
	}
}
