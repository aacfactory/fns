package transports

import (
	"bytes"
	"fmt"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/valyala/bytebufferpool"
	"sort"
)

type Params interface {
	Get(name []byte) []byte
	Set(name []byte, value []byte)
	Remove(name []byte)
	Len() int
	Encode() (p []byte)
}

type param struct {
	key []byte
	val []byte
}

func NewParams() Params {
	pp := make(defaultParams, 0, 1)
	return &pp
}

type defaultParams []param

func (params *defaultParams) Less(i, j int) bool {
	pp := *params
	return bytes.Compare(pp[i].key, pp[j].key) < 0
}

func (params *defaultParams) Swap(i, j int) {
	pp := *params
	pp[i], pp[j] = pp[j], pp[i]
	*params = pp
}

func (params *defaultParams) Get(name []byte) []byte {
	if name == nil {
		return nil
	}
	if len(name) == 0 {
		return nil
	}
	pp := *params
	for _, p := range pp {
		if bytes.Equal(p.key, name) {
			return p.val
		}
	}
	return nil
}

func (params *defaultParams) Set(name []byte, value []byte) {
	if name == nil || value == nil {
		return
	}
	if len(name) == 0 {
		return
	}
	pp := *params
	for _, p := range pp {
		if bytes.Equal(p.key, name) {
			p.val = value
			*params = pp
			return
		}
	}
	pp = append(pp, param{
		key: name,
		val: value,
	})
	*params = pp
}

func (params *defaultParams) Remove(name []byte) {
	if name == nil {
		return
	}
	if len(name) == 0 {
		return
	}
	pp := *params
	n := -1
	for i, p := range pp {
		if bytes.Equal(p.key, name) {
			n = i
			break
		}
	}
	if n == -1 {
		return
	}
	pp = append(pp[:n], pp[n+1:]...)
	*params = pp
}

func (params *defaultParams) Len() int {
	return len(*params)
}

func (params *defaultParams) Encode() []byte {
	if params.Len() == 0 {
		return nil
	}
	sort.Sort(params)
	pp := *params
	buf := bytebufferpool.Get()
	for _, p := range pp {
		_, _ = buf.WriteString(fmt.Sprintf("&%s=%s", bytex.ToString(p.key), bytex.ToString(p.val)))
	}
	p := buf.Bytes()[1:]
	bytebufferpool.Put(buf)
	return p
}
