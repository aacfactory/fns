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

package service

import (
	"context"
	stdjson "encoding/json"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service/internal/commons/objects"
	"github.com/aacfactory/json"
)

type ResultWriter interface {
	Succeed(v interface{})
	Failed(err errors.CodeError)
	Close()
}

type Result interface {
	Get(ctx context.Context, v interface{}) (has bool, err errors.CodeError)
	Value(ctx context.Context) (value interface{}, has bool, err errors.CodeError)
}

type FutureResult interface {
	ResultWriter
	Result
}

func NewResult() (f FutureResult) {
	return newFutureResult()
}

func newFutureResult() (fr *futureResult) {
	fr = &futureResult{
		ch: make(chan interface{}, 1),
	}
	return
}

type futureResult struct {
	ch chan interface{}
}

func (r *futureResult) Close() {
	close(r.ch)
}

func (r *futureResult) Succeed(v interface{}) {
	r.ch <- v
	close(r.ch)
}

func (r *futureResult) Failed(err errors.CodeError) {
	if err == nil {
		err = errors.Warning("fns: empty failed result").WithMeta("fns", "result")
	}
	r.ch <- err
	close(r.ch)
}

func (r *futureResult) Value(ctx context.Context) (value interface{}, has bool, err errors.CodeError) {
	select {
	case <-ctx.Done():
		err = errors.Timeout("fns: get result value timeout").WithMeta("fns", "result")
		return
	case data, ok := <-r.ch:
		if !ok {
			err = errors.Timeout("fns: future was closed").WithMeta("fns", "result")
			return
		}
		if data == nil {
			return
		}
		switch data.(type) {
		case errors.CodeError:
			err = data.(errors.CodeError)
			break
		case error:
			err = errors.Map(data.(error))
			break
		default:
			value = data
			has = true
		}
		return
	}
}

func (r *futureResult) Get(ctx context.Context, v interface{}) (has bool, err errors.CodeError) {
	var data interface{}
	data, has, err = r.Value(ctx)
	if err != nil {
		return
	}
	if !has {
		return
	}
	switch data.(type) {
	case *Empty:
		return
	case []byte, json.RawMessage, stdjson.RawMessage:
		var value []byte
		switch data.(type) {
		case []byte:
			value = data.([]byte)
			break
		case json.RawMessage:
			value = data.(json.RawMessage)
			break
		case stdjson.RawMessage:
			value = data.(stdjson.RawMessage)
			break
		}
		if len(value) == 0 {
			return
		}
		switch v.(type) {
		case *json.RawMessage:
			vv := v.(*json.RawMessage)
			*vv = append(*vv, value...)
		case *[]byte:
			vv := v.(*[]byte)
			*vv = append(*vv, value...)
		default:
			decodeErr := json.Unmarshal(value, v)
			if decodeErr != nil {
				err = errors.Warning("fns: get result failed").WithMeta("fns", "result").WithCause(decodeErr)
				return
			}
		}
		has = true
		return
	default:
		switch v.(type) {
		case *json.RawMessage:
			value, encodeErr := json.Marshal(data)
			if encodeErr != nil {
				err = errors.Warning("fns: get result failed").WithMeta("fns", "result").WithCause(encodeErr)
				return
			}
			vv := v.(*json.RawMessage)
			*vv = append(*vv, value...)
			break
		case *stdjson.RawMessage:
			value, encodeErr := json.Marshal(data)
			if encodeErr != nil {
				err = errors.Warning("fns: get result failed").WithMeta("fns", "result").WithCause(encodeErr)
				return
			}
			vv := v.(*stdjson.RawMessage)
			*vv = append(*vv, value...)
			break
		case *[]byte:
			value, encodeErr := json.Marshal(data)
			if encodeErr != nil {
				err = errors.Warning("fns: get result failed").WithMeta("fns", "result").WithCause(encodeErr)
				return
			}
			vv := v.(*[]byte)
			*vv = append(*vv, value...)
			break
		default:
			cpErr := objects.CopyInterface(v, data)
			if cpErr != nil {
				err = errors.Warning("fns: get result failed").WithMeta("fns", "result").WithCause(cpErr)
				return
			}
		}
		has = true
	}
	return
}
