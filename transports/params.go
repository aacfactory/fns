package transports

import (
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

func NewParams() Params {
	return make(defaultParams)
}

type defaultParams map[string][]byte

func (params defaultParams) Get(name []byte) []byte {
	if name == nil {
		return nil
	}
	if len(name) == 0 {
		return nil
	}
	value, has := params[bytex.ToString(name)]
	if !has {
		return nil
	}
	return value
}

func (params defaultParams) Set(name []byte, value []byte) {
	if name == nil || value == nil {
		return
	}
	if len(name) == 0 {
		return
	}
	params[bytex.ToString(name)] = value
}

func (params defaultParams) Remove(name []byte) {
	if name == nil {
		return
	}
	if len(name) == 0 {
		return
	}
	delete(params, bytex.ToString(name))
}

func (params defaultParams) Len() int {
	return len(params)
}

func (params defaultParams) Encode() []byte {
	size := len(params)
	if size == 0 {
		return nil
	}
	names := make([]string, 0, size)
	for name := range params {
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	buf := bytebufferpool.Get()
	for _, name := range names {
		_, _ = buf.WriteString(fmt.Sprintf("&%s=%s", name, bytex.ToString(params[name])))
	}
	p := buf.Bytes()[1:]
	bytebufferpool.Put(buf)
	return p
}
