/*
 * Copyright 2021 Wang Min Xiang
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
 */

package json

import (
	"bytes"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/valyala/bytebufferpool"
	"io"
)

func NewArrayFromBytes(b []byte) *Array {
	if b[0] != '[' || b[len(b)-1] != ']' {
		panic(fmt.Errorf("new json array from bytes failed, %s is not json array bytes", string(b)))
	}
	return &Array{
		raw: b,
	}
}

func NewArray() *Array {
	return &Array{
		raw: []byte{'[', ']'},
	}
}

type Array struct {
	raw []byte
}

func (array *Array) Empty() (ok bool) {
	if array.raw == nil || len(array.raw) == 0 {
		ok = true
		return
	}
	ok = !gjson.ParseBytes(array.raw).Exists()
	return
}

func (array *Array) Raw() (raw []byte) {
	raw = array.raw
	return
}

func (array *Array) Add(values ...interface{}) (err error) {
	if values == nil || len(values) == 0 {
		err = fmt.Errorf("json array add failed, values is empty")
		return
	}
	rb := bytes.NewReader(array.raw)
	nb := bytebufferpool.Get()
	_, _ = io.Copy(nb, rb)
	affected := nb.Bytes()
	bytebufferpool.Put(nb)
	var addErr error
	for i, value := range values {
		if value == nil {
			continue
		}
		affected, addErr = sjson.SetBytes(affected, "-1", value)
		if addErr != nil {
			err = fmt.Errorf("json array add %d failed", i)
			return
		}
	}
	array.raw = affected
	return
}

func (array *Array) Remove(i int) (err error) {
	if i < 0 {
		err = fmt.Errorf("json array remove failed, index is less than 0")
		return
	}
	affected, remErr := sjson.DeleteBytes(array.raw, fmt.Sprintf("%d", i))
	if remErr != nil {
		err = fmt.Errorf("json array remove %d failed", i)
		return
	}
	array.raw = affected
	return
}

func (array *Array) Len() (size int) {
	size = len(gjson.ParseBytes(array.raw).Array())
	return
}

func (array *Array) Get(i int, v interface{}) (err error) {
	if i < 0 || i >= array.Len() {
		err = fmt.Errorf("json array get failed, index is less than 0 or greater than len")
		return
	}
	if v == nil {
		err = fmt.Errorf("json array get %d failed, value is nil", i)
		return
	}
	raw := gjson.ParseBytes(array.raw).Array()[i].Raw
	decodeErr := Unmarshal([]byte(raw), v)
	if decodeErr != nil {
		err = fmt.Errorf("json array get %d failed, decode failed", i)
		return
	}
	return
}

func (array *Array) MapTo(v interface{}) (err error) {
	err = Unmarshal(array.raw, v)
	return
}

func (array *Array) MarshalJSON() (p []byte, err error) {
	p = array.raw
	return
}

func (array *Array) UnmarshalJSON(p []byte) (err error) {
	if p == nil || len(p) == 0 {
		return
	}
	if p[0] != '[' || p[len(p)-1] != ']' {
		err = fmt.Errorf("unmarshal json array from bytes failed, %s is not json array bytes", string(p))
		return
	}
	array.raw = p
	return
}
