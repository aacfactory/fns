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
	"github.com/aacfactory/fns/commons/secret"
	"net/http"
	"net/http/pprof"
)

const (
	httpPprofPath        = "/debug/pprof"
	httpPprofCmdlinePath = "/debug/pprof/cmdline"
	httpPprofProfilePath = "/debug/pprof/profile"
	httpPprofSymbolPath  = "/debug/pprof/symbol"
	httpPprofTracePath   = "/debug/pprof/trace"
)

type PprofHandlerOptions struct {
	Password string
}

func NewPprofHandler(options PprofHandlerOptions) (h Handler) {
	h = &pprofHandler{
		password: options.Password,
	}
	return
}

type pprofHandler struct {
	password string
}

func (h *pprofHandler) Handle(writer http.ResponseWriter, request *http.Request) (ok bool) {
	if request.Method != http.MethodGet {
		return
	}
	password := request.URL.Query().Get("password")
	if password == "" {
		return
	}
	ok = true
	if secret.ValidatePassword([]byte(h.password), []byte(password)) {
		writer.WriteHeader(http.StatusForbidden)
		return
	}
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
