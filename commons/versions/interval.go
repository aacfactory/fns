package versions

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/wildcard"
	"github.com/valyala/bytebufferpool"
	"strings"
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

// ParseInterval
// left:right
func ParseInterval(source string) (interval Interval, err error) {
	ss := strings.Split(source, ":")
	n := len(ss)
	if n == 0 {
		interval = Interval{Origin(), Latest()}
		return
	}
	if n == 1 {
		ver, parseErr := Parse(ss[0])
		if parseErr != nil {
			err = errors.Warning("fns: parse interval failed").WithMeta("source", source).WithCause(parseErr)
			return
		}
		interval = Interval{ver, Latest()}
		return
	}
	if n == 2 {
		left, leftErr := Parse(ss[0])
		if leftErr != nil {
			err = errors.Warning("fns: parse interval failed").WithMeta("source", source).WithCause(leftErr)
			return
		}
		right, rightErr := Parse(ss[1])
		if rightErr != nil {
			err = errors.Warning("fns: parse interval failed").WithMeta("source", source).WithCause(rightErr)
			return
		}
		if right.LessThan(left) {
			err = errors.Warning("fns: parse interval failed").WithMeta("source", source).WithCause(fmt.Errorf("invalid interval"))
			return
		}
		interval = Interval{left, right}
		return
	}
	err = errors.Warning("fns: parse interval failed").WithMeta("source", source).WithCause(fmt.Errorf("invalid interval"))
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

func (intervals Intervals) Bytes() []byte {
	if len(intervals) == 0 {
		return []byte{}
	}
	p := bytebufferpool.Get()
	defer bytebufferpool.Put(p)
	for key, interval := range intervals {
		_, _ = p.Write([]byte{',', ' '})
		_, _ = p.Write(bytex.FromString(key))
		_, _ = p.Write([]byte{'='})
		_, _ = p.Write(bytex.FromString(interval.String()))
	}
	return p.Bytes()[2:]
}

func (intervals Intervals) String() string {
	return bytex.ToString(intervals.Bytes())
}

// ParseIntervals
// key=left:right, ...
func ParseIntervals(source string) (intervals Intervals, err error) {
	ss := strings.Split(source, ",")
	for _, s := range ss {
		s = strings.TrimSpace(s)
		idx := strings.IndexByte(s, '=')
		if idx < 1 {
			err = errors.Warning("fns: parse intervals failed").WithMeta("source", source).WithCause(fmt.Errorf("invalid intervals"))
			return
		}
		key := strings.TrimSpace(s[0:idx])
		interval, parseErr := ParseInterval(s[idx+1:])
		if parseErr != nil {
			err = errors.Warning("fns: parse intervals failed").WithMeta("source", source).WithCause(parseErr)
			return
		}
		if intervals == nil {
			intervals = make(Intervals)
		}
		intervals[key] = interval
	}
	return
}

func AllowAllIntervals() Intervals {
	return Intervals{
		"*": Interval{Origin(), Latest()},
	}
}
