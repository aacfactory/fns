package tracings

import "time"

type Span struct {
	Id       string            `json:"id"`
	Endpoint string            `json:"endpoint"`
	Fn       string            `json:"fn"`
	Begin    time.Time         `json:"begin"`
	Waited   time.Time         `json:"waited"`
	End      time.Time         `json:"end"`
	Tags     map[string]string `json:"tags"`
	Children []*Span           `json:"children"`
	parent   *Span
}

func (span *Span) setTags(tags []string) {
	n := len(tags)
	if n == 0 {
		return
	}
	if n%2 != 0 {
		return
	}
	for i := 0; i < n; i += 2 {
		k := tags[i]
		v := tags[i+1]
		span.Tags[k] = v
	}
}

func (span *Span) mountChildrenParent() {
	for _, child := range span.Children {
		if child.parent == nil {
			child.parent = span
		}
		child.mountChildrenParent()
	}
}
