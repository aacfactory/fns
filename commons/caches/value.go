package caches

import (
	"encoding/binary"
	"github.com/valyala/bytebufferpool"
	"time"
)

func MakeKVS(k []byte, p []byte, ttl time.Duration, hash Hash) (kvs KVS) {
	kvs = make([]KV, 0, 1)
	kLen := len(k)
	pLen := len(p)
	if 4+kLen+10+pLen < chunkSize {
		//normal
		v := make([]byte, 10+pLen)
		v[0] = 1
		v[1] = 1
		if ttl > 0 {
			binary.BigEndian.PutUint64(v[2:10], uint64(time.Now().Add(ttl).UnixNano()))
		} else {
			binary.BigEndian.PutUint64(v[2:10], 0)
		}
		copy(v[10:], p)
		kvs = append(kvs, KV{
			k: k,
			v: v,
			h: hash.Sum(k),
		})
		return
	}
	// big key
	// first
	p0Len := chunkSize - 4 - kLen - 10 - 1
	p0 := p[0:p0Len]
	v0 := make([]byte, 10+p0Len)
	v0[0] = 1
	if ttl > 0 {
		binary.BigEndian.PutUint64(v0[2:10], uint64(time.Now().Add(ttl).UnixNano()))
	} else {
		binary.BigEndian.PutUint64(v0[2:10], 0)
	}
	copy(v0[10:], p0)
	kvs = append(kvs, KV{
		k: k,
		v: v0,
		h: hash.Sum(k),
	})
	// next
	p = p[p0Len:]
	maxChunkValueLen := chunkSize - 4 - kLen - 10 - 1
	pos := uint64(2)
	stop := false
	for {
		chunkValueLen := 0
		if npLen := len(p); npLen <= maxChunkValueLen {
			chunkValueLen = npLen
			stop = true
		} else {
			chunkValueLen = maxChunkValueLen
		}
		nk := make([]byte, kLen+8)
		copy(nk, k)
		binary.BigEndian.PutUint64(nk[kLen:], pos)

		np := make([]byte, 2+chunkValueLen)
		np[0] = byte(pos)
		copy(np[2:], p[0:chunkValueLen])
		kvs = append(kvs, KV{
			k: nk,
			v: np,
			h: hash.Sum(nk),
		})
		if stop {
			break
		}
		pos++
		p = p[chunkValueLen:]
	}

	kvLen := len(kvs)
	for _, kv := range kvs {
		kv.v[1] = byte(kvLen)
	}
	return
}

func MakeValue(p []byte) Value {
	pLen := len(p)
	v := make([]byte, 10+pLen)
	v[0] = 1
	v[1] = 1
	copy(v[10:], p)
	return v
}

// Value
// [1]pos [1]size, [8]deadline, [...]value
type Value []byte

func (v Value) Pos() int {
	return int(v[0])
}

func (v Value) Size() int {
	return int(v[1])
}

func (v Value) Normal() bool {
	return v.Size() == 1
}

func (v Value) BigKey() bool {
	return v.Size() > 1
}

func (v Value) SetDeadline(deadline time.Time) {
	if v.Pos() == 1 {
		n := deadline.UnixNano()
		binary.BigEndian.PutUint64(v[2:10], uint64(n))
	}
}

func (v Value) Deadline() time.Time {
	if v.Pos() == 1 {
		n := binary.BigEndian.Uint64(v[2:10])
		if n == 0 {
			return time.Time{}
		}
		return time.Unix(0, int64(n))
	}
	return time.Time{}
}

func (v Value) Bytes() (p []byte) {
	if v.Pos() == 1 {
		p = v[10:]
	} else {
		p = v[2:]
	}
	return
}

type KV struct {
	k []byte
	v Value
	h uint64
}

func (kv KV) Key() []byte {
	return kv.k
}

func (kv KV) Value() Value {
	return kv.v
}

func (kv KV) Hash() uint64 {
	return kv.h
}

type KVS []KV

func (kvs KVS) Value() (p []byte) {
	b := bytebufferpool.Get()
	for _, kv := range kvs {
		_, _ = b.Write(kv.v.Bytes())
	}
	p = b.Bytes()
	bytebufferpool.Put(b)
	return
}

func (kvs KVS) Deadline() time.Time {
	return kvs[0].v.Deadline()
}
