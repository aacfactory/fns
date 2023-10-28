package objects

import "bytes"

type BEntry struct {
	Key    []byte
	Values [][]byte
}

func NewBMap() BMap {
	return make(BMap, 0, 1)
}

type BMap []BEntry

func (b *BMap) Set(key []byte, val []byte) {
	s := *b
	for i, entry := range s {
		if bytes.Equal(entry.Key, key) {
			s[i] = BEntry{
				Key:    key,
				Values: [][]byte{val},
			}
			*b = s
			return
		}
	}
	s = append(s, BEntry{
		Key:    key,
		Values: [][]byte{val},
	})
	*b = s
}

func (b *BMap) Add(key []byte, val []byte) {
	s := *b
	for _, entry := range s {
		if bytes.Equal(entry.Key, key) {
			entry.Values = append(entry.Values, val)
			*b = s
			return
		}
	}
	s = append(s, BEntry{
		Key:    key,
		Values: [][]byte{val},
	})
	*b = s
}

func (b *BMap) Get(key []byte) ([]byte, bool) {
	s := *b
	for _, entry := range s {
		if bytes.Equal(entry.Key, key) {
			return entry.Values[0], true
		}
	}
	return nil, false
}

func (b *BMap) Values(key []byte) ([][]byte, bool) {
	s := *b
	for _, entry := range s {
		if bytes.Equal(entry.Key, key) {
			return entry.Values, true
		}
	}
	return nil, false
}

func (b *BMap) Remove(key []byte) {
	s := *b
	n := -1
	for i, entry := range s {
		if bytes.Equal(entry.Key, key) {
			n = i
			break
		}
	}
	if n > -1 {
		s = append(s[:n], s[n+1:]...)
		*b = s
	}
}

func (b *BMap) Foreach(fn func(key []byte, values [][]byte)) {
	s := *b
	for _, entry := range s {
		fn(entry.Key, entry.Values)
	}
}

func (b *BMap) Len() int {
	return len(*b)
}
