package core

import (
	"bufio"
	"unsafe"
)

type (
	BufferedReader struct {
		*bufio.Reader
	}
)

func (br *BufferedReader) ToString(escape bool) string {
	return "#object[BufferedReader]"
}

func (br *BufferedReader) Equals(other interface{}) bool {
	return br == other
}

func (br *BufferedReader) GetInfo() *ObjectInfo {
	return nil
}

func (br *BufferedReader) GetType() *Type {
	return TYPE.BufferedReader
}

func (br *BufferedReader) Hash() uint32 {
	return HashPtr(uintptr(unsafe.Pointer(br)))
}

func (br *BufferedReader) WithInfo(info *ObjectInfo) Object {
	return br
}
