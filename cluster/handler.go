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
	"github.com/aacfactory/json"
	"io/ioutil"
	"net/http"
)

const (
	contentType = "application/fns+cluster"
	joinPath    = "/cluster/join"
	leavePath   = "/cluster/leave"
	updatePath  = "/cluster/resource"
)

func NewHandler(manager *Manager) *Handler {
	return &Handler{manager: manager}
}

type Handler struct {
	manager *Manager
}

func (handler *Handler) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if contentType == request.Header.Get("Content-Type") && http.MethodPost == request.Method {
			requestPath := request.URL.Path
			switch requestPath {
			case joinPath:
				body, readErr := ioutil.ReadAll(request.Body)
				if readErr != nil {
					writer.WriteHeader(500)
					_, _ = writer.Write(json.UnsafeMarshal(errors.Warning("fns: join failed, read request body failed").WithCause(readErr)))
					return
				}
				result, joinErr := handler.handleJoin(body)
				if joinErr != nil {
					writer.WriteHeader(500)
					_, _ = writer.Write(json.UnsafeMarshal(joinErr))
					return
				}
				writer.WriteHeader(200)
				_, _ = writer.Write(result)
			case leavePath:
				body, readErr := ioutil.ReadAll(request.Body)
				if readErr != nil {
					writer.WriteHeader(500)
					_, _ = writer.Write(json.UnsafeMarshal(errors.Warning("fns: leave failed, read request body failed").WithCause(readErr)))
					return
				}
				handler.handleLeave(body)
			case updatePath:
				body, readErr := ioutil.ReadAll(request.Body)
				if readErr != nil {
					writer.WriteHeader(500)
					_, _ = writer.Write(json.UnsafeMarshal(errors.Warning("fns: join failed, read request body failed").WithCause(readErr)))
					return
				}
				result, joinErr := handler.handleUpdate(body)
				if joinErr != nil {
					writer.WriteHeader(500)
					_, _ = writer.Write(json.UnsafeMarshal(joinErr))
					return
				}
				writer.WriteHeader(200)
				_, _ = writer.Write(result)
			default:
				writer.WriteHeader(404)
			}
		} else {
			h(writer, request)
		}
	})
}

func (handler *Handler) handleJoin(body []byte) (result []byte, err errors.CodeError) {

	return
}

func (handler *Handler) handleLeave(body []byte) {

	return
}

func (handler *Handler) handleUpdate(body []byte) (result []byte, err errors.CodeError) {

	return
}
