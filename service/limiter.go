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
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"strings"
	"time"
)

var (
	ErrTooMayRequest = errors.New(http.StatusTooManyRequests, "***TOO MANY REQUEST***", "fns: too may request, try again later.")
)

type RateLimitCounterOptions struct {
	AppId      string
	AppName    string
	AppVersion versions.Version
	AppStatus  *Status
	Log        logs.Logger
	Config     configures.Config
	Discovery  EndpointDiscovery
	Shared     Shared
}

type RateLimitCounter interface {
	Name() (name string)
	Build(options RateLimitCounterOptions) (err error)
	Incr(ctx context.Context, key string) (ok bool, err error)
	Decr(ctx context.Context, key string) (err error)
	Close()
}

const (
	rateLimitMiddlewareName = "rateLimit"
)

type rateLimitMiddlewareConfig struct {
	RetryAfter int             `json:"retryAfter"`
	Counter    json.RawMessage `json:"counter"`
}

func RateLimitMiddleware() TransportMiddleware {
	return &rateLimitMiddleware{}
}

func CustomizeRateLimitMiddleware(counter RateLimitCounter) TransportMiddleware {
	return &rateLimitMiddleware{}
}

type rateLimitMiddleware struct {
	appId      string
	appName    string
	appVersion versions.Version
	appStatus  *Status
	log        logs.Logger
	config     configures.Config
	discovery  EndpointDiscovery
	shared     Shared
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
		AppId:      middleware.appId,
		AppName:    middleware.appName,
		AppVersion: middleware.appVersion,
		AppStatus:  middleware.appStatus,
		Log:        middleware.log.With("counter", middleware.counter.Name()),
		Config:     counterConfig,
		Discovery:  middleware.discovery,
		Shared:     middleware.shared,
	})
	if counterErr != nil {
		err = errors.Warning("fns: rate limit middleware build failed").WithCause(counterErr)
		return
	}
	return
}

func (middleware *rateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deviceId := r.Header.Get(httpDeviceIdHeader)
		ok, incrErr := middleware.counter.Incr(r.Context(), deviceId)
		if incrErr != nil {
			p, _ := json.Marshal(errors.Warning("fns: rate limit counter incr failed").WithCause(incrErr))
			w.WriteHeader(555)
			_, _ = w.Write(p)
			return
		}
		if !ok {
			p, _ := json.Marshal(ErrTooMayRequest)
			w.Header().Set(httpResponseRetryAfter, middleware.retryAfter)
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write(p)
			return
		}
		next.ServeHTTP(w, r)
		repayErr := middleware.counter.Decr(r.Context(), deviceId)
		if repayErr != nil && middleware.log.ErrorEnabled() {
			middleware.log.Error().Cause(
				errors.Warning("fns: rate limit counter decr failed").WithCause(repayErr),
			).With("middleware", middleware.Name()).Message("fns: rate limit counter decr failed")
		}
		return
	})
}

func (middleware *rateLimitMiddleware) Close() {
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
	shared Shared
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

func (counter *rateLimitCounter) Incr(ctx context.Context, key string) (ok bool, err error) {
	if key == "" {
		err = errors.Warning("fns: incr failed").WithCause(errors.Warning("key is nil")).WithMeta("counter", counter.Name())
		return
	}
	key = counter.preflight(key)
	n, incrErr := counter.shared.Store().Incr(ctx, bytex.FromString(key), 1)
	if incrErr != nil {
		err = errors.Warning("fns: incr failed").WithCause(incrErr).WithMeta("counter", counter.Name())
		return
	}
	ok = n <= counter.max
	if !ok {
		return
	}
	if n == 1 && counter.window > 0 {
		expireErr := counter.shared.Store().ExpireKey(ctx, bytex.FromString(key), counter.window)
		if expireErr != nil {
			err = errors.Warning("fns: incr failed").WithCause(expireErr).WithMeta("counter", counter.Name())
			return
		}
	}
	return
}

func (counter *rateLimitCounter) Decr(ctx context.Context, key string) (err error) {
	if key == "" {
		err = errors.Warning("fns: decr failed").WithCause(errors.Warning("key is nil")).WithMeta("counter", counter.Name())
		return
	}
	key = counter.preflight(key)
	n, incrErr := counter.shared.Store().Incr(ctx, bytex.FromString(key), -1)
	if incrErr != nil {
		err = errors.Warning("fns: decr failed").WithCause(incrErr).WithMeta("counter", counter.Name())
		return
	}
	if n < 0 {
		_ = counter.shared.Store().Remove(ctx, bytex.FromString(key))
	}
	return
}

func (counter *rateLimitCounter) preflight(key string) string {
	if counter.window == 0 {
		return key
	}
	return fmt.Sprintf("%s:%d", key, time.Now().Truncate(counter.window).Unix())
}

func (counter *rateLimitCounter) Close() {}
