package core

import "testing"

type retainingTestCallable struct {
	Object
	args []Object
}

func (f *retainingTestCallable) Call(args []Object) Object {
	f.args = args
	return NIL
}

func TestVMBuiltinCallableArguments(t *testing.T) {
	key := MakeKeyword("key")
	symbol := MakeSymbol("key")
	value := Int{I: 42}.WithInfo(&ObjectInfo{})
	m := EmptyArrayMap()
	m.Add(key, value)
	m.Add(symbol, value)
	set := EmptySet()
	set.Add(value)
	cases := []struct {
		callee Object
		args   []Object
		want   Object
	}{
		{key, []Object{m}, value},
		{symbol, []Object{m}, value},
		{key, []Object{NIL, value}, value},
		{key, []Object{NIL}, NIL},
		{m, []Object{key}, value},
		{m, []Object{NIL, value}, value},
		{NewHashMap(key, value), []Object{key}, value},
		{set, []Object{value}, value},
		{set, []Object{NIL}, NIL},
		{NewArrayVectorFrom(value), []Object{Int{I: 0}}, value},
		{NewVectorFrom(value), []Object{Int{I: 0}}, value},
	}
	for _, test := range cases {
		vm := NewVM()
		vm.Push(test.callee)
		for _, arg := range test.args {
			vm.Push(arg)
		}
		if vm.callValue(test.callee, len(test.args)) || vm.Pop() != test.want {
			t.Fatalf("incorrect callable result for %T", test.callee)
		}
		for _, slot := range vm.stack {
			if slot != nil {
				t.Fatalf("%T retained a stack value", test.callee)
			}
		}
		expectJokerPanic(t, func() {
			Evaluate(&CallExpr{callable: &LiteralExpr{obj: test.callee}})
		})
	}
}

func TestVMUnknownCallableOwnsArguments(t *testing.T) {
	f := &retainingTestCallable{Object: NIL}
	vm := NewVM()
	vm.Push(f)
	vm.Push(Int{I: 7})
	vm.callValue(f, 1)
	vm.Reset()
	vm.Push(Int{I: 8})
	vm.Push(Int{I: 9})
	if len(f.args) != 1 || !f.args[0].Equals(Int{I: 7}) {
		t.Fatal("unknown callable's arguments overwritten by stack reuse")
	}
}
