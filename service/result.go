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
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service/internal/commons/objects"
	"github.com/aacfactory/json"
)

type Promise interface {
	Succeed(v interface{})
	Failed(err errors.CodeError)
	Close()
}

type FutureResult interface {
	json.Marshaler
	Exist() (ok bool)
	Scan(v interface{}) (err errors.CodeError)
}

type Future interface {
	Get(ctx context.Context) (result FutureResult, err errors.CodeError)
}

func NewFuture() (p Promise, f Future) {
	fp := newFuture()
	p = fp
	f = fp
	return
}

func newFuture() (fp *futurePipe) {
	fp = &futurePipe{
		ch: make(chan interface{}, 1),
	}
	return
}

type futurePipe struct {
	ch chan interface{}
}

func (fp *futurePipe) Close() {
	close(fp.ch)
}

func (fp *futurePipe) Succeed(v interface{}) {
	fp.ch <- v
	close(fp.ch)
}

func (fp *futurePipe) Failed(err errors.CodeError) {
	if err == nil {
		err = errors.Warning("fns: empty failed result").WithMeta("fns", "future")
	}
	fp.ch <- err
	close(fp.ch)
}

func (fp *futurePipe) Get(ctx context.Context) (result FutureResult, err errors.CodeError) {
	select {
	case <-ctx.Done():
		err = errors.Timeout("fns: get result value timeout").WithMeta("fns", "future")
		return
	case data, ok := <-fp.ch:
		if !ok {
			err = errors.Warning("fns: future was closed").WithMeta("fns", "future")
			return
		}
		if data == nil {
			result = &futureResult{
				data: nil,
			}
			return
		}
		switch data.(type) {
		case errors.CodeError:
			err = data.(errors.CodeError)
			return
		case error:
			err = errors.Map(data.(error))
			return
		default:
			result = &futureResult{
				data: data,
			}
		}
		return
	}
}

type futureResult struct {
	data interface{}
}

func (fr *futureResult) Exist() (ok bool) {
	if fr.data == nil {
		return
	}
	switch fr.data.(type) {
	case []byte:
		p := fr.data.([]byte)
		ok = len(p) > 0 && nilJson != bytex.ToString(p)
	case json.RawMessage:
		p := fr.data.(json.RawMessage)
		ok = len(p) > 0 && nilJson != bytex.ToString(p)
	case stdjson.RawMessage:
		p := fr.data.(stdjson.RawMessage)
		ok = len(p) > 0 && nilJson != bytex.ToString(p)
	default:
		ok = true
		break
	}
	return
}

func (fr *futureResult) Scan(v interface{}) (err errors.CodeError) {
	if fr.data == nil {
		return
	}
	switch fr.data.(type) {
	case *Empty, Empty:
		return
	case []byte, json.RawMessage, stdjson.RawMessage:
		var value []byte
		switch fr.data.(type) {
		case []byte:
			value = fr.data.([]byte)
			break
		case json.RawMessage:
			value = fr.data.(json.RawMessage)
			break
		case stdjson.RawMessage:
			value = fr.data.(stdjson.RawMessage)
			break
		}
		if len(value) == 0 {
			return
		}
		switch v.(type) {
		case *json.RawMessage:
			vv := v.(*json.RawMessage)
			*vv = append(*vv, value...)
		case *stdjson.RawMessage:
			vv := v.(*stdjson.RawMessage)
			*vv = append(*vv, value...)
		case *[]byte:
			vv := v.(*[]byte)
			*vv = append(*vv, value...)
		default:
			if nilJson == bytex.ToString(value) || emptyJson == bytex.ToString(value) || emptyArrayJson == bytex.ToString(value) {
				return
			}
			decodeErr := json.Unmarshal(value, v)
			if decodeErr != nil {
				err = errors.Warning("fns: future result scan failed").WithMeta("fns", "future").WithCause(decodeErr)
				return
			}
		}
		return
	default:
		switch v.(type) {
		case *json.RawMessage:
			value, encodeErr := json.Marshal(fr.data)
			if encodeErr != nil {
				err = errors.Warning("fns: future result scan failed").WithMeta("fns", "future").WithCause(encodeErr)
				return
			}
			vv := v.(*json.RawMessage)
			*vv = append(*vv, value...)
			break
		case *stdjson.RawMessage:
			value, encodeErr := json.Marshal(fr.data)
			if encodeErr != nil {
				err = errors.Warning("fns: future result scan failed").WithMeta("fns", "future").WithCause(encodeErr)
				return
			}
			vv := v.(*stdjson.RawMessage)
			*vv = append(*vv, value...)
			break
		case *[]byte:
			value, encodeErr := json.Marshal(fr.data)
			if encodeErr != nil {
				err = errors.Warning("fns: future result scan failed").WithMeta("fns", "future").WithCause(encodeErr)
				return
			}
			vv := v.(*[]byte)
			*vv = append(*vv, value...)
			break
		default:
			cpErr := objects.CopyInterface(v, fr.data)
			if cpErr != nil {
				err = errors.Warning("fns: future result scan failed").WithMeta("fns", "future").WithCause(cpErr)
				return
			}
		}
	}
	return
}

func (fr *futureResult) MarshalJSON() (p []byte, err error) {
	if fr.data == nil {
		p = bytex.FromString(nilJson)
		return
	}
	switch fr.data.(type) {
	case []byte:
		x := fr.data.([]byte)
		if json.Validate(x) {
			p = x
		} else {
			p, err = json.Marshal(fr.data)
		}
	case json.RawMessage:
		p = fr.data.(json.RawMessage)
	case stdjson.RawMessage:
		p = fr.data.(stdjson.RawMessage)
	default:
		p, err = json.Marshal(fr.data)
	}
	return
}
