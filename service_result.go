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

package fns

import (
	sc "context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"reflect"
)

func AsyncResult() Result {
	return &futureResult{
		ch: make(chan []byte, 1),
	}
}

type futureResult struct {
	ch chan []byte
}

func (r *futureResult) Succeed(v interface{}) {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		rt := reflect.TypeOf(v)
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
		}
		if rt.Kind() == reflect.Struct {
			r.ch <- []byte("+{}")
		} else if rt.Kind() == reflect.Slice || rt.Kind() == reflect.Array {
			r.ch <- []byte("+[]")
		} else {
			r.ch <- []byte("+")
		}
		return
	}

	var p []byte
	switch v.(type) {
	case []byte:
		p = v.([]byte)
	default:
		p0, encodeErr := json.Marshal(v)
		if encodeErr != nil {
			r.Failed(errors.ServiceError(encodeErr.Error()))
			return
		}
		p = p0
	}

	data := make([]byte, len(p)+1)
	data[0] = '+'
	copy(data[1:], p)
	r.ch <- data
	close(r.ch)
}

func (r *futureResult) Failed(err errors.CodeError) {
	p, encodeErr := json.Marshal(err)
	if encodeErr != nil {
		r.Failed(errors.ServiceError(encodeErr.Error()))
		return
	}
	data := make([]byte, len(p)+1)
	data[0] = '-'
	copy(data[1:], p)
	r.ch <- data
	close(r.ch)
}

func (r *futureResult) Get(ctx sc.Context, v interface{}) (err errors.CodeError) {
	select {
	case <-ctx.Done():
		err = errors.Timeout("timeout")
		return
	case data := <-r.ch:
		if data[0] == '-' {
			err = errors.ServiceError("")
			decodeErr := json.Unmarshal(data[1:], err)
			if decodeErr != nil {
				err = errors.Map(decodeErr)
				return
			}
			return
		}
		if len(data) == 1 {
			// empty
			return
		}
		decodeErr := json.Unmarshal(data[1:], v)
		if decodeErr != nil {
			err = errors.Map(decodeErr)
			return
		}
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func SyncResult() Result {
	return &syncResult{
		data: nil,
		err:  nil,
	}
}

type syncResult struct {
	data []byte
	err  errors.CodeError
}

func (r *syncResult) Succeed(v interface{}) {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		rt := reflect.TypeOf(v)
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
		}
		if rt.Kind() == reflect.Struct {
			r.data = []byte("{}")
		} else if rt.Kind() == reflect.Slice || rt.Kind() == reflect.Array {
			r.data = []byte("[]")
		} else {
			r.data = nullJson
		}
		return
	}
	data, ok := v.([]byte)
	if ok {
		r.data = data
		return
	}
	p, encodeErr := json.Marshal(v)
	if encodeErr != nil {
		r.Failed(errors.ServiceError(encodeErr.Error()))
		return
	}
	r.data = p
}

func (r *syncResult) Failed(err errors.CodeError) {
	r.err = err
}

func (r *syncResult) Get(_ sc.Context, v interface{}) (err errors.CodeError) {
	if r.err != nil {
		err = r.err
		return
	}
	if r.data == nil {
		return
	}
	decodeErr := json.Unmarshal(r.data, v)
	if decodeErr != nil {
		err = errors.Map(decodeErr)
		return
	}
	return
}
