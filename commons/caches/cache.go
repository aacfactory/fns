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
	"github.com/cespare/xxhash/v2"
	"github.com/valyala/bytebufferpool"
	"time"
)

const (
	defaultMaxBytes        = 64 << (10 * 2)
	bucketsCount           = 512
	chunkSize              = 64 * 1024
	bucketSizeBits         = 40
	genSizeBits            = 64 - bucketSizeBits
	maxGen                 = 1<<genSizeBits - 1
	maxBucketSize   uint64 = 1 << bucketSizeBits
	maxSubValueLen         = chunkSize - 16 - 4 - 1
	maxKeyLen              = chunkSize - 16 - 4 - 1
)

var (
	ErrTooBigKey    = fmt.Errorf("key was too big, must not be greater than 63k")
	ErrInvalidKey   = fmt.Errorf("key is invalid")
	ErrInvalidValue = fmt.Errorf("value content is invalid")
)

func New(maxBytes int) (cache *Cache) {
	cache = NewWithHash(maxBytes, MemHash{})
	return
}

func NewWithHash(maxBytes int, h Hash) (cache *Cache) {
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	cache = &Cache{
		buckets: [512]bucket{},
		bigKeys: [1]bucket{},
		hash:    h,
	}
	maxBucketBytes := uint64((maxBytes + bucketsCount - 1) / bucketsCount)
	for i := range cache.buckets[:] {
		cache.buckets[i].create(maxBucketBytes)
	}
	cache.bigKeys[0].create(maxBucketBytes)
	return
}

type Cache struct {
	buckets [bucketsCount]bucket
	bigKeys [1]bucket
	hash    Hash
}

func (c *Cache) Set(k []byte, v []byte) (err error) {
	err = c.SetWithTTL(k, v, 0)
	return
}

func (c *Cache) SetWithTTL(k []byte, v []byte, ttl time.Duration) (err error) {
	if k == nil || len(k) == 0 {
		err = ErrInvalidKey
		return
	}
	if len(k) > maxKeyLen {
		err = ErrTooBigKey
		return
	}
	if v == nil {
		v = []byte{}
	}
	if len(v) >= 16 && unmarshalUint64(v[8:16]) > 0 {
		err = ErrInvalidValue
		return
	}
	deadline := uint64(0)
	if ttl > 0 {
		deadline = uint64(time.Now().Add(ttl).UnixNano())
	}
	expire := make([]byte, 8)
	binary.BigEndian.PutUint64(expire, deadline)
	v = append(v, expire...)
	h := xxhash.Sum64(k)
	if len(v) > maxSubValueLen {
		c.setBig(k, v)
		c.bigKeys[0].Set(k, k, h)
		return
	}
	idx := h % bucketsCount
	c.buckets[idx].Set(k, v, h)
	return
}

func (c *Cache) set(k []byte, v []byte) {
	h := xxhash.Sum64(k)
	idx := h % bucketsCount
	c.buckets[idx].Set(k, v, h)
}

func (c *Cache) Get(k []byte) ([]byte, bool) {
	dst := make([]byte, 0, 8)
	h := xxhash.Sum64(k)
	idx := h % bucketsCount
	v, has := c.buckets[idx].Get(dst, k, h, true)
	if has && len(v) == 16 && unmarshalUint64(v) > 0 {
		_, hasBigKey := c.bigKeys[0].Get(nil, k, xxhash.Sum64(k), false)
		if !hasBigKey {
			return nil, false
		}
		dst = make([]byte, 0, 8)
		v = c.getBig(dst, k)
		return c.checkExpire(k, v)
	}
	return c.checkExpire(k, v)
}

func (c *Cache) checkExpire(k []byte, v []byte) ([]byte, bool) {
	vLen := len(v)
	if vLen < 8 {
		c.Remove(k)
		return nil, false
	}
	deadlinePos := vLen - 8
	deadline := binary.BigEndian.Uint64(v[vLen-8:])
	if deadline == 0 {
		return v[0:deadlinePos], true
	}
	if deadline < uint64(time.Now().UnixNano()) {
		c.Remove(k)
		return nil, false
	}
	return v[0:deadlinePos], true
}

func (c *Cache) get(dst []byte, k []byte) ([]byte, bool) {
	h := xxhash.Sum64(k)
	idx := h % bucketsCount
	has := false
	dst, has = c.buckets[idx].Get(dst, k, h, true)
	return dst, has
}

func (c *Cache) Exist(k []byte) bool {
	h := xxhash.Sum64(k)
	idx := h % bucketsCount
	_, ok := c.buckets[idx].Get(nil, k, h, false)
	return ok
}

func (c *Cache) Remove(k []byte) {
	if len(k) > maxKeyLen {
		return
	}
	h := xxhash.Sum64(k)
	idx := h % bucketsCount
	c.buckets[idx].Remove(h)
	c.bigKeys[0].Remove(h)
}

func (c *Cache) setBig(k []byte, v []byte) {
	if len(k) > maxKeyLen {
		return
	}
	valueLen := len(v)
	valueHash := xxhash.Sum64(v)

	subKey := bytebufferpool.Get()
	var i uint64
	for len(v) > 0 {
		subKey.B = marshalUint64(subKey.B[:0], valueHash)
		subKey.B = marshalUint64(subKey.B, i)
		i++
		subValueLen := maxSubValueLen
		if len(v) < subValueLen {
			subValueLen = len(v)
		}
		subValue := v[:subValueLen]
		v = v[subValueLen:]
		c.set(subKey.B, subValue)
	}

	subKey.B = marshalUint64(subKey.B[:0], valueHash)
	subKey.B = marshalUint64(subKey.B, uint64(valueLen))
	c.set(k, subKey.B)
	bytebufferpool.Put(subKey)
}

func (c *Cache) getBig(dst, k []byte) (r []byte) {
	subKey := bytebufferpool.Get()
	dstWasNil := dst == nil
	defer func() {
		bytebufferpool.Put(subKey)
		if len(r) == 0 && dstWasNil {
			r = nil
		}
	}()

	subKey.B, _ = c.get(subKey.B[:0], k)
	if len(subKey.B) == 0 {
		return dst
	}
	if len(subKey.B) != 16 {
		return dst
	}
	valueHash := unmarshalUint64(subKey.B)
	valueLen := unmarshalUint64(subKey.B[8:])

	dstLen := len(dst)
	if n := dstLen + int(valueLen) - cap(dst); n > 0 {
		dst = append(dst[:cap(dst)], make([]byte, n)...)
	}
	dst = dst[:dstLen]
	var i uint64
	for uint64(len(dst)-dstLen) < valueLen {
		subKey.B = marshalUint64(subKey.B[:0], valueHash)
		subKey.B = marshalUint64(subKey.B, i)
		i++
		dstNew, _ := c.get(dst, subKey.B)
		if len(dstNew) == len(dst) {
			return dst[:dstLen]
		}
		dst = dstNew
	}
	v := dst[dstLen:]
	if uint64(len(v)) != valueLen {
		return dst[:dstLen]
	}
	h := xxhash.Sum64(v)
	if h != valueHash {
		return dst[:dstLen]
	}
	return dst
}

func marshalUint64(dst []byte, u uint64) []byte {
	return append(dst, byte(u>>56), byte(u>>48), byte(u>>40), byte(u>>32), byte(u>>24), byte(u>>16), byte(u>>8), byte(u))
}

func unmarshalUint64(src []byte) uint64 {
	_ = src[7]
	return uint64(src[0])<<56 | uint64(src[1])<<48 | uint64(src[2])<<40 | uint64(src[3])<<32 | uint64(src[4])<<24 | uint64(src[5])<<16 | uint64(src[6])<<8 | uint64(src[7])
}
