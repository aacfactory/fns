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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
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

func newRequestHeader(v http.Header) RequestHeader {
	return &requestHeader{
		value: v,
	}
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

func newRequestUser() (u RequestUser) {
	u = &requestUser{
		id:            "",
		authenticated: false,
		attributes:    json.NewObject(),
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

type Request interface {
	Id() (id string)
	Authorization() (v string)
	Internal() (ok bool)
	RemoteIp() (v string)
	Header() (header RequestHeader)
	User() (user RequestUser)
	SocketId() (id string)
}

func emptyRequest() Request {
	return &request{
		id:     UID(),
		header: newRequestHeader(http.Header{}),
		user:   newRequestUser(),
	}
}

func newRequest(req *http.Request) Request {
	requestId := req.Header.Get(httpIdHeader)
	internal := requestId != ""
	if requestId == "" {
		requestId = UID()
	}
	remoteIp := req.RemoteAddr
	if remoteIp != "" {
		if strings.Index(remoteIp, ".") > 0 && strings.Index(remoteIp, ":") > 0 {
			// ip:port
			remoteIp = remoteIp[0:strings.Index(remoteIp, ":")]
		}
	}
	return &request{
		id:       requestId,
		internal: internal,
		remoteIp: remoteIp,
		header:   newRequestHeader(req.Header),
		user:     newRequestUser(),
	}
}

func newWebsocketRequest(authorization string) (r Request) {
	header := http.Header{}
	if authorization != "" {
		header.Set("Authorization", authorization)
	}
	r = &request{
		id:     UID(),
		header: newRequestHeader(header),
		user:   newRequestUser(),
	}
	return
}

type request struct {
	id       string
	internal bool
	remoteIp string
	socketId string
	header   RequestHeader
	user     RequestUser
}

func (r *request) Id() (id string) {
	id = r.id
	return
}

func (r *request) Header() (header RequestHeader) {
	header = r.header
	return
}

func (r *request) User() (user RequestUser) {
	user = r.user
	return
}

func (r *request) Internal() (ok bool) {
	ok = r.internal
	return
}

func (r *request) RemoteIp() (v string) {
	realIp := r.header.Get(httpXRealIp)
	if realIp != "" {
		return
	}
	forwarded := r.header.Get(httpXForwardedFor)
	if forwarded != "" {
		items := strings.Split(forwarded, ",")
		v = strings.TrimSpace(items[len(items)-1])
		return
	}
	v = r.remoteIp
	return
}

func (r *request) Authorization() (v string) {
	v = r.header.Get(httpAuthorizationHeader)
	return
}

func (r *request) SocketId() (id string) {
	id = r.socketId
	return
}

// +-------------------------------------------------------------------------------------------------------------------+
