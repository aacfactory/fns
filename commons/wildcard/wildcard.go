package wildcard

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
)

func Match(pattern []byte, target []byte) (ok bool) {
	ok = New(pattern).Match(target)
	return
}

func New(pattern []byte) (w *Wildcard) {
	if len(pattern) == 1 && pattern[0] == '*' {
		w = &Wildcard{
			prefix: nil,
			suffix: nil,
		}
		return
	}
	idx := bytes.IndexByte(pattern, '*')
	if idx < 0 {
		w = &Wildcard{
			prefix: pattern,
			suffix: nil,
		}
		return
	}
	w = &Wildcard{
		prefix: pattern[0:idx],
		suffix: pattern[idx+1:],
	}
	return
}

type Wildcard struct {
	prefix []byte
	suffix []byte
}

func (w *Wildcard) Match(s []byte) bool {
	if len(w.suffix) == 0 {
		return bytex.Equal(w.prefix, s)
	}
	return len(s) >= len(w.prefix)+len(w.suffix) && bytes.HasPrefix(s, w.prefix) && bytes.HasSuffix(s, w.suffix)
}
