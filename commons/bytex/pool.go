package bytex

import (
	"sync"
)

var (
	bufPool = sync.Pool{New: func() any {
		return make([]byte, 4096)
	}}
)

func Acquire4KBuffer() []byte {
	x := bufPool.Get()
	if x == nil {
		return make([]byte, 4096)
	}
	return x.([]byte)
}

func Release4KBuffer(buf []byte) {
	bufPool.Put(buf)
}
