package main

import (
	"context"
	"io"
	"math/rand"
	"strconv"
	"testing"
)

var _ io.ReadWriteCloser = noopReadWriteCloser{}

type noopReadWriteCloser struct{}

func (noopReadWriteCloser) Close() error {
	return nil
}

func (noopReadWriteCloser) Read(_ []byte) (int, error) {
	return 0, nil
}

func (noopReadWriteCloser) Write(_ []byte) (int, error) {
	return 0, nil
}

func BenchmarkCache_Rand(b *testing.B) {
	var o readWriterCloserCacheOption
	o.defaults()
	o.on.store(true)
	o.max.store(8192)
	c := new(readWriterCloserCache)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if ok := c.init(ctx, o); !ok {
		b.Fatal("failed to start cache")
	}
	defer c.Close()
	var n noopReadWriteCloser
	c.opener = func(path string) (io.ReadWriteCloser, error) {
		return n, nil
	}
	trace := make([]string, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = strconv.FormatInt(rand.Int63()%32768, 10)
	}
	var hit, miss int
	var ok bool
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			c.Put(trace[i])
		} else {
			_, ok = c.Get(trace[i])
			if ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}
