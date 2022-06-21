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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/json"
)

type ResultWriter interface {
	Succeed(v interface{})
	Failed(err errors.CodeError)
}

type Result interface {
	Get(ctx context.Context, v interface{}) (has bool, err errors.CodeError)
	Value(ctx context.Context) (value interface{}, has bool, err errors.CodeError)
}

type FutureResult interface {
	ResultWriter
	Result
}

func NewResult() FutureResult {
	return &futureResult{
		ch: make(chan interface{}, 1),
	}
}

type futureResult struct {
	ch chan interface{}
}

func (r *futureResult) Succeed(v interface{}) {
	if v == nil {
		close(r.ch)
		return
	}
	raw, ok := v.(json.RawMessage)
	if ok && len(raw) == 0 {
		close(r.ch)
		return
	}
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
			return
		}
		value = data
		has = true
		return
	}
}

func (r *futureResult) Get(ctx context.Context, v interface{}) (has bool, err errors.CodeError) {
	select {
	case <-ctx.Done():
		err = errors.Timeout("fns: get result timeout").WithMeta("fns", "result")
		return
	case data, ok := <-r.ch:
		if !ok {
			return
		}
		switch data.(type) {
		case errors.CodeError:
			err = data.(errors.CodeError)
			return
		case error:
			err = errors.Warning(data.(error).Error())
			return
		case []byte, json.RawMessage:
			value := data.([]byte)
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
			case *[]byte:
				value, encodeErr := json.Marshal(data)
				if encodeErr != nil {
					err = errors.Warning("fns: get result failed").WithMeta("fns", "result").WithCause(encodeErr)
					return
				}
				vv := v.(*[]byte)
				*vv = append(*vv, value...)
			default:
				cpErr := commons.CopyInterface(v, data)
				if cpErr != nil {
					err = errors.Warning("fns: get result failed").WithMeta("fns", "result").WithCause(cpErr)
					return
				}
			}
			has = true
		}
	}
	return
}
