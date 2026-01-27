package core

import (
	"bytes"
	"sync"
)

// bufferPool reuses bytes.Buffer for building strings in hot paths
// (stacktrace, Callstack.String, SeqToString).
var bufferPool = sync.Pool{
	New: func() interface{} { return &bytes.Buffer{} },
}

func getBuffer() *bytes.Buffer {
	b := bufferPool.Get().(*bytes.Buffer)
	b.Reset()
	return b
}

func putBuffer(b *bytes.Buffer) {
	b.Reset()
	bufferPool.Put(b)
}
