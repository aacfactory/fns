package services

import (
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services/documents"
	"sort"
	"strings"
	"unsafe"
)

type Endpoint interface {
	Name() (name string)
	Internal() (ok bool)
	Document() (document documents.Document)
	Functions() (functions Fns)
	Shutdown(ctx context.Context)
}

type EndpointInfo struct {
	Id        string             `json:"id"`
	Name      string             `json:"name"`
	Version   versions.Version   `json:"version"`
	Internal  bool               `json:"internal"`
	Functions FnInfos            `json:"functions"`
	Document  documents.Document `json:"document"`
}

type EndpointInfos []EndpointInfo

func (infos EndpointInfos) Len() int {
	return len(infos)
}

func (infos EndpointInfos) Less(i, j int) bool {
	x := infos[i]
	y := infos[j]
	n := strings.Compare(x.Name, y.Name)
	if n < 0 {
		return true
	} else if n == 0 {
		return x.Version.LessThan(y.Version)
	} else {
		return false
	}
}

func (infos EndpointInfos) Swap(i, j int) {
	infos[i], infos[j] = infos[j], infos[i]
}

type EndpointGetOption func(options *EndpointGetOptions)

type EndpointGetOptions struct {
	id              []byte
	requestVersions versions.Intervals
}

func (options EndpointGetOptions) Id() []byte {
	return options.id
}

func (options EndpointGetOptions) Versions() versions.Intervals {
	return options.requestVersions
}

func EndpointId(id []byte) EndpointGetOption {
	return func(options *EndpointGetOptions) {
		options.id = id
		return
	}
}

func EndpointVersions(requestVersions versions.Intervals) EndpointGetOption {
	return func(options *EndpointGetOptions) {
		options.requestVersions = requestVersions
		return
	}
}

type Endpoints interface {
	Info() (infos []EndpointInfo)
	Get(ctx context.Context, name []byte, options ...EndpointGetOption) (endpoint Endpoint, has bool)
	Request(ctx context.Context, name []byte, fn []byte, param interface{}, options ...RequestOption) (response Response, err error)
}

type Deployed []Endpoint

func (s Deployed) Len() int {
	return len(s)
}

func (s Deployed) Less(i, j int) bool {
	return strings.Compare(s[i].Name(), s[j].Name()) < 0
}

func (s Deployed) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Deployed) Add(v Endpoint) Deployed {
	ss := append(s, v)
	sort.Sort(ss)
	return ss
}

func (s Deployed) Find(name []byte) (v Endpoint, found bool) {
	ns := unsafe.String(unsafe.SliceData(name), len(name))
	n := s.Len()
	if n < 65 {
		for _, endpoint := range s {
			if endpoint.Name() == ns {
				v = endpoint
				found = true
				break
			}
		}
		return
	}
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)
		if strings.Compare(s[h].Name(), ns) < 0 {
			i = h + 1
		} else {
			j = h
		}
	}
	found = i < n && s[i].Name() == ns
	if found {
		v = s[i]
	}
	return
}
