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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
)

func asyncResult() Result {
	return &futureResult{
		ch: make(chan []byte, 1),
	}
}

type futureResult struct {
	ch chan []byte
}

func (r *futureResult) Succeed(v interface{}) {
	if v == nil {
		r.ch <- []byte("+")
		return
	}
	p, encodeErr := json.Marshal(v)
	if encodeErr != nil {
		r.Failed(errors.ServiceError(encodeErr.Error()))
		return
	}
	data := make([]byte, len(p)+1)
	data = append(data, '+')
	data = append(data, p...)
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
	data = append(data, '-')
	data = append(data, p...)
	r.ch <- data
	close(r.ch)
}

func (r *futureResult) Get(v interface{}) (err errors.CodeError) {
	data := <-r.ch
	if data[0] == '-' {
		err = errors.ServiceError("")
		decodeErr := json.Unmarshal(data[1:], &err)
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
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func syncResult() Result {
	return &result{
		data: nil,
		err:  nil,
	}
}

type result struct {
	data []byte
	err  errors.CodeError
}

func (r *result) Succeed(v interface{}) {
	if v == nil {
		return
	}
	p, encodeErr := json.Marshal(v)
	if encodeErr != nil {
		r.Failed(errors.ServiceError(encodeErr.Error()))
		return
	}
	r.data = p
}

func (r *result) Failed(err errors.CodeError) {
	r.err = err
}

func (r *result) Get(v interface{}) (err errors.CodeError) {
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
