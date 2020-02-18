package core

import (
	"io"
	"unsafe"
)

type (
	IOReader struct {
		io.Reader
		hash uint32
	}
)

func MakeIOReader(r io.Reader) *IOReader {
	res := &IOReader{r, 0}
	res.hash = HashPtr(uintptr(unsafe.Pointer(res)))
	return res
}

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
	return ior.hash
}

func (ior *IOReader) WithInfo(info *ObjectInfo) Object {
	return ior
}

func (ior *IOReader) Close() error {
	if c, ok := ior.Reader.(io.Closer); ok {
		return c.Close()
	} else {
		return RT.NewError("Object is not closable: " + ior.ToString(false))
	}
}
