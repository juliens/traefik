package httputil

import (
	"sync"
)

const bufferPoolSize = 32 * 1024

func newBufferPool() *bufferPool {
	b := &bufferPool{
		pool: sync.Pool{},
	}

	b.pool.New = func() interface{} {
		b.count++
		return make([]byte, bufferPoolSize)
	}
	return b
}

type bufferPool struct {
	pool  sync.Pool
	count int
}

func (b *bufferPool) Get() []byte {
	return b.pool.Get().([]byte)
}

func (b *bufferPool) Put(bytes []byte) {
	b.pool.Put(bytes)
}
