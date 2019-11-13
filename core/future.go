package core

import (
	"unsafe"
)

type (
	FutureResult struct {
		value Object
		err   Error
	}
	Future struct {
		ch          chan FutureResult
		result      FutureResult
		isDelivered bool
	}
)

func MakeFutureResult(value Object, err Error) FutureResult {
	return FutureResult{value: value, err: err}
}

func (f *Future) ToString(escape bool) string {
	return "#object[Future]"
}

func (f *Future) Equals(other interface{}) bool {
	return f == other
}

func (f *Future) GetInfo() *ObjectInfo {
	return nil
}

func (f *Future) GetType() *Type {
	return TYPE.Future
}

func (f *Future) Hash() uint32 {
	return HashPtr(uintptr(unsafe.Pointer(f)))
}

func (f *Future) WithInfo(info *ObjectInfo) Object {
	return f
}

func MakeFuture(ch chan FutureResult) *Future {
	return &Future{ch: ch}
}

func ExtractFuture(args []Object, index int) *Future {
	return EnsureFuture(args, index)
}

func (d *Future) Deref() Object {
	if !d.isDelivered {
		d.result = <-d.ch
		d.isDelivered = true
	}
	if d.result.err != nil {
		panic(d.result.err)
	}
	return d.result.value
}

func ExecFuture(f func() Object) *Future {
	ch := make(chan FutureResult)
	go func() {

		defer func() {
			if r := recover(); r != nil {
				switch r := r.(type) {
				case Error:
					ch <- MakeFutureResult(NIL, r)
				default:
					panic(r)
				}
			}
		}()

		res := f()
		ch <- MakeFutureResult(res, nil)
	}()
	return MakeFuture(ch)
}
