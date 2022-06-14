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
	"crypto/md5"
	"encoding/hex"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"io/ioutil"
	"net/http"
	"strings"
)

type internalRequest struct {
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
	user := service.NewRequestUser("", json.NewObject())
	if ir.User != nil {
		decodeUserErr := json.Unmarshal(ir.User, user)
		if decodeUserErr != nil {
			err = errors.NotAcceptable("fns: decode internal request body failed").WithCause(decodeUserErr)
			return
		}
	}
	remoteIp := req.RemoteAddr
	if remoteIp != "" {
		if strings.Index(remoteIp, ".") > 0 && strings.Index(remoteIp, ":") > 0 {
			// ip:port
			remoteIp = remoteIp[0:strings.Index(remoteIp, ":")]
		}
	}
	hash := md5.New()
	hash.Write(body)
	hashCode := hex.EncodeToString(hash.Sum(nil))
	r = &request{
		id:       id,
		remoteIp: remoteIp,
		user:     user,
		header:   service.NewRequestHeader(req.Header),
		service:  sn,
		fn:       fn,
		argument: service.NewArgument(ir.Argument),
		hashCode: hashCode,
	}
	return
}

type request struct {
	id       string
	remoteIp string
	user     service.RequestUser
	header   service.RequestHeader
	service  string
	fn       string
	argument service.Argument
	hashCode string
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

func (r *request) Hash() (code string) {
	code = r.hashCode
	return
}
