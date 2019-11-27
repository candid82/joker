package core

import (
	"unsafe"
)

type (
	Channel struct {
		ch       chan FutureResult
		isClosed bool
	}
)

func (ch *Channel) ToString(escape bool) string {
	return "#object[Channel]"
}

func (ch *Channel) Equals(other interface{}) bool {
	return ch == other
}

func (ch *Channel) GetInfo() *ObjectInfo {
	return nil
}

func (ch *Channel) GetType() *Type {
	return TYPE.Channel
}

func (ch *Channel) Hash() uint32 {
	return HashPtr(uintptr(unsafe.Pointer(ch)))
}

func (ch *Channel) WithInfo(info *ObjectInfo) Object {
	return ch
}

func MakeChannel(ch chan FutureResult) *Channel {
	return &Channel{ch: ch}
}

func ExtractChannel(args []Object, index int) *Channel {
	return EnsureChannel(args, index)
}

func (ch *Channel) Close() {
	if !ch.isClosed {
		close(ch.ch)
		ch.isClosed = true
	}
}
