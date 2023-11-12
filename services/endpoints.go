package services

import (
	"context"
	"github.com/aacfactory/fns/services/documents"
	"sort"
	"strings"
)

type Handler interface {
	Handle(ctx Request) (v interface{}, err error)
}

type HandlerFunc func(ctx Request) (v interface{}, err error)

func (f HandlerFunc) Handle(ctx Request) (v interface{}, err error) {
	v, err = f(ctx)
	return
}

type Endpoint interface {
	Name() (name string)
	Internal() (ok bool)
	Document() (document documents.Document)
	Handler
	Shutdown(ctx context.Context)
}

type Endpoints interface {
	Request(ctx context.Context, name []byte, fn []byte, argument Argument, options ...RequestOption) (response Response, err error)
}

type HostEndpoints interface {
	Endpoints
	Documents() (v documents.Documents)
}

type SortEndpoints []Endpoint

func (s SortEndpoints) Len() int {
	return len(s)
}

func (s SortEndpoints) Less(i, j int) bool {
	return strings.Compare(s[i].Name(), s[j].Name()) < 0
}

func (s SortEndpoints) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SortEndpoints) Add(v Endpoint) SortEndpoints {
	ss := append(s, v)
	sort.Sort(ss)
	return ss
}

func (s SortEndpoints) Find(name string) (v Endpoint, found bool) {
	n := s.Len()
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)
		if strings.Compare(s[h].Name(), name) < 0 {
			i = h + 1
		} else {
			j = h
		}
	}
	found = i < n && s[i].Name() == name
	if found {
		v = s[i]
	}
	return
}
