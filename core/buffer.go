package core

import (
	"bytes"
	"unsafe"
)

type (
	Buffer struct {
		*bytes.Buffer
		hash uint32
	}
)

func MakeBuffer(b *bytes.Buffer) *Buffer {
	res := &Buffer{b, 0}
	res.hash = HashPtr(uintptr(unsafe.Pointer(res)))
	return res
}

func (b *Buffer) ToString(escape bool) string {
	return b.String()
}

func (b *Buffer) Equals(other interface{}) bool {
	return b == other
}

func (b *Buffer) GetInfo() *ObjectInfo {
	return nil
}

func (b *Buffer) GetType() *Type {
	return TYPE.Buffer
}

func (b *Buffer) Hash() uint32 {
	return b.hash
}

func (b *Buffer) WithInfo(info *ObjectInfo) Object {
	return b
}
