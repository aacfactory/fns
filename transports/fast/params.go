/*
 * Copyright 2023 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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

func (params *Params) Add(name []byte, value []byte) {
	params.args.AddBytesKV(name, value)
}

func (params *Params) Values(name []byte) [][]byte {
	return params.args.PeekMultiBytes(name)
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
