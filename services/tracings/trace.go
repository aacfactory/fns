package tracings

import (
	"github.com/aacfactory/fns/commons/bytex"
	"time"
)

func New(id []byte) *Trace {
	return &Trace{
		Id:      bytex.ToString(id),
		Span:    nil,
		current: nil,
	}
}

// todo
// use  middleware and listen wrap
type Trace struct {
	Id      string `json:"id"`
	Span    *Span  `json:"span"`
	current *Span
}

func (trace *Trace) Begin(pid []byte, endpoint []byte, fn []byte, tags ...string) {
	if trace.current != nil && trace.current.Id == bytex.ToString(pid) {
		return
	}
	current := &Span{
		Id:       bytex.ToString(pid),
		Endpoint: bytex.ToString(endpoint),
		Fn:       bytex.ToString(fn),
		Begin:    time.Now(),
		Waited:   time.Time{},
		End:      time.Time{},
		Tags:     make(map[string]string),
		Children: nil,
		parent:   nil,
	}
	current.setTags(tags)
	if trace.Span == nil {
		trace.Span = current
		trace.current = current
		return
	}
	parent := trace.current
	if parent.Children == nil {
		parent.Children = make([]*Span, 0, 1)
	}
	parent.Children = append(parent.Children, current)
	current.parent = parent
	trace.current = current
}

func (trace *Trace) Waited(tags ...string) {
	if trace.current == nil {
		return
	}
	trace.current.Waited = time.Now()
	trace.current.setTags(tags)
	return
}

func (trace *Trace) Tagging(tags ...string) {
	if trace.current == nil {
		return
	}
	trace.current.setTags(tags)
	return
}

func (trace *Trace) Finish(tags ...string) {
	if trace.current == nil {
		return
	}
	if trace.current.Waited.IsZero() {
		trace.current.Waited = trace.current.Begin
	}
	trace.current.End = time.Now()
	trace.current.setTags(tags)
	if trace.current.parent != nil {
		trace.current = trace.current.parent
	}
}

func (trace *Trace) Mount(child *Span) {
	if trace.current == nil {
		return
	}
	if child == nil {
		return
	}
	if trace.current.Children == nil {
		trace.current.Children = make([]*Span, 0, 1)
	}
	trace.current.Children = append(trace.current.Children, child)
}
