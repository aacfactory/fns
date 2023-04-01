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
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/internal/commons/flags"
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
	Log        logs.Logger
	Config     configures.Config
	Discovery  EndpointDiscovery
	Shared     Shared
}

type TransportMiddleware interface {
	Name() (name string)
	Build(options TransportMiddlewareOptions) (err error)
	Handler(next http.Handler) http.Handler
	Close()
}

type TransportMiddlewaresOptions struct {
	AppId      string
	AppName    string
	AppVersion versions.Version
	AppRunning *flags.Flag
	Log        logs.Logger
	Config     configures.Config
	Discovery  EndpointDiscovery
	Shared     Shared
}

func newTransportMiddlewares(options TransportMiddlewaresOptions) *TransportMiddlewares {
	middlewares := make([]TransportMiddleware, 0, 1)
	middlewares = append(middlewares, newTransportApplicationMiddleware(options.AppRunning))
	return &TransportMiddlewares{
		appId:       options.AppId,
		appName:     options.AppName,
		appVersion:  options.AppVersion,
		log:         options.Log.With("transports", "middlewares"),
		config:      options.Config,
		discovery:   options.Discovery,
		shared:      options.Shared,
		middlewares: make([]TransportMiddleware, 0, 1),
	}
}

type TransportMiddlewares struct {
	appId       string
	appName     string
	appVersion  versions.Version
	log         logs.Logger
	config      configures.Config
	discovery   EndpointDiscovery
	shared      Shared
	middlewares []TransportMiddleware
}

func (middlewares *TransportMiddlewares) Append(middleware TransportMiddleware) (err error) {
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
		AppId:      middlewares.appId,
		AppName:    middlewares.appName,
		AppVersion: middlewares.appVersion,
		Log:        middlewares.log.With("middleware", name),
		Config:     config,
		Discovery:  middlewares.discovery,
		Shared:     middlewares.shared,
	})
	if buildErr != nil {
		err = errors.Warning("append middleware failed").WithCause(buildErr).WithMeta("middleware", name)
		return
	}
	middlewares.middlewares = append(middlewares.middlewares, middleware)
	return
}

func (middlewares *TransportMiddlewares) Wrap(handler http.Handler) http.Handler {
	for i := len(middlewares.middlewares) - 1; i > -1; i-- {
		handler = middlewares.middlewares[i].Handler(handler)
	}
	return newBufferResponseMiddleware(handler)
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

func newBufferResponseMiddleware(handler http.Handler) http.Handler {
	return &bufferResponseMiddleware{
		pool: sync.Pool{
			New: func() any {
				return &responseWriter{
					status: http.StatusOK,
					header: nil,
					buf:    nil,
				}
			},
		},
		next: handler,
	}
}

type bufferResponseMiddleware struct {
	pool sync.Pool
	next http.Handler
}

func (middleware *bufferResponseMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var bw *responseWriter
	got := middleware.pool.Get()
	if got == nil {
		bw = &responseWriter{}
	} else {
		ok := false
		bw, ok = got.(*responseWriter)
		if !ok {
			w.Header().Set(httpContentType, httpContentTypeJson)
			w.WriteHeader(555)
			p, _ := json.Marshal(errors.NotFound("fns: get buffer response writer from pool failed").
				WithCause(errors.Warning("type was not matched")).
				WithMeta("fns", "handlers"))
			_, _ = w.Write(p)
			return
		}
	}
	bw.header = http.Header{}
	bw.buf = bytebufferpool.Get()
	middleware.next.ServeHTTP(bw, r)
	if bw.header != nil && len(bw.header) > 0 {
		for k, vv := range bw.header {
			if vv == nil || len(vv) == 0 {
				continue
			}
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
	}
	w.Header().Set(httpContentType, httpContentTypeJson)
	w.WriteHeader(bw.status)
	bodyLen := bw.buf.Len()
	if bodyLen == 0 {
		bytebufferpool.Put(bw.buf)
		middleware.pool.Put(bw)
		return
	}
	body := bw.buf.Bytes()
	n := 0
	for n < bodyLen {
		nn, writeErr := w.Write(body[n:])
		if writeErr != nil {
			break
		}
		n += nn
	}
	bytebufferpool.Put(bw.buf)
	middleware.pool.Put(bw)
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	transportApplicationMiddlewareName = "application"
)

func newTransportApplicationMiddleware(appRunning *flags.Flag) *transportApplicationMiddleware {
	return &transportApplicationMiddleware{
		appId:        "",
		appName:      "",
		appVersion:   versions.Version{},
		appRunning:   appRunning,
		launchAT:     time.Time{},
		statsEnabled: false,
		counter:      sync.WaitGroup{},
	}
}

type transportApplicationMiddleware struct {
	appId          string
	appName        string
	appVersion     versions.Version
	appRunning     *flags.Flag
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

func (middleware *transportApplicationMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if middleware.appRunning.IsOff() {
			w.Header().Set(httpConnectionHeader, httpCloseHeader)
			w.WriteHeader(http.StatusServiceUnavailable)
			p, _ := json.Marshal(ErrUnavailable)
			_, _ = w.Write(p)
			return
		}
		if middleware.appRunning.IsHalfOn() {
			w.Header().Set(httpResponseRetryAfter, "10")
			w.WriteHeader(http.StatusTooEarly)
			p, _ := json.Marshal(ErrTooEarly)
			_, _ = w.Write(p)
			return
		}
		middleware.counter.Add(1)
		// latency
		handleBeg := time.Time{}
		if middleware.latencyEnabled {
			handleBeg = time.Now()
		}
		// deviceId
		deviceId := strings.TrimSpace(r.Header.Get(httpDeviceIdHeader))
		if deviceId == "" {
			deviceId = strings.TrimSpace(r.URL.Query().Get("deviceId"))
			if deviceId == "" {
				if middleware.latencyEnabled {
					w.Header().Set(httpHandleLatencyHeader, time.Now().Sub(handleBeg).String())
				}
				w.WriteHeader(555)
				p, _ := json.Marshal(ErrDeviceId)
				_, _ = w.Write(p)
				middleware.counter.Done()
				return
			}
		}
		// device ip
		deviceIp := r.Header.Get(httpDeviceIpHeader)
		if deviceIp == "" {
			if trueClientIp := r.Header.Get(httpTrueClientIp); trueClientIp != "" {
				deviceIp = trueClientIp
			} else if realIp := r.Header.Get(httpXRealIp); realIp != "" {
				deviceIp = realIp
			} else if forwarded := r.Header.Get(httpXForwardedForHeader); forwarded != "" {
				i := strings.Index(forwarded, ", ")
				if i == -1 {
					i = len(forwarded)
				}
				deviceIp = forwarded[:i]
			} else {
				remoteIp, _, remoteIpErr := net.SplitHostPort(r.RemoteAddr)
				if remoteIpErr != nil {
					remoteIp = r.RemoteAddr
				}
				deviceIp = remoteIp
			}
			r.Header.Set(httpDeviceIpHeader, deviceIp)
		}
		deviceIp = middleware.canonicalizeIp(deviceIp)
		r.Header.Set(httpDeviceIpHeader, deviceIp)
		// requestId
		requestId := strings.TrimSpace(r.Header.Get(httpRequestIdHeader))
		if requestId == "" {
			requestId = strings.TrimSpace(r.URL.Query().Get("requestId"))
			if requestId != "" {
				r.Header.Set(httpRequestIdHeader, requestId)
			}
		}
		next.ServeHTTP(w, r)
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
