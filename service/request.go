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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/json"
	"github.com/cespare/xxhash/v2"
	"github.com/valyala/bytebufferpool"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type RequestHeader interface {
	Contains(key string) (ok bool)
	Get(key string) (value string)
	Values(key string) (values []string)
	Raw() (v http.Header)
}

func NewRequestHeader(value http.Header) RequestHeader {
	return &requestHeader{value: value}
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

func (header *requestHeader) Raw() (v http.Header) {
	v = header.value
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type RequestUser interface {
	json.Marshaler
	json.Unmarshaler
	Authenticated() (ok bool)
	Id() (id string)
	IntId() (id int64)
	Attributes() (attributes *json.Object)
}

func NewRequestUser(id string, attributes *json.Object) (u RequestUser) {
	u = &requestUser{
		id:            id,
		authenticated: id != "",
		attributes:    attributes,
	}
	return
}

type requestUser struct {
	authenticated bool
	id            string
	attributes    *json.Object
}

func (u *requestUser) Authenticated() (ok bool) {
	ok = u.authenticated
	return
}

func (u *requestUser) Id() (id string) {
	id = u.id
	return
}

func (u *requestUser) IntId() (id int64) {
	n, parseErr := strconv.ParseInt(u.id, 10, 64)
	if parseErr != nil {
		panic(errors.Warning(fmt.Sprintf("fns: parse user id to int failed")).WithMeta("scope", "system").WithMeta("id", u.id).WithCause(parseErr))
	}
	id = n
	return
}

func (u *requestUser) Attributes() (attributes *json.Object) {
	attributes = u.attributes
	return
}

func (u *requestUser) MarshalJSON() (p []byte, err error) {
	o := json.NewObject()
	_ = o.Put("id", u.id)
	_ = o.Put("authenticated", u.authenticated)
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
	err = o.Get("authenticated", &u.authenticated)
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

type RequestLocal interface {
	json.Marshaler
	Scan(key string, value interface{}) (has bool, err errors.CodeError)
	Put(key string, value interface{})
	Remove(key string)
}

func NewRequestLocal() (v RequestLocal) {
	v = &requestLocal{
		values: make(map[string]interface{}),
	}
	return
}

type requestLocal struct {
	values map[string]interface{}
}

func (local *requestLocal) Scan(key string, value interface{}) (has bool, err errors.CodeError) {
	v, exist := local.values[key]
	if !exist {
		return
	}
	cpErr := commons.CopyInterface(value, v)
	if cpErr != nil {
		err = errors.Warning("fns: request local scan failed").WithCause(cpErr).WithMeta("key", key)
		return
	}
	return
}

func (local *requestLocal) Put(key string, value interface{}) {
	local.values[key] = value
}

func (local *requestLocal) Remove(key string) {
	delete(local.values, key)
}

func (local *requestLocal) MarshalJSON() (p []byte, err error) {
	p, err = json.Marshal(local.values)
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type Request interface {
	Id() (id string)
	Internal() (ok bool)
	Authorization() (v string)
	RemoteIp() (v string)
	User() (user RequestUser)
	SetUser(id string, attr *json.Object)
	Local() (local RequestLocal)
	Header() (header RequestHeader)
	Fn() (service string, fn string)
	Argument() (argument Argument)
	Hash() (code uint64)
}

func NewRequest(req *http.Request) (r Request, err errors.CodeError) {
	pathItems := strings.Split(req.URL.Path, "/")
	if len(pathItems) != 3 {
		err = errors.NotAcceptable("fns: invalid request url path")
		return
	}
	service := pathItems[1]
	fn := pathItems[2]
	body, readBodyErr := ioutil.ReadAll(req.Body)
	if readBodyErr != nil {
		err = errors.NotAcceptable("fns: invalid request body")
		return
	}
	remoteIp := req.RemoteAddr
	if remoteIp != "" {
		if strings.Index(remoteIp, ".") > 0 && strings.Index(remoteIp, ":") > 0 {
			// ip:port
			remoteIp = remoteIp[0:strings.Index(remoteIp, ":")]
		}
	}
	buf := bytebufferpool.Get()
	_, _ = buf.Write([]byte(service + fn))
	authorization := req.Header.Get("Authorization")
	if authorization != "" {
		_, _ = buf.Write([]byte(authorization))
	}
	_, _ = buf.Write(body)
	hashCode := xxhash.Sum64(buf.Bytes())
	bytebufferpool.Put(buf)
	r = &request{
		id:       uid.UID(),
		internal: false,
		remoteIp: remoteIp,
		user:     NewRequestUser("", json.NewObject()),
		local: &requestLocal{
			values: make(map[string]interface{}),
		},
		header:   NewRequestHeader(req.Header),
		service:  service,
		fn:       fn,
		argument: NewArgument(body),
		hashCode: hashCode,
	}
	return
}

func NewInternalRequest(service string, fn string, arg interface{}) (r Request, err errors.CodeError) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, _ = buf.Write([]byte(service + fn))
	if arg != nil {
		switch arg.(type) {
		case []byte:
			_, _ = buf.Write(arg.([]byte))
			break
		case string:
			_, _ = buf.Write([]byte(arg.(string)))
			break
		case json.RawMessage:
			_, _ = buf.Write(arg.(json.RawMessage))
			break
		case stdjson.RawMessage:
			_, _ = buf.Write(arg.(stdjson.RawMessage))
			break
		case json.Marshaler:
			encoder := arg.(json.Marshaler)
			p, encodeErr := encoder.MarshalJSON()
			if encodeErr != nil {
				err = errors.Warning("fns: new internal request failed").WithCause(encodeErr)
				return
			}
			_, _ = buf.Write(p)
		case stdjson.Marshaler:
			encoder := arg.(stdjson.Marshaler)
			p, encodeErr := encoder.MarshalJSON()
			if encodeErr != nil {
				err = errors.Warning("fns: new internal request failed").WithCause(encodeErr)
				return
			}
			_, _ = buf.Write(p)
		case Argument:
			encoder := arg.(Argument)
			p, encodeErr := encoder.MarshalJSON()
			if encodeErr != nil {
				err = errors.Warning("fns: new internal request failed").WithCause(encodeErr)
				return
			}
			_, _ = buf.Write(p)
		default:
			p, encodeErr := json.Marshal(arg)
			if encodeErr != nil {
				err = errors.Warning("fns: new internal request failed").WithCause(encodeErr)
				return
			}
			_, _ = buf.Write(p)
			break
		}
	}
	hashCode := xxhash.Sum64(buf.Bytes())
	r = &request{
		id:       uid.UID(),
		internal: true,
		remoteIp: "",
		user:     NewRequestUser("", json.NewObject()),
		local: &requestLocal{
			values: make(map[string]interface{}),
		},
		header:   NewRequestHeader(http.Header{}),
		service:  service,
		fn:       fn,
		argument: NewArgument(arg),
		hashCode: hashCode,
	}
	return
}

type request struct {
	id       string
	internal bool
	remoteIp string
	user     RequestUser
	local    RequestLocal
	header   RequestHeader
	service  string
	fn       string
	argument Argument
	hashCode uint64
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

func (r *request) SetUser(id string, attributes *json.Object) {
	r.user = NewRequestUser(id, attributes)
}

func (r *request) Local() (local RequestLocal) {
	local = r.local
	return
}

func (r *request) RemoteIp() (v string) {
	realIp := r.header.Get("X-Real-Ip")
	if realIp != "" {
		return
	}
	forwarded := r.header.Get("X-Forwarded-For")
	if forwarded != "" {
		items := strings.Split(forwarded, ",")
		v = strings.TrimSpace(items[len(items)-1])
		return
	}
	v = r.remoteIp
	return
}

func (r *request) Authorization() (v string) {
	v = r.header.Get("Authorization")
	return
}

func (r *request) Header() (header RequestHeader) {
	header = r.header
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

func (r *request) Hash() (code uint64) {
	code = r.hashCode
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	contextRequestKey = "_request_"
)

func GetRequest(ctx context.Context) (r Request, has bool) {
	vbv := ctx.Value(contextRequestKey)
	if vbv == nil {
		return
	}
	r, has = vbv.(Request)
	return
}

func SetRequest(ctx context.Context, r Request) context.Context {
	_, has := GetRequest(ctx)
	if has {
		return ctx
	}
	ctx = context.WithValue(ctx, contextRequestKey, r)
	return ctx
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
