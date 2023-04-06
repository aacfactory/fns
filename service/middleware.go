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
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/valyala/bytebufferpool"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type TransportMiddlewareOptions struct {
	AppId      string
	AppName    string
	AppVersion versions.Version
	AppStatus  *Status
	Log        logs.Logger
	Config     configures.Config
	Discovery  EndpointDiscovery
	Shared     Shared
}

type TransportMiddleware interface {
	Name() (name string)
	Build(options TransportMiddlewareOptions) (err error)
	Handler(next transports.Handler) transports.Handler
	Close()
}

type transportMiddlewaresOptions struct {
	Runtime *Runtime
	Cors    *transports.CorsConfig
	Config  configures.Config
}

func newTransportMiddlewares(options transportMiddlewaresOptions) *transportMiddlewares {
	middlewares := make([]TransportMiddleware, 0, 1)
	middlewares = append(middlewares, newTransportApplicationMiddleware(options.Runtime.status))
	return &transportMiddlewares{
		config:      options.Config,
		runtime:     options.Runtime,
		cors:        options.Cors,
		middlewares: make([]TransportMiddleware, 0, 1),
	}
}

type transportMiddlewares struct {
	config      configures.Config
	runtime     *Runtime
	cors        *transports.CorsConfig
	middlewares []TransportMiddleware
}

func (middlewares *transportMiddlewares) Append(middleware TransportMiddleware) (err error) {
	if middleware == nil {
		return
	}
	name := strings.TrimSpace(middleware.Name())
	if name == "" {
		err = errors.Warning("append middleware failed").WithCause(errors.Warning("one of middlewares has no name"))
		return
	}
	pos := sort.Search(len(middlewares.middlewares), func(i int) bool {
		return middlewares.middlewares[i].Name() == name
	})
	if pos < len(middlewares.middlewares) {
		err = errors.Warning("append middleware failed").WithCause(errors.Warning("this middleware was appended")).WithMeta("middleware", name)
		return
	}
	config, exist := middlewares.config.Node(name)
	if !exist {
		config, _ = configures.NewJsonConfig([]byte{'{', '}'})
	}
	buildErr := middleware.Build(TransportMiddlewareOptions{
		AppId:      middlewares.runtime.AppId(),
		AppName:    middlewares.runtime.AppName(),
		AppVersion: middlewares.runtime.AppVersion(),
		AppStatus:  middlewares.runtime.AppStatus(),
		Log:        middlewares.runtime.log.With("transports", "middlewares").With("middleware", name),
		Config:     config,
		Discovery:  middlewares.runtime.Discovery(),
		Shared:     middlewares.runtime.Shared(),
	})
	if buildErr != nil {
		err = errors.Warning("append middleware failed").WithCause(buildErr).WithMeta("middleware", name)
		return
	}
	middlewares.middlewares = append(middlewares.middlewares, middleware)
	return
}

func (middlewares *transportMiddlewares) Handler(handlers *transportHandlers) transports.Handler {
	var handler transports.Handler = handlers
	for i := len(middlewares.middlewares) - 1; i > -1; i-- {
		handler = middlewares.middlewares[i].Handler(handler)
	}
	return newCorsHandler(middlewares.cors, newRuntimeMiddleware(middlewares.runtime, handler))
}

// +-------------------------------------------------------------------------------------------------------------------+

type responseWriter struct {
	status int
	header http.Header
	buf    *bytebufferpool.ByteBuffer
}

func (r *responseWriter) Header() http.Header {
	return r.header
}

func (r *responseWriter) Write(p []byte) (int, error) {
	return r.buf.Write(p)
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.status = statusCode
}

func newRuntimeMiddleware(runtime *Runtime, handler transports.Handler) transports.Handler {
	return &runtimeMiddleware{
		runtime: runtime,
		next:    handler,
	}
}

type runtimeMiddleware struct {
	runtime *Runtime
	next    transports.Handler
}

func (middleware *runtimeMiddleware) Handle(w transports.ResponseWriter, r *transports.Request) {
	r.WithContext(middleware.runtime.SetIntoContext(r.Context()))
	middleware.next.Handle(w, r)
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	transportApplicationMiddlewareName = "application"
)

func newTransportApplicationMiddleware(status *Status) *transportApplicationMiddleware {
	return &transportApplicationMiddleware{
		appId:        "",
		appName:      "",
		appVersion:   versions.Version{},
		appStatus:    status,
		launchAT:     time.Time{},
		statsEnabled: false,
		counter:      sync.WaitGroup{},
	}
}

type transportApplicationMiddleware struct {
	appId          string
	appName        string
	appVersion     versions.Version
	appStatus      *Status
	launchAT       time.Time
	statsEnabled   bool
	latencyEnabled bool
	counter        sync.WaitGroup
}

func (middleware *transportApplicationMiddleware) Name() (name string) {
	name = transportApplicationMiddlewareName
	return
}

func (middleware *transportApplicationMiddleware) Build(options TransportMiddlewareOptions) (err error) {
	middleware.appId = options.AppId
	middleware.appName = options.AppName
	middleware.appVersion = options.AppVersion
	middleware.launchAT = time.Now()
	_, statsErr := options.Config.Get("statsEnable", &middleware.statsEnabled)
	if statsErr != nil {
		err = errors.Warning("fns: application middleware handler build failed").WithCause(statsErr).WithMeta("middleware", middleware.Name())
		return
	}
	_, latencyErr := options.Config.Get("latencyEnabled", &middleware.latencyEnabled)
	if latencyErr != nil {
		err = errors.Warning("fns: application middleware handler build failed").WithCause(latencyErr).WithMeta("middleware", middleware.Name())
		return
	}
	middleware.counter = sync.WaitGroup{}
	return
}

func (middleware *transportApplicationMiddleware) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r *transports.Request) {
		if middleware.appStatus.Closed() {
			w.Failed(ErrUnavailable)
			return
		}
		if middleware.appStatus.Starting() {
			w.Header().Set(httpResponseRetryAfter, "10")
			w.Failed(ErrTooEarly)
			return
		}
		middleware.counter.Add(1)
		// latency
		handleBeg := time.Time{}
		if middleware.latencyEnabled {
			handleBeg = time.Now()
		}
		// deviceId
		deviceId := strings.TrimSpace(r.Header().Get(httpDeviceIdHeader))
		if deviceId == "" {
			deviceId = strings.TrimSpace(bytex.ToString(r.Param("deviceId")))
			if deviceId == "" {
				if middleware.latencyEnabled {
					w.Header().Set(httpHandleLatencyHeader, time.Now().Sub(handleBeg).String())
				}
				w.SetStatus(555)
				p, _ := json.Marshal(ErrDeviceId)
				_, _ = w.Write(p)
				middleware.counter.Done()
				return
			}
		}
		// device ip
		deviceIp := r.Header().Get(httpDeviceIpHeader)
		if deviceIp == "" {
			if trueClientIp := r.Header().Get(httpTrueClientIp); trueClientIp != "" {
				deviceIp = trueClientIp
			} else if realIp := r.Header().Get(httpXRealIp); realIp != "" {
				deviceIp = realIp
			} else if forwarded := r.Header().Get(httpXForwardedForHeader); forwarded != "" {
				i := strings.Index(forwarded, ", ")
				if i == -1 {
					i = len(forwarded)
				}
				deviceIp = forwarded[:i]
			} else {
				remoteIp, _, remoteIpErr := net.SplitHostPort(bytex.ToString(r.RemoteAddr()))
				if remoteIpErr != nil {
					remoteIp = bytex.ToString(r.RemoteAddr())
				}
				deviceIp = remoteIp
			}
		}
		deviceIp = middleware.canonicalizeIp(deviceIp)
		r.Header().Set(httpDeviceIpHeader, deviceIp)
		// requestId
		requestId := strings.TrimSpace(r.Header().Get(httpRequestIdHeader))
		if requestId == "" {
			requestId = strings.TrimSpace(bytex.ToString(r.Param("requestId")))
			if requestId != "" {
				r.Header().Set(httpRequestIdHeader, requestId)
			}
		}
		next.Handle(w, r)
		if middleware.latencyEnabled {
			w.Header().Set(httpHandleLatencyHeader, time.Now().Sub(handleBeg).String())
		}
		middleware.counter.Done()
		return
	})
}

func (middleware *transportApplicationMiddleware) Close() {
	middleware.counter.Wait()
}

func (middleware *transportApplicationMiddleware) canonicalizeIp(ip string) string {
	isIPv6 := false
	for i := 0; !isIPv6 && i < len(ip); i++ {
		switch ip[i] {
		case '.':
			// IPv4
			return ip
		case ':':
			// IPv6
			isIPv6 = true
			break
		}
	}
	if !isIPv6 {
		return ip
	}
	ipv6 := net.ParseIP(ip)
	if ipv6 == nil {
		return ip
	}
	return ipv6.Mask(net.CIDRMask(64, 128)).String()
}
