package gen_go

// This module currently is needed only to give gen_code access to
// private fields in Joker's core-package types.
//
// Since Joker's core package isn't really designed to be imported by
// any component outside of Joker itself, it's possible that making
// all (pertinent) fields public could be a straightforward way to
// eliminate the need for this module.

// NOTE: This code comes from github.com/jcburley/go-spew (originally
// davecgh/go-spew):

import (
	"reflect"
	"unsafe"
)

const flagPrivate = 0x20

var flagValOffset = func() uintptr {
	field, ok := reflect.TypeOf(reflect.Value{}).FieldByName("flag")
	if !ok {
		panic("reflect.Value has no flag field")
	}
	return field.Offset
}()

type flag uintptr

func flagField(v *reflect.Value) *flag {
	return (*flag)(unsafe.Pointer(uintptr(unsafe.Pointer(v)) + flagValOffset))
}

func UnsafeReflectValue(v reflect.Value) reflect.Value {
	if !v.IsValid() || (v.CanInterface() && v.CanAddr()) {
		return v
	}
	flagFieldPtr := flagField(&v)
	*flagFieldPtr &^= flagPrivate
	return v
}
