package objects

import "github.com/aacfactory/fns/commons/bytex"

type Entry struct {
	key []byte
	val any
}

type Entries []Entry

func (entries *Entries) Get(key []byte) (val any) {
	s := *entries
	for _, entry := range s {
		if bytex.Equal(key, entry.key) {
			val = entry.val
			return
		}
	}
	return
}

func (entries *Entries) Set(key []byte, val any) {
	s := *entries
	for _, entry := range s {
		if bytex.Equal(key, entry.key) {
			entry.val = val
			*entries = s
			return
		}
	}
	s = append(s, Entry{
		key: key,
		val: val,
	})
	*entries = s
	return
}

func (entries *Entries) Remove(key []byte) {
	s := *entries
	n := -1
	for i, entry := range s {
		if bytex.Equal(key, entry.key) {
			n = i
			break
		}
	}
	if n > -1 {
		s = append(s[:n], s[n+1:]...)
		*entries = s
	}
	return
}

func (entries *Entries) Foreach(fn func(key []byte, value any)) {
	s := *entries
	for _, entry := range s {
		fn(entry.key, entry.val)
	}
}

func (entries *Entries) Len() int {
	return len(*entries)
}
