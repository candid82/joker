package gen_go

import (
	"reflect"
	"unsafe"
)

// NOTE: Below this line, code comes from github.com/jcburley/go-spew:

// This comes from (davecgh|jcburley)/go-spew/bypass.go.
const flagPrivate = 0x20

// This comes from (davecgh|jcburley)/go-spew/bypass.go.
var flagValOffset = func() uintptr {
	field, ok := reflect.TypeOf(reflect.Value{}).FieldByName("flag")
	if !ok {
		panic("reflect.Value has no flag field")
	}
	return field.Offset
}()

// This comes from (davecgh|jcburley)/go-spew/bypass.go.
type flag uintptr

// This comes from (davecgh|jcburley)/go-spew/bypass.go.
func flagField(v *reflect.Value) *flag {
	return (*flag)(unsafe.Pointer(uintptr(unsafe.Pointer(v)) + flagValOffset))
}

// This comes from (davecgh|jcburley)/go-spew/bypass.go.
func UnsafeReflectValue(v reflect.Value) reflect.Value {
	if !v.IsValid() || (v.CanInterface() && v.CanAddr()) {
		return v
	}
	flagFieldPtr := flagField(&v)
	*flagFieldPtr &^= flagPrivate
	return v
}
