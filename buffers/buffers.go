package buffers

import (
	"bytes"
	"strings"
	"sync"
)

var bytesBuf = &sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

var stringsBuf = &sync.Pool{
	New: func() interface{} { return new(strings.Builder) },
}

var sliceBuffer = &sync.Pool{
	New: func() interface{} { return make([]byte, 1024) },
}

// GetBytes returns bytes.Buffer from the pool
func GetBytes() *bytes.Buffer {
	return bytesBuf.Get().(*bytes.Buffer)
}

// PutBytes resets buf and puts it back info the pool
func PutBytes(buf *bytes.Buffer) {
	buf.Reset()
	bytesBuf.Put(buf)
}

// GetString returns strings.Builder from the pool
func GetString() *strings.Builder {
	return stringsBuf.Get().(*strings.Builder)
}

// PutString resets buf and puts it back info the pool
func PutString(buf *strings.Builder) {
	buf.Reset()
	stringsBuf.Put(buf)
}

// GetSlice returns []byte of size 1024 from the pool
func GetSlice() []byte {
	return sliceBuffer.Get().([]byte)
}

// PutSlice resets buf and puts it back info the pool
func PutSlice(buf []byte) {
	stringsBuf.Put(buf[:0])
}
