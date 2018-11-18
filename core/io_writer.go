package core

import (
	"io"
	"unsafe"
)

type (
	IOWriter struct {
		io.Writer
	}
)

func (iow *IOWriter) ToString(escape bool) string {
	return "#object[IOWriter]"
}

func (iow *IOWriter) Equals(other interface{}) bool {
	return iow == other
}

func (iow *IOWriter) GetInfo() *ObjectInfo {
	return nil
}

func (iow *IOWriter) GetType() *Type {
	return TYPE.IOWriter
}

func (iow *IOWriter) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(iow)))
}

func (iow *IOWriter) WithInfo(info *ObjectInfo) Object {
	return iow
}
