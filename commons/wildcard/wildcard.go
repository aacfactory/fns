package wildcard

import "strings"

func Match(pattern string, target string) (ok bool) {
	ok = New(pattern).Match(target)
	return
}

func New(pattern string) (w *Wildcard) {
	if pattern == "*" {
		w = &Wildcard{
			prefix: "",
			suffix: "",
		}
		return
	}
	idx := strings.IndexByte(pattern, '*')
	if idx < 0 {
		w = &Wildcard{
			prefix: pattern,
			suffix: "",
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
	prefix string
	suffix string
}

func (w *Wildcard) Match(s string) bool {
	if w.suffix == "" {
		return w.prefix == s
	}
	return len(s) >= len(w.prefix)+len(w.suffix) && strings.HasPrefix(s, w.prefix) && strings.HasSuffix(s, w.suffix)
}
