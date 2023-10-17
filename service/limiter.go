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

package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/shareds"
	transports2 "github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"strconv"
	"strings"
	"time"
)

type RateLimitCounterOptions struct {
	AppId      string
	AppName    string
	AppVersion versions.Version
	AppStatus  *Status
	Log        logs.Logger
	Config     configures.Config
	Discovery  EndpointDiscovery
	Shared     shareds.Shared
}

// todo use golang.org/x/time/rate，外层用rate，内层是每个device的
type RateLimitCounter interface {
	Name() (name string)
	Build(options RateLimitCounterOptions) (err error)
	Incr(ctx context.Context, key []byte) (ok bool, err error)
	Decr(ctx context.Context, key []byte) (err error)
	Close()
}

const (
	rateLimitMiddlewareName = "rateLimit"
)

type rateLimitMiddlewareConfig struct {
	RetryAfter int             `json:"retryAfter" yaml:"retryAfter,omitempty"`
	Counter    json.RawMessage `json:"counter" yaml:"counter,omitempty"`
}

func RateLimitMiddleware() TransportMiddleware {
	return &rateLimitMiddleware{
		counter: &rateLimitCounter{},
	}
}

func CustomizeRateLimitMiddleware(counter RateLimitCounter) TransportMiddleware {
	return &rateLimitMiddleware{
		counter: counter,
	}
}

type rateLimitMiddleware struct {
	log        logs.Logger
	counter    RateLimitCounter
	retryAfter string
}

func (middleware *rateLimitMiddleware) Name() (name string) {
	name = rateLimitMiddlewareName
	return
}

func (middleware *rateLimitMiddleware) Build(options TransportMiddlewareOptions) (err error) {
	middleware.log = options.Log

	config := rateLimitMiddlewareConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: rate limit middleware build failed").WithCause(configErr)
		return
	}
	if config.RetryAfter > 0 {
		middleware.retryAfter = fmt.Sprintf("%d", config.RetryAfter)
	} else {
		middleware.retryAfter = "10"
	}
	if config.Counter == nil || len(config.Counter) == 0 {
		config.Counter = json.RawMessage{'{', '}'}
	}
	counterConfig, counterConfigErr := configures.NewJsonConfig(config.Counter)
	if counterConfigErr != nil {
		err = errors.Warning("fns: rate limit middleware build failed").WithCause(counterConfigErr)
		return
	}
	counterErr := middleware.counter.Build(RateLimitCounterOptions{
		AppId:      options.Runtime.AppId(),
		AppName:    options.Runtime.AppName(),
		AppVersion: options.Runtime.AppVersion(),
		AppStatus:  options.Runtime.AppStatus(),
		Log:        middleware.log.With("counter", middleware.counter.Name()),
		Config:     counterConfig,
		Discovery:  options.Runtime.Discovery(),
		Shared:     options.Runtime.Shared(),
	})
	if counterErr != nil {
		err = errors.Warning("fns: rate limit middleware build failed").WithCause(counterErr)
		return
	}
	return
}

func (middleware *rateLimitMiddleware) Handler(next transports2.Handler) transports2.Handler {
	return transports2.HandlerFunc(func(w transports2.ResponseWriter, r *transports2.Request) {
		deviceId := r.Header().Get(httpDeviceIdHeader)
		key := fmt.Sprintf("rateLimit/%s", deviceId)
		ok, incrErr := middleware.counter.Incr(r.Context(), bytex.FromString(key))
		if incrErr != nil {
			w.Failed(errors.Warning("fns: rate limit counter incr failed").WithCause(incrErr))
			return
		}
		if !ok {
			w.Header().Set(httpResponseRetryAfter, middleware.retryAfter)
			w.Failed(ErrTooMayRequest)
			return
		}
		next.Handle(w, r)
		repayErr := middleware.counter.Decr(r.Context(), bytex.FromString(key))
		if repayErr != nil && middleware.log.ErrorEnabled() {
			middleware.log.Error().Cause(
				errors.Warning("fns: rate limit counter decr failed").WithCause(repayErr),
			).With("middleware", middleware.Name()).Message("fns: rate limit counter decr failed")
		}
		return
	})
}

func (middleware *rateLimitMiddleware) Close() (err error) {
	middleware.counter.Close()
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	defaultRateLimitCounterName = "default"
)

type rateLimitCounterConfig struct {
	Max    uint64 `json:"max"`
	Window string `json:"window"`
}

type rateLimitCounter struct {
	max    int64
	window time.Duration
	shared shareds.Shared
}

func (counter *rateLimitCounter) Name() (name string) {
	name = defaultRateLimitCounterName
	return
}

func (counter *rateLimitCounter) Build(options RateLimitCounterOptions) (err error) {
	config := rateLimitCounterConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: build default rate limit counter failed").WithCause(configErr)
		return
	}
	if config.Max == 0 {
		config.Max = 5
	}
	counter.max = int64(config.Max)
	if config.Window != "" {
		counter.window, err = time.ParseDuration(strings.TrimSpace(config.Window))
		if err != nil {
			err = errors.Warning("fns: build default rate limit counter failed").WithCause(errors.Warning("window must be time.Duration format")).WithCause(err)
			return
		}
		if counter.window < 0 {
			err = errors.Warning("fns: build default rate limit counter failed").WithCause(errors.Warning("window must be greater than 0"))
			return
		}
	}
	counter.shared = options.Shared
	return err
}

func (counter *rateLimitCounter) Incr(ctx context.Context, key []byte) (ok bool, err error) {
	if len(key) == 0 {
		err = errors.Warning("fns: incr failed").WithCause(errors.Warning("key is nil")).WithMeta("counter", counter.Name())
		return
	}
	key = counter.preflight(key)
	n, incrErr := counter.shared.Store().Incr(ctx, key, 1, shareds.SystemScope())
	if incrErr != nil {
		err = errors.Warning("fns: incr failed").WithCause(incrErr).WithMeta("counter", counter.Name())
		return
	}
	ok = n <= counter.max
	if !ok {
		return
	}
	if n == 1 && counter.window > 0 {
		expireErr := counter.shared.Store().ExpireKey(ctx, key, counter.window, shareds.SystemScope())
		if expireErr != nil {
			err = errors.Warning("fns: incr failed").WithCause(expireErr).WithMeta("counter", counter.Name())
			return
		}
	}
	return
}

func (counter *rateLimitCounter) Decr(ctx context.Context, key []byte) (err error) {
	if len(key) == 0 {
		err = errors.Warning("fns: decr failed").WithCause(errors.Warning("key is nil")).WithMeta("counter", counter.Name())
		return
	}
	key = counter.preflight(key)
	n, incrErr := counter.shared.Store().Incr(ctx, key, -1, shareds.SystemScope())
	if incrErr != nil {
		err = errors.Warning("fns: decr failed").WithCause(incrErr).WithMeta("counter", counter.Name())
		return
	}
	if n < 0 {
		_ = counter.shared.Store().Remove(ctx, key, shareds.SystemScope())
	}
	return
}

var (
	rateLimitCounterKeySep = []byte{':'}
)

func (counter *rateLimitCounter) preflight(key []byte) []byte {
	if counter.window == 0 {
		return key
	}
	window := bytex.FromString(strconv.FormatInt(time.Now().Truncate(counter.window).Unix(), 10))
	return bytes.Join([][]byte{key, window}, rateLimitCounterKeySep)
}

func (counter *rateLimitCounter) Close() {}
