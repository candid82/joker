package core

import (
	"net"
	"os"
	"unsafe"
)

type JokerWriter struct {
	O *os.File
	N net.Conn
}

func (j JokerWriter) Write(p []byte) (n int, err error) {
	if j.O != nil {
		n, err = j.O.Write(p)
	} else {
		n, err = j.N.Write(p)
	}
	return
}

func (j JokerWriter) Sync() error {
	if j.O != nil {
		return j.O.Sync()
	}
	return nil
}

func (br *JokerWriter) ToString(escape bool) string {
	return "#object[JokerWriter]"
}

func (br *JokerWriter) Equals(other interface{}) bool {
	return br == other
}

func (br *JokerWriter) GetInfo() *ObjectInfo {
	return nil
}

func (br *JokerWriter) GetType() *Type {
	return TYPE.JokerWriter
}

func (br *JokerWriter) Hash() uint32 {
	return hashPtr(uintptr(unsafe.Pointer(br)))
}

func (br *JokerWriter) WithInfo(info *ObjectInfo) Object {
	return br
}
