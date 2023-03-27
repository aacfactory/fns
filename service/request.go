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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/json"
	"net/http"
	"strconv"
	"strings"
)

type RequestHeader interface {
	Contains(key string) (ok bool)
	Get(key string) (value string)
	Values(key string) (values []string)
	Set(key string, value string)
	Add(key string, value string)
	ForEach(fn func(key string, values []string) (next bool))
	DeviceId() (id string)
	DeviceIp() (ip string)
	Authorization() (authorization string, has bool)
	VersionRange() (left versions.Version, right versions.Version, err error)
	MapToHttpHeader() (v http.Header)
}

func newRequestHeader() RequestHeader {
	return &requestHeader{value: make(map[string][]string)}
}

type requestHeader struct {
	value http.Header
}

func (header *requestHeader) Contains(key string) (ok bool) {
	_, ok = header.value[key]
	return
}

func (header *requestHeader) Get(key string) (value string) {
	value = header.value.Get(key)
	return
}

func (header *requestHeader) Values(key string) (values []string) {
	values = header.value.Values(key)
	return
}

func (header *requestHeader) Set(key string, value string) {
	header.value.Set(key, value)
}

func (header *requestHeader) Add(key string, value string) {
	header.value.Add(key, value)
}

func (header *requestHeader) ForEach(fn func(key string, values []string) (next bool)) {
	if fn == nil {
		return
	}
	for key, values := range header.value {
		next := fn(key, values)
		if !next {
			break
		}
	}
}

func (header *requestHeader) Authorization() (authorization string, has bool) {
	authorization = header.Get("Authorization")
	has = authorization != ""
	return
}

func (header *requestHeader) DeviceId() (id string) {
	id = header.Get(httpDeviceIdHeader)
	return
}

func (header *requestHeader) DeviceIp() (id string) {
	id = header.Get(httpDeviceIpHeader)
	return
}

func (header *requestHeader) VersionRange() (left versions.Version, right versions.Version, err error) {
	version := header.Get(httpRequestVersionsHeader)
	if version == "" {
		right = versions.Max()
		return
	}
	versionRange := strings.Split(version, ",")
	leftVersionValue := strings.TrimSpace(versionRange[0])
	if leftVersionValue != "" {
		left, err = versions.Parse(leftVersionValue)
		if err != nil {
			err = errors.Warning("fns: read request version failed").WithCause(err)
			return
		}
	}
	if len(versionRange) > 1 {
		rightVersionValue := strings.TrimSpace(versionRange[1])
		if rightVersionValue != "" {
			right, err = versions.Parse(rightVersionValue)
			if err != nil {
				err = errors.Warning("fns: read request version failed").WithCause(err)
				return
			}
		}
	} else {
		right = versions.Max()
	}
	return
}

func (header *requestHeader) MapToHttpHeader() (v http.Header) {
	v = header.value
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type RequestUserId string

func (id RequestUserId) Int() (n int64) {
	s := string(id)
	v, parseErr := strconv.ParseInt(s, 10, 64)
	if parseErr != nil {
		panic(errors.Warning(fmt.Sprintf("fns: parse user id to int failed")).WithMeta("scope", "system").WithMeta("id", s).WithCause(parseErr))
	}
	n = v
	return
}

func (id RequestUserId) String() string {
	return string(id)
}

func (id RequestUserId) Exist() (ok bool) {
	ok = id != "" && id != "0"
	return
}

type RequestUser interface {
	json.Marshaler
	json.Unmarshaler
	Authenticated() (ok bool)
	Id() (id RequestUserId)
	SetId(id RequestUserId)
	Attributes() (attributes *json.Object)
	SetAttributes(attributes *json.Object)
}

func NewRequestUser(id RequestUserId, attributes *json.Object) (u RequestUser) {
	if attributes == nil {
		attributes = json.NewObject()
	}
	u = &requestUser{
		id:         id,
		attributes: attributes,
	}
	return
}

type requestUser struct {
	id         RequestUserId
	attributes *json.Object
}

func (u *requestUser) Authenticated() (ok bool) {
	ok = u.id != ""
	return
}

func (u *requestUser) Id() (id RequestUserId) {
	id = u.id
	return
}

func (u *requestUser) SetId(id RequestUserId) {
	u.id = id
}

func (u *requestUser) Attributes() (attributes *json.Object) {
	attributes = u.attributes
	return
}

func (u *requestUser) SetAttributes(attributes *json.Object) {
	if attributes == nil {
		attributes = json.NewObject()
	}
	u.attributes = attributes
}

func (u *requestUser) MarshalJSON() (p []byte, err error) {
	o := json.NewObject()
	_ = o.Put("id", u.id)
	_ = o.Put("authenticated", u.Authenticated())
	if u.attributes == nil {
		u.attributes = json.NewObject()
	}
	_ = o.Put("attributes", u.attributes)
	p, err = o.MarshalJSON()
	return
}

func (u *requestUser) UnmarshalJSON(p []byte) (err error) {
	o := json.NewObjectFromBytes(p)
	err = o.Get("id", &u.id)
	if err != nil {
		return
	}
	if u.attributes == nil {
		u.attributes = json.NewObject()
	}
	err = o.Get("attributes", &u.attributes)
	if err != nil {
		return
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type RequestTrunk interface {
	json.Marshaler
	json.Unmarshaler
	Get(key string) (value []byte, has bool)
	Put(key string, value []byte)
	ForEach(fn func(key string, value []byte) (next bool))
	ReadFrom(o RequestTrunk)
	Remove(key string)
}

func newRequestTrunk() RequestTrunk {
	return &requestTrunk{
		values: make(map[string][]byte),
	}
}

type requestTrunk struct {
	values map[string][]byte
}

func (trunk *requestTrunk) MarshalJSON() (p []byte, err error) {
	p, err = json.Marshal(trunk.values)
	return
}

func (trunk *requestTrunk) UnmarshalJSON(p []byte) (err error) {
	values := make(map[string][]byte)
	err = json.Unmarshal(p, &values)
	if err != nil {
		return
	}
	trunk.values = values
	return
}

func (trunk *requestTrunk) ReadFrom(o RequestTrunk) {
	trunk.values = make(map[string][]byte)
	o.ForEach(func(key string, value []byte) (next bool) {
		trunk.values[key] = value
		next = true
		return
	})
	return
}

func (trunk *requestTrunk) Get(key string) (value []byte, has bool) {
	value, has = trunk.values[key]
	return
}

func (trunk *requestTrunk) Put(key string, value []byte) {
	trunk.values[key] = value
}

func (trunk *requestTrunk) ForEach(fn func(key string, value []byte) (next bool)) {
	if fn == nil {
		return
	}
	for ket, value := range trunk.values {
		next := fn(ket, value)
		if !next {
			break
		}
	}
}

func (trunk *requestTrunk) Remove(key string) {
	delete(trunk.values, key)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Request interface {
	Id() (id string)
	Header() (header RequestHeader)
	ResponseHeader() (header http.Header)
	Fn() (service string, fn string)
	Argument() (argument Argument)
	Internal() (ok bool)
	User() (user RequestUser)
	Trunk() (trunk RequestTrunk)
}

type RequestOption func(*RequestOptions)

func WithRequestId(id string) RequestOption {
	return func(options *RequestOptions) {
		options.id = id
	}
}

func WithHttpRequestHeader(header http.Header) RequestOption {
	return func(options *RequestOptions) {
		for key, values := range header {
			for _, value := range values {
				options.header.Add(key, value)
			}
		}
	}
}

func WithHttpResponseHeader(header http.Header) RequestOption {
	return func(options *RequestOptions) {
		options.responseHeader = header
	}
}

func WithDeviceId(id string) RequestOption {
	return func(options *RequestOptions) {
		options.deviceId = id
	}
}

func WithDeviceIp(ip string) RequestOption {
	return func(options *RequestOptions) {
		options.deviceIp = ip
	}
}

func WithInternalRequest() RequestOption {
	return func(options *RequestOptions) {
		options.internal = true
	}
}

func WithRequestUser(id RequestUserId, attributes *json.Object) RequestOption {
	return func(options *RequestOptions) {
		options.user = NewRequestUser(id, attributes)
	}
}

func WithRequestTrunk(trunk RequestTrunk) RequestOption {
	return func(options *RequestOptions) {
		options.trunk = trunk
	}
}

type RequestOptions struct {
	id             string
	header         RequestHeader
	responseHeader http.Header
	deviceId       string
	deviceIp       string
	internal       bool
	user           RequestUser
	trunk          RequestTrunk
}

func NewRequest(ctx context.Context, service string, fn string, argument Argument, options ...RequestOption) (v Request) {
	opt := &RequestOptions{
		id:             "",
		header:         newRequestHeader(),
		responseHeader: http.Header{},
		deviceId:       "",
		deviceIp:       "",
		user:           nil,
		internal:       false,
		trunk:          newRequestTrunk(),
	}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			option(opt)
		}
	}
	if opt.deviceIp != "" {
		opt.header.Add(httpDeviceIpHeader, opt.deviceIp)
	}

	prev, hasPrev := GetRequest(ctx)
	if hasPrev {
		id := opt.id
		internal := false
		if id == "" {
			id = prev.Id()
			internal = true
		} else {
			internal = opt.internal
		}
		header := prev.Header()
		if len(opt.header.MapToHttpHeader()) > 0 {
			opt.header.ForEach(func(key string, values []string) (next bool) {
				for _, value := range values {
					header.Add(key, value)
				}
				next = true
				return
			})
		}
		user := opt.user
		if user == nil {
			user = prev.User()
		}
		v = &request{
			id:             id,
			internal:       internal,
			user:           user,
			trunk:          prev.Trunk(),
			header:         header,
			responseHeader: prev.ResponseHeader(),
			service:        service,
			fn:             fn,
			argument:       argument,
		}
	} else {
		id := opt.id
		if id == "" {
			id = uid.UID()
		}
		header := newRequestHeader()
		if len(opt.header.MapToHttpHeader()) > 0 {
			opt.header.ForEach(func(key string, values []string) (next bool) {
				for _, value := range values {
					header.Add(key, value)
				}
				next = true
				return
			})
		}
		if opt.deviceId != "" {
			header.Add(httpDeviceIdHeader, opt.deviceId)
		}
		user := opt.user
		if user == nil {
			user = NewRequestUser("", nil)
		}
		v = &request{
			id:             id,
			internal:       opt.internal,
			user:           user,
			trunk:          opt.trunk,
			header:         header,
			responseHeader: opt.responseHeader,
			service:        service,
			fn:             fn,
			argument:       argument,
		}
	}
	return
}

type request struct {
	id             string
	internal       bool
	user           RequestUser
	trunk          RequestTrunk
	header         RequestHeader
	responseHeader http.Header
	service        string
	fn             string
	argument       Argument
}

func (r *request) Id() (id string) {
	id = r.id
	return
}

func (r *request) Internal() (ok bool) {
	ok = r.internal
	return
}

func (r *request) User() (user RequestUser) {
	user = r.user
	return
}

func (r *request) Trunk() (trunk RequestTrunk) {
	trunk = r.trunk
	return
}

func (r *request) Header() (header RequestHeader) {
	header = r.header
	return
}

func (r *request) ResponseHeader() (header http.Header) {
	header = r.responseHeader
	return
}

func (r *request) Fn() (service string, fn string) {
	service, fn = r.service, r.fn
	return
}

func (r *request) Argument() (argument Argument) {
	argument = r.argument
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	contextRequestKey = "@fns_request"
)

func GetRequest(ctx context.Context) (r Request, has bool) {
	vbv := ctx.Value(contextRequestKey)
	if vbv == nil {
		return
	}
	r, has = vbv.(Request)
	return
}

func withRequest(ctx context.Context, r Request) context.Context {
	return context.WithValue(ctx, contextRequestKey, r)
}

func GetRequestUser(ctx context.Context) (user RequestUser, authenticated bool) {
	req, hasReq := GetRequest(ctx)
	if !hasReq {
		return
	}
	user = req.User()
	authenticated = user.Authenticated()
	return
}

type internalRequestImpl struct {
	User  *requestUser    `json:"user"`
	Trunk *requestTrunk   `json:"trunk"`
	Body  json.RawMessage `json:"body"`
}

type internalRequest struct {
	User  RequestUser     `json:"user"`
	Trunk RequestTrunk    `json:"trunk"`
	Body  json.RawMessage `json:"body"`
}

type internalResponseImpl struct {
	User   *requestUser    `json:"user"`
	Trunk  *requestTrunk   `json:"trunk"`
	Span   *Span           `json:"Span"`
	Header http.Header     `json:"header"`
	Body   json.RawMessage `json:"body"`
}

type internalResponse struct {
	User   RequestUser     `json:"user"`
	Trunk  RequestTrunk    `json:"trunk"`
	Span   *Span           `json:"Span"`
	Header http.Header     `json:"header"`
	Body   json.RawMessage `json:"body"`
}
