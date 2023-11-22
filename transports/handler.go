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

package transports

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/logs"
)

type Handler interface {
	Handle(w ResponseWriter, r Request)
}

type HandlerFunc func(ResponseWriter, Request)

func (f HandlerFunc) Handle(w ResponseWriter, r Request) {
	f(w, r)
}

type MuxHandlerOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type MuxHandler interface {
	Name() string
	Construct(options MuxHandlerOptions) error
	Match(ctx context.Context, method []byte, path []byte, header Header) bool
	Handler
}

func NewMux() *Mux {
	return &Mux{
		handlers: make([]MuxHandler, 0, 1),
	}
}

type Mux struct {
	handlers []MuxHandler
}

func (mux *Mux) Add(handler MuxHandler) {
	mux.handlers = append(mux.handlers, handler)
}

func (mux *Mux) Handle(w ResponseWriter, r Request) {
	for _, handler := range mux.handlers {
		matched := handler.Match(r, r.Method(), r.Path(), r.Header())
		if matched {
			handler.Handle(w, r)
			return
		}
	}
	w.Failed(errors.NotFound("fns: not found").WithMeta("handler", "mux"))
}
