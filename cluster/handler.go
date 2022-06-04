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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"io/ioutil"
	"net/http"
)

const (
	contentType = "application/fns+cluster"
	joinPath    = "/cluster/join"
	leavePath   = "/cluster/leave"
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
			signedBody, readBodyErr := ioutil.ReadAll(request.Body)
			if readBodyErr != nil {
				handler.failed(writer, errors.BadRequest("fns: read request body failed").WithCause(readBodyErr))
				return
			}
			body, bodyOk := decodeRequestBody(signedBody)
			if !bodyOk {
				handler.failed(writer, errors.BadRequest("fns: read request body failed").WithCause(fmt.Errorf("invalid body")))
				return
			}
			requestPath := request.URL.Path
			switch requestPath {
			case joinPath:
				result, joinErr := handler.handleJoin(body)
				if joinErr != nil {
					handler.failed(writer, joinErr)
					return
				}
				handler.succeed(writer, result)
				break
			case leavePath:
				leaveErr := handler.handleLeave(body)
				if leaveErr != nil {
					handler.failed(writer, leaveErr)
					return
				}
				handler.succeed(writer, nil)
				break
			default:
				handler.failed(writer, errors.NotFound("fns: not found"))
				break
			}
		} else {
			h(writer, request)
		}
	})
}

func (handler *Handler) succeed(response http.ResponseWriter, body []byte) {
	response.Header().Set("Server", "Fns")
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(200)
	if body == nil || len(body) == 0 {
		body = []byte("{}")
	}
	_, _ = response.Write(encodeResponseBody(body, nil))
}

func (handler *Handler) failed(response http.ResponseWriter, codeErr errors.CodeError) {
	response.Header().Set("Server", "Fns")
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(codeErr.Code())
	_, _ = response.Write(encodeResponseBody(nil, codeErr))
}

type joinResult struct {
	Node    *Node   `json:"node"`
	Members []*Node `json:"members"`
}

func (handler *Handler) handleJoin(body []byte) (result []byte, err errors.CodeError) {
	node := &Node{}
	decodeErr := json.Unmarshal(body, node)
	if decodeErr != nil {
		err = errors.BadRequest("fns: json failed").WithCause(decodeErr)
		return
	}
	if node.Id == "" {
		err = errors.BadRequest("fns: json failed").WithCause(fmt.Errorf("node id is empty"))
		return
	}
	if node.Address == "" {
		err = errors.BadRequest("fns: json failed").WithCause(fmt.Errorf("node address is empty"))
		return
	}
	if (node.Services == nil || len(node.Services) == 0) && (node.InternalServices == nil || len(node.InternalServices) == 0) {
		err = errors.BadRequest("fns: json failed").WithCause(fmt.Errorf("node services is empty"))
		return
	}
	if handler.manager.registrations.containsMember(node) {
		return
	}
	members := handler.manager.registrations.members()
	node.client = handler.manager.client
	handler.manager.registrations.register(node)
	jr := &joinResult{
		Node:    handler.manager.Node(),
		Members: members,
	}
	jrp, encodeErr := json.Marshal(jr)
	if encodeErr != nil {
		err = errors.ServiceError("fns: json failed").WithCause(encodeErr)
		return
	}
	result = jrp
	return
}

func (handler *Handler) handleLeave(body []byte) (err errors.CodeError) {
	node := &Node{}
	decodeErr := json.Unmarshal(body, node)
	if decodeErr != nil {
		err = errors.BadRequest("fns: leave failed").WithCause(decodeErr)
		return
	}
	if node.Id == "" {
		err = errors.BadRequest("fns: leave failed").WithCause(fmt.Errorf("node id is empty"))
		return
	}
	if node.Address == "" {
		err = errors.BadRequest("fns: leave failed").WithCause(fmt.Errorf("node address is empty"))
		return
	}
	if (node.Services == nil || len(node.Services) == 0) && (node.InternalServices == nil || len(node.InternalServices) == 0) {
		err = errors.BadRequest("fns: leave failed").WithCause(fmt.Errorf("node services is empty"))
		return
	}
	handler.manager.Registrations().deregister(node)
	return
}
