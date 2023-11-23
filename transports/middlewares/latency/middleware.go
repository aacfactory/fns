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

package latency

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"time"
)

func Middleware() transports.Middleware {
	return &middleware{}
}

type Config struct {
	Enabled bool `json:"enabled"`
}

type middleware struct {
	enabled bool
}

func (m *middleware) Name() string {
	return "latency"
}

func (m *middleware) Construct(options transports.MiddlewareOptions) error {
	config := Config{}
	err := options.Config.As(&config)
	if err != nil {
		err = errors.Warning("fns: construct latency middleware failed").WithCause(err)
		return err
	}
	m.enabled = config.Enabled
	return nil
}

func (m *middleware) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
		beg := time.Time{}
		if m.enabled {
			beg = time.Now()
		}
		next.Handle(w, r)
		if w.Hijacked() {
			return
		}
		latency := time.Now().Sub(beg)
		w.Header().Set(transports.HandleLatencyHeaderName, bytex.FromString(latency.String()))
	})
}

func (m *middleware) Close() (err error) {
	return
}
