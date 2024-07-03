package io

import (
	. "github.com/candid82/joker/core"
	"io"
)

func pipe() Object {
	r, w := io.Pipe()
	res := EmptyVector()
	res = res.Conjoin(MakeIOReader(r))
	res = res.Conjoin(MakeIOWriter(w))
	return res
}

func close(f Object) Nil {
	if c, ok := f.(io.Closer); ok {
		if err := c.Close(); err != nil {
			panic(RT.NewError(err.Error()))
		}
		return NIL
	}
	panic(RT.NewError("Object is not closable: " + f.ToString(false)))
}

func read(r io.Reader, n int) string {
	buf := make([]byte, n)
	cnt, err := r.Read(buf)
	if err != io.EOF {
		PanicOnErr(err)
	}
	return string(buf[:cnt])
}
