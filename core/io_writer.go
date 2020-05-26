package core

import (
	"io"
	"unsafe"
)

type (
	IOWriter struct {
		io.Writer
		hash uint32
	}
)

func MakeIOWriter(w io.Writer) *IOWriter {
	res := &IOWriter{w, 0}
	res.hash = HashPtr(uintptr(unsafe.Pointer(res)))
	return res
}

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
	return iow.hash
}

func (iow *IOWriter) WithInfo(info *ObjectInfo) Object {
	return iow
}

func (iow *IOWriter) Close() error {
	if c, ok := iow.Writer.(io.Closer); ok {
		return c.Close()
	} else {
		return RT.NewError("Object is not closable: " + iow.ToString(false))
	}
}
