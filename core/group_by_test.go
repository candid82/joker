package core

import "testing"

func TestGroupByForeignCallbackArguments(t *testing.T) {
	var saved [][]Object
	f := Proc{Package: "test", Fn: func(args []Object) Object {
		saved = append(saved, args)
		return NIL
	}}
	input := NewArrayVectorFrom(Int{I: 1}, Int{I: 2}, Int{I: 3})
	result := procGroupBy([]Object{f, input}).(Map)
	if len(saved) != input.Count() {
		t.Fatal("group-by did not call the key function once per item")
	}
	for i, args := range saved {
		if !args[0].Equals(input.At(i)) {
			t.Fatal("foreign callback's retained arguments were overwritten")
		}
	}
	if ok, group := result.Get(NIL); !ok || !group.Equals(input) {
		t.Fatal("incorrect grouping")
	}
}
