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
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	httpContentType            = "Content-Type"
	httpContentTypeJson        = "application/json"
	httpConnectionHeader       = "Connection"
	httpUpgradeHeader          = "Upgrade"
	httpCloseHeader            = "close"
	httpCacheControlHeader     = "Cache-Control"
	httpPragmaHeader           = "Pragma"
	httpETagHeader             = "ETag"
	httpCacheControlIfNonMatch = "If-None-Match"
	httpClearSiteData          = "Clear-Site-Data"
	httpTrueClientIp           = "True-Client-Ip"
	httpXRealIp                = "X-Real-IP"
	httpXForwardedForHeader    = "X-Forwarded-For"
	httpRequestIdHeader        = "X-Fns-Request-Id"
	httpRequestSignatureHeader = "X-Fns-Request-Signature"
	httpRequestInternalHeader  = "X-Fns-Request-Internal"
	httpRequestTimeoutHeader   = "X-Fns-Request-Timeout"
	httpRequestVersionsHeader  = "X-Fns-Request-Version"
	httpHandleLatencyHeader    = "X-Fns-Handle-Latency"
	httpDeviceIdHeader         = "X-Fns-Device-Id"
	httpDeviceIpHeader         = "X-Fns-Device-Ip"
	httpDevModeHeader          = "X-Fns-Dev-Mode"
	httpResponseRetryAfter     = "Retry-After"
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
	Close() (err error)
}

type transportMiddlewaresOptions struct {
	Runtime *Runtime
	Cors    *transports.CorsConfig
	Config  configures.Config
}

func newTransportMiddlewares(options transportMiddlewaresOptions) *transportMiddlewares {
	middlewares := make([]TransportMiddleware, 0, 1)
	middlewares = append(middlewares, newTransportApplicationMiddleware(options.Runtime))
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

func (middlewares *transportMiddlewares) Close() (err error) {
	errs := errors.MakeErrors()
	for _, middleware := range middlewares.middlewares {
		err = middleware.Close()
		if err != nil {
			errs.Append(err)
		}
	}
	err = errs.Error()
	return
}

func (middlewares *transportMiddlewares) Handler(handlers *transportHandlers) transports.Handler {
	var handler transports.Handler = handlers
	for i := len(middlewares.middlewares) - 1; i > -1; i-- {
		handler = middlewares.middlewares[i].Handler(handler)
	}
	if middlewares.cors == nil {
		middlewares.cors = &transports.CorsConfig{
			AllowedOrigins:      []string{"*"},
			AllowedHeaders:      []string{"*"},
			ExposedHeaders:      make([]string, 0, 1),
			AllowCredentials:    false,
			MaxAge:              86400,
			AllowPrivateNetwork: false,
		}
	}
	middlewares.cors.TryFillAllowedHeaders([]string{
		httpConnectionHeader, httpUpgradeHeader,
		httpXForwardedForHeader, httpTrueClientIp, httpXRealIp,
		httpDeviceIpHeader, httpDeviceIdHeader,
		httpRequestIdHeader,
		httpRequestSignatureHeader, httpRequestTimeoutHeader, httpRequestVersionsHeader,
		httpETagHeader, httpCacheControlIfNonMatch, httpPragmaHeader, httpClearSiteData, httpResponseRetryAfter,
	})
	middlewares.cors.TryFillExposedHeaders([]string{
		httpRequestIdHeader, httpRequestSignatureHeader, httpHandleLatencyHeader,
		httpCacheControlHeader, httpETagHeader, httpClearSiteData, httpResponseRetryAfter,
	})
	return middlewares.cors.Handler(handler)
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	transportApplicationMiddlewareName = "application"
)

func newTransportApplicationMiddleware(runtime *Runtime) *transportApplicationMiddleware {
	return &transportApplicationMiddleware{
		runtime:      runtime,
		launchAT:     time.Time{},
		statsEnabled: false,
		counter:      sync.WaitGroup{},
	}
}

type transportApplicationMiddleware struct {
	runtime        *Runtime
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
		if middleware.runtime.status.Closed() {
			w.Failed(ErrUnavailable)
			return
		}
		if middleware.runtime.status.Starting() {
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
		next.Handle(w, r.WithContext(middleware.runtime.SetIntoContext(r.Context())))
		if !w.Hijacked() && middleware.latencyEnabled {
			w.Header().Set(httpHandleLatencyHeader, time.Now().Sub(handleBeg).String())
		}
		middleware.counter.Done()
		return
	})
}

func (middleware *transportApplicationMiddleware) Close() (err error) {
	middleware.counter.Wait()
	return
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