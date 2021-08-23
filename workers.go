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
	"context"
	"fmt"
	"github.com/aacfactory/workers"
	"github.com/valyala/fasthttp"
	"strings"
	"sync"
)

type FnExecutorHandler interface {
	Handle(ctx Context, arg Argument)
}

type fnWorkers struct {
	wp workers.Workers
}

func (fw *fnWorkers) Execute(request *fasthttp.Request) (result chan *fnWorkResult, accepted bool) {

	return
}

func (fw *fnWorkers) Start() {
	fw.wp.Start()
	return
}

func (fw *fnWorkers) Stop() {
	fw.wp.Stop()
	return
}

type fnWorkResult struct {
	ok   bool
	data interface{}
}

type fnWorkPayload struct {
	request *fasthttp.Request
	result  chan *fnWorkResult
}

const (
	fnWorkActionExecute = "fc"
)

type fnWorkersUnitHandler struct {
	log                    Logs
	discovery              Discovery
	authorizationValidator AuthorizationValidator
	permissionValidator    PermissionValidator
	requestHandler         ServiceRequestHandler
	payloadPool            sync.Pool
}

func (handler *fnWorkersUnitHandler) Handle(action string, _payload interface{}) {
	payload, typeOk := _payload.(*fnWorkPayload)
	if !typeOk {
		return
	}
	defer handler.payloadPool.Put(payload)
	if action != fnWorkActionExecute {
		payload.result <- &fnWorkResult{
			ok:   false,
			data: NotAcceptableArgumentError(fmt.Sprintf("%s action is not support", action)),
		}
		return
	}
	request := payload.request
	namespace := strings.TrimSpace(string(request.Header.Peek(httpHeaderNamespace)))
	if namespace == "" {
		payload.result <- &fnWorkResult{
			ok:   false,
			data: NotAcceptableArgumentError(fmt.Sprintf("%s is not set", httpHeaderNamespace)),
		}
		return
	}
	fnName := strings.TrimSpace(string(request.Header.Peek(httpHeaderFnName)))
	if fnName == "" {
		payload.result <- &fnWorkResult{
			ok:   false,
			data: NotAcceptableArgumentError(fmt.Sprintf("%s is not set", httpHeaderFnName)),
		}
		return
	}
	// user -> header: httpHeaderAuthorization
	// meta -> header: X-Fns-Meta meta.Encode()
	authorization := strings.TrimSpace(string(request.Header.Peek(httpHeaderAuthorization)))

	requestId := strings.TrimSpace(string(request.Header.Peek(httpHeaderRequestId)))
	ctx := newContext(context.TODO(), handler.log, handler.discovery, namespace, fnName, authorization, requestId)

}
