package core

import (
	"bufio"
	"io"
	"unsafe"
)

type (
	BufferedReader struct {
		*bufio.Reader
		hash uint32
	}
)

func MakeBufferedReader(rd io.Reader) *BufferedReader {
	res := &BufferedReader{bufio.NewReader(rd), 0}
	res.hash = HashPtr(uintptr(unsafe.Pointer(res)))
	return res
}

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
	return br.hash
}

func (br *BufferedReader) WithInfo(info *ObjectInfo) Object {
	return br
}
