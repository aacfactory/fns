/*
 * Copyright 2021 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package caches

import (
	"encoding/binary"
	"fmt"
	"github.com/valyala/bytebufferpool"
	"sync"
	"time"
)

const (
	defaultMaxBytes        = 64 << (10 * 2)
	bucketsCount           = 512
	chunkSize              = 1 << 16
	bucketSizeBits         = 40
	genSizeBits            = 64 - bucketSizeBits
	maxGen                 = 1<<genSizeBits - 1
	maxBucketSize   uint64 = 1 << bucketSizeBits
	maxKeyLen              = chunkSize - 4 - 2 - 8 - 8
)

var (
	ErrTooBigKey  = fmt.Errorf("key was too big, must not be greater than 63k")
	ErrInvalidKey = fmt.Errorf("key is invalid")
)

func New(maxBytes uint64) (cache *Cache) {
	cache = NewWithHash(maxBytes, MemHash{})
	return
}

func NewWithHash(maxBytes uint64, h Hash) (cache *Cache) {
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	if maxBytes >= maxBucketSize {
		maxBytes = maxBucketSize - 1<<30
	}
	cache = &Cache{
		locker:       sync.RWMutex{},
		maxItemBytes: maxBytes / 2,
		buckets:      [bucketsCount]bucket{},
		hash:         h,
	}
	maxBucketBytes := (maxBytes + bucketsCount - 1) / bucketsCount
	for i := range cache.buckets[:] {
		cache.buckets[i].create(maxBucketBytes, cache.evict)
	}
	return
}

type Cache struct {
	locker       sync.RWMutex
	maxItemBytes uint64
	buckets      [bucketsCount]bucket
	hash         Hash
}

func (c *Cache) canSet(k []byte, v []byte) (ok bool) {
	vLen := len(v)
	if vLen == 0 {
		vLen = 8
	}
	itemLen := uint64(len(k) + vLen + 4 + 10)
	ok = itemLen < c.maxItemBytes
	return
}

func (c *Cache) set(k []byte, v []byte, h uint64) {
	idx := h % bucketsCount
	c.buckets[idx].Set(k, v, h)
}

func (c *Cache) get(k []byte) (p []byte, found bool) {
	p = make([]byte, 0, 8)
	h := c.hash.Sum(k)
	idx := h % bucketsCount
	p, found = c.buckets[idx].Get(p, k, h, true)
	return
}

func (c *Cache) contains(k []byte) (ok bool) {
	h := c.hash.Sum(k)
	idx := h % bucketsCount
	_, ok = c.buckets[idx].Get(nil, k, h, false)
	return
}

func (c *Cache) SetWithTTL(k []byte, v []byte, ttl time.Duration) (err error) {
	if len(k) == 0 || len(v) == 0 {
		err = ErrInvalidKey
		return
	}

	if !c.canSet(k, v) {
		err = ErrTooBigKey
		return
	}
	c.locker.Lock()
	kvs := MakeKVS(k, v, ttl, c.hash)
	for _, kv := range kvs {
		c.set(kv.k, kv.v, kv.h)
	}
	c.locker.Unlock()
	return
}

func (c *Cache) Set(k []byte, v []byte) (err error) {
	err = c.SetWithTTL(k, v, 0)
	return
}

func (c *Cache) Get(k []byte) (p []byte, ok bool) {
	c.locker.RLock()
	// first
	dst, found := c.get(k)
	if !found {
		c.locker.RUnlock()
		return
	}
	v := Value(dst)
	if v.Pos() > 1 {
		c.locker.RUnlock()
		return
	}
	if deadline := v.Deadline(); !deadline.IsZero() {
		if deadline.Before(time.Now()) {
			c.locker.RUnlock()
			return
		}
	}
	size := v.Size()
	if size == 1 {
		p = v.Bytes()
		ok = true
		c.locker.RUnlock()
		return
	}

	// big key
	kLen := len(k)
	nkLen := kLen + 8
	b := bytebufferpool.Get()
	_, _ = b.Write(v.Bytes())
	for i := 2; i <= size; i++ {
		nk := make([]byte, nkLen)
		copy(nk, k)
		binary.BigEndian.PutUint64(nk[kLen:], uint64(i))
		np, has := c.get(nk)
		if !has {
			return
		}
		_, _ = b.Write(Value(np).Bytes())
	}
	p = b.Bytes()
	bytebufferpool.Put(b)
	ok = true
	c.locker.RUnlock()
	return
}

func (c *Cache) Contains(k []byte) bool {
	c.locker.RLock()
	defer c.locker.RUnlock()
	return c.contains(k)
}

func (c *Cache) Expire(k []byte, ttl time.Duration) {
	c.locker.Lock()
	dst, found := c.get(k)
	if !found {
		c.locker.Unlock()
		return
	}
	v := Value(dst)
	if v.Pos() > 1 {
		c.locker.Unlock()
		return
	}
	v.SetDeadline(time.Now().Add(ttl))
	c.locker.Unlock()
	return
}

func (c *Cache) Incr(k []byte, delta int64) (n int64, err error) {
	if len(k) == 0 {
		err = ErrInvalidKey
		return
	}
	if !c.canSet(k, nil) {
		err = ErrTooBigKey
		return
	}
	c.locker.Lock()

	p, found := c.get(k)
	if found && len(p) == 18 {
		n = int64(binary.BigEndian.Uint64(p[10:]))
		n += delta
		binary.BigEndian.PutUint64(p[10:], uint64(n))
		c.set(k, p, c.hash.Sum(k))
		c.locker.Unlock()
		return
	}

	p = make([]byte, 18)
	p[0] = 1
	p[1] = 1
	binary.BigEndian.PutUint64(p[10:], uint64(delta))
	c.set(k, p, c.hash.Sum(k))
	n = delta
	c.locker.Unlock()
	return
}

func (c *Cache) Remove(k []byte) {
	if len(k) > maxKeyLen {
		return
	}
	c.locker.Lock()
	h := c.hash.Sum(k)
	idx := h % bucketsCount
	c.buckets[idx].Remove(h)
	c.locker.Unlock()
}

func (c *Cache) evict(_ uint64) {

}
