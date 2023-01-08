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

package cluster

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/cespare/xxhash/v2"
	"github.com/valyala/bytebufferpool"
	"io/ioutil"
	"net/http"
	"strings"
)

type internalRequest struct {
	Local    json.RawMessage `json:"local"`
	User     json.RawMessage `json:"user"`
	Argument json.RawMessage `json:"argument"`
}

func newRequest(req *http.Request) (r service.Request, err errors.CodeError) {
	pathItems := strings.Split(req.URL.Path, "/")
	if len(pathItems) != 3 {
		err = errors.NotAcceptable("fns: invalid request url path")
		return
	}
	sn := pathItems[1]
	fn := pathItems[2]
	id := req.Header.Get("X-Fns-Request-Id")
	if id == "" {
		err = errors.NotAcceptable("fns: X-Fns-Request-Id was not found in header")
		return
	}
	// body
	body, readBodyErr := ioutil.ReadAll(req.Body)
	if readBodyErr != nil {
		err = errors.NotAcceptable("fns: invalid request body")
		return
	}
	bodyVerified := false
	body, bodyVerified = decodeRequestBody(body)
	if !bodyVerified {
		err = errors.NotAcceptable("fns: internal request body is not verified")
		return
	}
	ir := &internalRequest{}
	decodeIrErr := json.Unmarshal(body, ir)
	if decodeIrErr != nil {
		err = errors.NotAcceptable("fns: decode internal request body failed").WithCause(decodeIrErr)
		return
	}
	local := &requestLocal{
		values: make(map[string]interface{}),
		remote: nil,
	}
	if ir.Local != nil && json.Validate(ir.Local) {
		local.remote = json.NewObjectFromBytes(ir.Local)
	} else {
		local.remote = json.NewObject()
	}
	user := service.NewRequestUser("", json.NewObject())
	if ir.User != nil {
		decodeUserErr := json.Unmarshal(ir.User, user)
		if decodeUserErr != nil {
			err = errors.NotAcceptable("fns: decode internal request body failed").WithCause(decodeUserErr)
			return
		}
	}
	remoteIp := req.Header.Get("X-Real-Ip")
	if remoteIp == "" {
		forwarded := req.Header.Get("X-Forwarded-For")
		if forwarded != "" {
			forwardedIps := strings.Split(forwarded, ",")
			remoteIp = strings.TrimSpace(forwardedIps[len(forwardedIps)-1])
		}
	}
	if remoteIp == "" {
		remoteIp = req.RemoteAddr
		if remoteIp != "" {
			if strings.Index(remoteIp, ".") > 0 && strings.Index(remoteIp, ":") > 0 {
				remoteIp = remoteIp[0:strings.Index(remoteIp, ":")]
			}
		}
	}
	buf := bytebufferpool.Get()
	_, _ = buf.Write([]byte(sn + fn))
	authorization := req.Header.Get("Authorization")
	if authorization != "" {
		_, _ = buf.Write([]byte(authorization))
	}
	if remoteIp != "" {
		_, _ = buf.Write([]byte(remoteIp))
	}
	userAgent := req.UserAgent()
	if userAgent != "" {
		_, _ = buf.Write([]byte(userAgent))
	}
	_, _ = buf.Write(body)
	hashCode := xxhash.Sum64(buf.Bytes())
	bytebufferpool.Put(buf)
	r = &request{
		id:       id,
		remoteIp: remoteIp,
		user:     user,
		local:    local,
		header:   service.NewRequestHeader(req.Header),
		service:  sn,
		fn:       fn,
		argument: service.NewArgument(ir.Argument),
		hashCode: hashCode,
	}
	return
}

type requestLocal struct {
	values map[string]interface{}
	remote *json.Object
}

func (local *requestLocal) Scan(key string, value interface{}) (has bool, err errors.CodeError) {
	v, exist := local.values[key]
	if !exist {
		if local.remote.Contains(key) {
			getErr := local.remote.Get(key, value)
			if getErr != nil {
				err = errors.Warning("fns: request local scan failed").WithCause(getErr).WithMeta("key", key)
				return
			}
			local.values[key] = value
		}
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
	if local.remote.Contains(key) {
		_ = local.remote.Remove(key)
	}
}

func (local *requestLocal) MarshalJSON() (p []byte, err error) {
	obj := json.NewObject()
	mergeErr := obj.Merge(local.remote)
	if mergeErr != nil {
		err = mergeErr
		return
	}
	for k, v := range local.values {
		if obj.Contains(k) {
			continue
		}
		putErr := obj.Put(k, v)
		if putErr != nil {
			err = putErr
			return
		}
	}
	p, err = obj.MarshalJSON()
	return
}

type request struct {
	id       string
	remoteIp string
	user     service.RequestUser
	local    service.RequestLocal
	header   service.RequestHeader
	service  string
	fn       string
	argument service.Argument
	hashCode uint64
}

func (r *request) Id() (id string) {
	id = r.id
	return
}

func (r *request) Internal() (ok bool) {
	ok = true
	return
}

func (r *request) User() (user service.RequestUser) {
	user = r.user
	return
}

func (r *request) SetUser(id string, attributes *json.Object) {
	r.user = service.NewRequestUser(id, attributes)
}

func (r *request) Local() (local service.RequestLocal) {
	local = r.local
	return
}

func (r *request) RemoteIp() (v string) {
	v = r.remoteIp
	return
}

func (r *request) Authorization() (v string) {
	v = r.header.Get("Authorization")
	return
}

func (r *request) Header() (header service.RequestHeader) {
	header = r.header
	return
}

func (r *request) Fn() (service string, fn string) {
	service, fn = r.service, r.fn
	return
}

func (r *request) Argument() (argument service.Argument) {
	argument = r.argument
	return
}

func (r *request) Hash() (code uint64) {
	code = r.hashCode
	return
}
