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
