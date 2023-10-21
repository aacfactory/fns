package versions

import (
	"fmt"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/wildcard"
	"github.com/valyala/bytebufferpool"
)

type Interval []Version

func (interval Interval) Accept(target Version) (ok bool) {
	n := len(interval)
	if n == 0 { // [v0.0.0, latest)
		ok = true
		return
	}
	if n == 1 { // [{left}, latest)
		ok = target.Between(interval[0], Latest())
		return
	}
	// [{left}, {right}})
	ok = target.Between(interval[0], interval[1])
	return
}

func (interval Interval) String() string {
	n := len(interval)
	if n == 0 {
		return "[v0.0.0, latest)"
	}
	if n == 1 {
		return fmt.Sprintf("[%s, latest)", interval[0].String())
	}
	return fmt.Sprintf("[%s, %s)", interval[0].String(), interval[1].String())
}

func ParseInterval(s string) (interval Interval, err error) {

	return
}

type Intervals map[string]Interval

func (intervals Intervals) Accept(pattern string, target Version) (ok bool) {
	for key, interval := range intervals {
		if !wildcard.Match(pattern, key) {
			continue
		}
		ok = interval.Accept(target)
		if ok {
			break
		}
	}
	return
}

func (intervals Intervals) String() string {
	p := bytebufferpool.Get()
	defer bytebufferpool.Put(p)
	for key, interval := range intervals {
		_, _ = p.Write([]byte{',', ' '})
		_, _ = p.Write(bytex.FromString(key))
		_, _ = p.Write([]byte{'='})
		_, _ = p.Write(bytex.FromString(interval.String()))
	}
	if p.Len() > 0 {
		return bytex.ToString(p.Bytes()[2:])
	}
	return ""
}
