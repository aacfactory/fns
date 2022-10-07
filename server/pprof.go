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

package server

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/secret"
	"github.com/aacfactory/json"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"strings"
)

const (
	httpPprofPath        = "/debug/pprof"
	httpPprofCmdlinePath = "/debug/pprof/cmdline"
	httpPprofProfilePath = "/debug/pprof/profile"
	httpPprofSymbolPath  = "/debug/pprof/symbol"
	httpPprofTracePath   = "/debug/pprof/trace"
)

type PprofArgument struct {
	Password string `json:"password"`
	Sec      int    `json:"sec"`
}

type PprofHandlerOptions struct {
	Password string
}

func PprofHandler() (h Handler) {
	h = &pprofHandler{}
	return
}

type pprofHandler struct {
	password string
}

func (h *pprofHandler) Name() (name string) {
	name = "pprof"
	return
}

func (h *pprofHandler) Build(options *HandlerOptions) (err error) {
	password := ""
	_, _ = options.Config.Get("password", &password)
	if password == "" {
		err = errors.Warning("fns: create pprof handler failed, password is required")
		return
	}
	h.password = password
	return
}

func (h *pprofHandler) Handle(writer http.ResponseWriter, request *http.Request) (ok bool) {
	if !(request.Method == http.MethodPost && request.Header.Get(httpContentType) == httpContentTypeJson) {
		return
	}
	if strings.Index(request.URL.Path, "/debug/pprof") != 0 {
		return
	}
	body, readBodyErr := ioutil.ReadAll(request.Body)
	if readBodyErr != nil {
		h.failed(writer, errors.NotAcceptable("fns: invalid request body").WithCause(readBodyErr))
		ok = true
		return
	}
	arg := PprofArgument{}
	decodeErr := json.Unmarshal(body, &arg)
	if decodeErr != nil {
		h.failed(writer, errors.NotAcceptable("fns: read request body").WithCause(decodeErr))
		ok = true
		return
	}

	password := arg.Password
	if password == "" {
		return
	}
	ok = true
	if secret.ValidatePassword([]byte(h.password), []byte(password)) {
		writer.WriteHeader(http.StatusForbidden)
		return
	}
	request.Form.Set("sec", fmt.Sprintf("%d", arg.Sec))
	switch request.URL.Path {
	case httpPprofPath:
		pprof.Index(writer, request)
		break
	case httpPprofCmdlinePath:
		pprof.Cmdline(writer, request)
		break
	case httpPprofProfilePath:
		pprof.Profile(writer, request)
		break
	case httpPprofSymbolPath:
		pprof.Symbol(writer, request)
		break
	case httpPprofTracePath:
		pprof.Trace(writer, request)
		break
	default:
		ok = false
		return
	}
	return
}

func (h *pprofHandler) Close() {
}

func (h *pprofHandler) failed(writer http.ResponseWriter, codeErr errors.CodeError) {
	writer.Header().Set(httpServerHeader, httpServerHeaderValue)
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(codeErr.Code())
	p, _ := json.Marshal(codeErr)
	_, _ = writer.Write(p)
}
