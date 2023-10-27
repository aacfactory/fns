package fast

import (
	"bytes"
	"github.com/valyala/fasthttp"
)

type Params struct {
	args *fasthttp.Args
}

func (params *Params) Get(name []byte) []byte {
	return params.args.PeekBytes(name)
}

func (params *Params) Set(name []byte, value []byte) {
	params.args.SetBytesKV(name, value)
}

func (params *Params) Remove(name []byte) {
	params.args.DelBytes(name)
}

func (params *Params) Len() int {
	return params.args.Len()
}

func (params *Params) Encode() (p []byte) {
	params.args.Sort(bytes.Compare)
	return params.args.QueryString()
}
