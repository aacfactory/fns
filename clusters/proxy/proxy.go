/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package proxy

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
)

var (
	handlerPathPrefix = []byte("/proxy")
	contentType       = []byte("application/json+proxy")
)

var (
	ErrInvalidPath         = errors.Warning("fns: invalid path")
	ErrInvalidBody         = errors.Warning("fns: invalid body")
	ErrSignatureLost       = errors.New(488, "***SIGNATURE LOST***", "X-Fns-Signature was required")
	ErrSignatureUnverified = errors.New(458, "***SIGNATURE INVALID***", "X-Fns-Signature was invalid")
)

func NewHandler(signature signatures.Signature, manager services.EndpointsManager, shared shareds.Shared) transports.MuxHandler {
	return &Handler{
		log:       nil,
		signature: signature,
		shared:    NewSharedHandler(shared),
		manager:   NewManagerHandler(manager),
	}
}

type Handler struct {
	log       logs.Logger
	signature signatures.Signature
	shared    transports.Handler
	manager   transports.Handler
}

func (handler *Handler) Name() string {
	return "development"
}

func (handler *Handler) Construct(options transports.MuxHandlerOptions) error {
	handler.log = options.Log
	return nil
}

func (handler *Handler) Match(_ context.Context, method []byte, path []byte, header transports.Header) bool {
	ok := bytes.Equal(method, transports.MethodPost) &&
		bytes.Index(path, handlerPathPrefix) == 0 &&
		len(header.Get(transports.SignatureHeaderName)) != 0 &&
		bytes.Equal(header.Get(transports.ContentTypeHeaderName), contentType)
	return ok
}

func (handler *Handler) Handle(w transports.ResponseWriter, r transports.Request) {
	path := r.Path()
	// sign
	sign := r.Header().Get(transports.SignatureHeaderName)
	if len(sign) == 0 {
		w.Failed(ErrSignatureLost.WithMeta("path", bytex.ToString(path)))
		return
	}
	body, bodyErr := r.Body()
	if bodyErr != nil {
		w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)))
		return
	}
	if !handler.signature.Verify(body, sign) {
		w.Failed(ErrSignatureUnverified.WithMeta("path", bytex.ToString(path)))
		return
	}
	// match
	if bytes.Equal(path, managerHandlerPath) {
		handler.manager.Handle(w, r)
	} else if bytes.Equal(path, sharedHandlerPath) {
		handler.shared.Handle(w, r)
	} else {
		w.Failed(ErrInvalidPath.WithMeta("path", bytex.ToString(path)))
	}
}
