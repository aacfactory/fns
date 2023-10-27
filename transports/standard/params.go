package standard

import (
	"github.com/aacfactory/fns/commons/bytex"
	"net/url"
)

type Params struct {
	values url.Values
}

func (params *Params) Get(name []byte) []byte {
	return bytex.FromString(params.values.Get(bytex.ToString(name)))
}

func (params *Params) Set(name []byte, value []byte) {
	params.values.Set(bytex.ToString(name), bytex.ToString(value))
}

func (params *Params) Remove(name []byte) {
	params.values.Del(bytex.ToString(name))
}

func (params *Params) Len() int {
	return len(params.values)
}

func (params *Params) Encode() (p []byte) {
	p = bytex.FromString(params.values.Encode())
	return
}
