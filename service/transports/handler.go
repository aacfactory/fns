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

package transports

import (
	"context"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/logs"
)

type HandlerOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type Handler interface {
	Name() (name string)
	Build(ctx context.Context, options HandlerOptions) (err error)
	Handle(w ResponseWriter, r *Request)
	Close() (err error)
}

type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) Handle(w ResponseWriter, r *Request) {
	f(w, r)
}
