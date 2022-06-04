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
	"bytes"
	sc "context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/documents"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
)

type ServiceComponent interface {
	Name() (name string)
	Build(env Environments) (err error)
}

// Service
// 管理 Fn 的服务
type Service interface {
	Name() (name string)
	Internal() (internal bool)
	Build(env Environments) (err error)
	Components() (components map[string]ServiceComponent)
	Document() (doc *documents.Service)
	Handle(context Context, fn string, argument Argument, result ResultWriter)
	Shutdown(context sc.Context) (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

type ServiceOptions struct {
	components map[string]ServiceComponent
}

type ServiceOption func(*ServiceOptions) error

func ServiceComponents(components ...ServiceComponent) (opt ServiceOption) {
	return func(options *ServiceOptions) (err error) {
		if components == nil || len(components) == 0 {
			err = fmt.Errorf("fns: append service components failed for components is empty")
			return
		}
		if options.components == nil {
			options.components = make(map[string]ServiceComponent)
		}
		for _, component := range components {
			if component == nil {
				err = fmt.Errorf("fns: append service components failed for one of components is nil")
				return
			}
			name := component.Name()
			if name == "" {
				err = fmt.Errorf("fns: append service components failed for one of components's name is empty")
				return
			}
			_, has := options.components[name]
			if has {
				err = fmt.Errorf("fns: append service components failed for %s has appended", component.Name())
				return
			}
			options.components[name] = component
		}
		return
	}
}

func NewAbstractService(options ...ServiceOption) AbstractService {
	opt := &ServiceOptions{
		components: make(map[string]ServiceComponent),
	}
	if options != nil {
		for _, option := range options {
			if option != nil {
				optErr := option(opt)
				if optErr != nil {
					panic(optErr)
				}
			}
		}
	}
	return AbstractService{
		components: opt.components,
	}
}

type AbstractService struct {
	components map[string]ServiceComponent
}

func (s *AbstractService) Components() (components map[string]ServiceComponent) {
	components = s.components
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type Runtime interface {
	AppId() (id string)
	AppAddress() (address string)
	Log() (log logs.Logger)
	Endpoints() (endpoints Endpoints)
	Validator() (v Validator)
}

func newServiceRuntime(env Environments, endpoints Endpoints, validator Validator) (rt *serviceRuntime) {
	rt = &serviceRuntime{
		appId:      env.AppId(),
		appAddress: env.AppAddress(),
		log:        env.Log(),
		endpoints:  endpoints,
		validator:  validator,
	}
	return
}

type serviceRuntime struct {
	appId      string
	appAddress string
	log        logs.Logger
	endpoints  Endpoints
	validator  Validator
}

func (rt *serviceRuntime) AppId() (id string) {
	id = rt.appId
	return
}

func (rt *serviceRuntime) AppAddress() (address string) {
	address = rt.appAddress
	return
}

func (rt *serviceRuntime) Log() (log logs.Logger) {
	log = rt.log
	return
}

func (rt *serviceRuntime) Endpoints() (endpoints Endpoints) {
	endpoints = rt.endpoints
	return
}

func (rt *serviceRuntime) Validator() (v Validator) {
	v = rt.validator
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type Argument interface {
	json.Marshaler
	json.Unmarshaler
	IsNil() (ok bool)
	As(v interface{}) (err errors.CodeError)
}

func EmptyArgument() (arg Argument) {
	arg = NewArgument(nil)
	return
}

func NewArgument(v interface{}) (arg Argument) {
	arg = &argument{
		value: v,
	}
	return
}

type argument struct {
	value interface{}
}

func (arg argument) MarshalJSON() (data []byte, err error) {
	if arg.IsNil() {
		data = nullJson
		return
	}
	switch arg.value.(type) {
	case []byte:
		value := arg.value.([]byte)
		if !json.Validate(value) {
			err = errors.Warning("fns: type of argument is not json bytes").WithMeta("scope", "argument")
			return
		}
		data = value
		break
	case json.RawMessage:
		data = arg.value.(json.RawMessage)
		break
	default:
		data, err = json.Marshal(arg.value)
		if err != nil {
			err = errors.Warning("fns: encode argument to json failed").WithMeta("scope", "argument").WithCause(err)
			return
		}
	}
	return
}

func (arg *argument) UnmarshalJSON(data []byte) (err error) {
	arg.value = json.RawMessage(data)
	return
}

func (arg *argument) IsNil() (ok bool) {
	if arg.value == nil {
		ok = true
		return
	}
	switch arg.value.(type) {
	case []byte:
		value := arg.value.([]byte)
		ok = bytes.Equal(value, nullJson) || len(value) == 0
		break
	case json.RawMessage:
		value := arg.value.(json.RawMessage)
		ok = bytes.Equal(value, nullJson) || len(value) == 0
		break
	}
	return
}

func (arg *argument) As(v interface{}) (err errors.CodeError) {
	if arg.IsNil() {
		return
	}
	switch arg.value.(type) {
	case []byte:
		value := arg.value.([]byte)
		if json.Validate(value) {
			decodeErr := json.Unmarshal(value, v)
			if decodeErr != nil {
				err = errors.Warning("fns: decode argument failed").WithMeta("scope", "argument").WithCause(decodeErr)
				return
			}
		} else {
			cpErr := commons.CopyInterface(v, arg.value)
			if cpErr != nil {
				err = errors.Warning("fns: decode argument failed").WithMeta("scope", "argument").WithCause(cpErr)
				return
			}
		}
		break
	case json.RawMessage:
		value := arg.value.(json.RawMessage)
		decodeErr := json.Unmarshal(value, v)
		if decodeErr != nil {
			err = errors.Warning("fns: decode argument failed").WithMeta("scope", "argument").WithCause(decodeErr)
			return
		}
		break
	default:
		cpErr := commons.CopyInterface(v, arg.value)
		if cpErr != nil {
			err = errors.Warning("fns: decode argument failed").WithMeta("scope", "argument").WithCause(cpErr)
			return
		}
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type ResultWriter interface {
	Succeed(v interface{})
	Failed(err errors.CodeError)
}

type Result interface {
	ResultWriter
	Get(ctx sc.Context, v interface{}) (has bool, err errors.CodeError)
}

func NewResult() Result {
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
		err = errors.Warning("fns: failed result").WithMeta("scope", "result")
	}
	r.ch <- err
	close(r.ch)
}

func (r *futureResult) Get(ctx sc.Context, v interface{}) (has bool, err errors.CodeError) {
	select {
	case <-ctx.Done():
		err = errors.Timeout("fns: get result timeout").WithMeta("scope", "result")
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
					err = errors.Warning("fns: get result failed").WithMeta("scope", "result").WithCause(decodeErr)
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
					err = errors.Warning("fns: get result failed").WithMeta("scope", "result").WithCause(encodeErr)
					return
				}
				vv := v.(*json.RawMessage)
				*vv = append(*vv, value...)
			case *[]byte:
				value, encodeErr := json.Marshal(data)
				if encodeErr != nil {
					err = errors.Warning("fns: get result failed").WithMeta("scope", "result").WithCause(encodeErr)
					return
				}
				vv := v.(*[]byte)
				*vv = append(*vv, value...)
			default:
				cpErr := commons.CopyInterface(v, data)
				if cpErr != nil {
					err = errors.Warning("fns: get result failed").WithMeta("scope", "result").WithCause(cpErr)
					return
				}
			}
			has = true
		}
	}
	return
}

// Empty
// @description Empty
type Empty struct{}

// +-------------------------------------------------------------------------------------------------------------------+
