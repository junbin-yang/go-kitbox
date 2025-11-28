package binpack

import "sync"

// BufferPool 字节缓冲池
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool 创建指定大小的 buffer 池
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get 从池中获取 buffer
func (p *BufferPool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put 将 buffer 放回池中
func (p *BufferPool) Put(buf *[]byte) {
	p.pool.Put(*buf)
}

// DefaultBufferPool 默认 buffer 池（1KB）
var DefaultBufferPool = NewBufferPool(1024)
