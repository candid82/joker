package core

import (
	"io"
	"unsafe"
)

type (
	IOReader struct {
		io.Reader
	}
)

func (ior *IOReader) ToString(escape bool) string {
	return "#object[IOReader]"
}

func (ior *IOReader) Equals(other interface{}) bool {
	return ior == other
}

func (ior *IOReader) GetInfo() *ObjectInfo {
	return nil
}

func (ior *IOReader) GetType() *Type {
	return TYPE.IOReader
}

func (ior *IOReader) Hash() uint32 {
	return HashPtr(uintptr(unsafe.Pointer(ior)))
}

func (ior *IOReader) WithInfo(info *ObjectInfo) Object {
	return ior
}
