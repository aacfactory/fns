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
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/internal/commons/flags"
	"github.com/aacfactory/fns/service/internal/ratelimit"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/systems/cpu"
	"github.com/aacfactory/systems/memory"
	"github.com/rs/cors"
	"github.com/valyala/bytebufferpool"
	"golang.org/x/sync/singleflight"
	"net"
	"net/http"
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
	httpAppIdHeader            = "X-Fns-Id"
	httpAppNameHeader          = "X-Fns-Name"
	httpAppVersionHeader       = "X-Fns-Version"
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

// +-------------------------------------------------------------------------------------------------------------------+

type TransportHandlerOptions struct {
	AppId       string
	AppName     string
	AppVersion  versions.Version
	Log         logs.Logger
	Config      configures.Config
	Discovery   EndpointDiscovery
	ClusterMode bool
}

type TransportHandler interface {
	Name() (name string)
	Build(options TransportHandlerOptions) (err error)
	Accept(r *http.Request) (ok bool)
	http.Handler
	Close()
}

type TransportHandlersOptions struct {
	AppId       string
	AppName     string
	AppVersion  versions.Version
	AppRunning  *flags.Flag
	Log         logs.Logger
	Config      configures.Config
	Discovery   EndpointDiscovery
	ClusterMode bool
}

func newTransportHandlers(options TransportHandlersOptions) *TransportHandlers {
	handlers := make([]TransportHandler, 0, 1)
	handlers = append(handlers, newTransportApplicationHandler(options.AppRunning))
	return &TransportHandlers{
		appId:       options.AppId,
		appName:     options.AppName,
		appVersion:  options.AppVersion,
		log:         options.Log.With("transports", "handlers"),
		config:      options.Config,
		discovery:   options.Discovery,
		clusterMode: options.ClusterMode,
		handlers:    handlers,
	}
}

type TransportHandlers struct {
	appId       string
	appName     string
	appVersion  versions.Version
	log         logs.Logger
	config      configures.Config
	discovery   EndpointDiscovery
	clusterMode bool
	handlers    []TransportHandler
}

func (handlers *TransportHandlers) Append(handler TransportHandler) (err error) {
	if handler == nil {
		return
	}
	name := strings.TrimSpace(handler.Name())
	if name == "" {
		err = errors.Warning("append handler failed").WithCause(errors.Warning("one of handler has no name"))
		return
	}
	pos := sort.Search(len(handlers.handlers), func(i int) bool {
		return handlers.handlers[i].Name() == name
	})
	if pos < len(handlers.handlers) {
		err = errors.Warning("append handler failed").WithCause(errors.Warning("this handle was appended")).WithMeta("handler", name)
		return
	}
	config, exist := handlers.config.Node(name)
	if !exist {
		config, _ = configures.NewJsonConfig([]byte{'{', '}'})
	}
	buildErr := handler.Build(TransportHandlerOptions{
		AppId:       handlers.appId,
		AppName:     handlers.appName,
		AppVersion:  handlers.appVersion,
		Log:         handlers.log.With("handler", name),
		Config:      config,
		Discovery:   handlers.discovery,
		ClusterMode: handlers.clusterMode,
	})
	if buildErr != nil {
		err = errors.Warning("append handler failed").WithCause(buildErr).WithMeta("handler", name)
		return
	}
	handlers.handlers = append(handlers.handlers, handler)
	return
}

func (handlers *TransportHandlers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handled := false
	for _, handler := range handlers.handlers {
		if accepted := handler.Accept(r); accepted {
			handler.ServeHTTP(w, r)
			handled = true
			break
		}
	}
	if !handled {
		w.Header().Set(httpContentType, httpContentTypeJson)
		w.WriteHeader(http.StatusNotFound)
		p, _ := json.Marshal(errors.NotFound("fns: no handlers accept request").WithMeta("fns", "handlers"))
		_, _ = w.Write(p)
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

type TransportMiddlewareOptions struct {
	AppId       string
	AppName     string
	AppVersion  versions.Version
	Log         logs.Logger
	Config      configures.Config
	Discovery   EndpointDiscovery
	ClusterMode bool
}

type TransportMiddleware interface {
	Name() (name string)
	Build(options TransportMiddlewareOptions) (err error)
	Handler(next http.Handler) http.Handler
	Close()
}

type TransportMiddlewaresOptions struct {
	AppId       string
	AppName     string
	AppVersion  versions.Version
	AppRunning  *flags.Flag
	Log         logs.Logger
	Config      configures.Config
	Discovery   EndpointDiscovery
	ClusterMode bool
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
		clusterMode: options.ClusterMode,
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
	clusterMode bool
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
		AppId:       middlewares.appId,
		AppName:     middlewares.appName,
		AppVersion:  middlewares.appVersion,
		Log:         middlewares.log.With("middleware", name),
		Config:      config,
		Discovery:   middlewares.discovery,
		ClusterMode: middlewares.clusterMode,
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

func createTransportOptions(config *HttpConfig, log logs.Logger, handler http.Handler) (opt transports.Options, err error) {
	log = log.With("fns", "transport")
	opt = transports.Options{
		Port:      80,
		ServerTLS: nil,
		ClientTLS: nil,
		Handler:   handler,
		Log:       log,
		Options:   nil,
	}
	if config == nil {
		return
	}
	var serverTLS *tls.Config
	var clientTLS *tls.Config
	if config.TLS != nil {
		var tlsErr error
		serverTLS, clientTLS, tlsErr = config.TLS.Config()
		if tlsErr != nil {
			err = errors.Warning("create transport options failed").WithCause(tlsErr)
			return
		}
	}
	opt.ServerTLS = serverTLS
	opt.ClientTLS = clientTLS
	port := config.Port
	if port == 0 {
		if serverTLS == nil {
			port = 80
		} else {
			port = 443
		}
	}
	if port < 1 || port > 65535 {
		err = errors.Warning("create transport options failed").WithCause(fmt.Errorf("port is invalid, port must great than 1024 or less than 65536"))
		return
	}
	opt.Port = port
	if config.Options == nil {
		config.Options = []byte("{}")
	}
	opt.Options, err = configures.NewJsonConfig(config.Options)
	if err != nil {
		err = errors.Warning("create transport options failed").WithCause(fmt.Errorf("options is invalid")).WithCause(err)
		return
	}
	return
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
			p, _ := json.Marshal(errors.Unavailable("fns: application is closed").WithMeta("middleware", middleware.Name()))
			_, _ = w.Write(p)
			return
		}
		if middleware.appRunning.IsHalfOn() {
			w.Header().Set(httpResponseRetryAfter, "10")
			w.WriteHeader(http.StatusTooEarly)
			p, _ := json.Marshal(errors.New(http.StatusTooEarly, "***TOO EARLY***", "fns: application is not ready, try later again").WithMeta("middleware", middleware.Name()))
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
				p, _ := json.Marshal(errors.Warning("fns: device id was required"))
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

// +-------------------------------------------------------------------------------------------------------------------+

const (
	transportApplicationHandlerName = "application"
)

type applicationStats struct {
	Id      string         `json:"id"`
	Name    string         `json:"name"`
	Running int64          `json:"running"`
	Mem     *memory.Memory `json:"mem"`
	CPU     *cpuOccupancy  `json:"cpu"`
}

type cpuOccupancy struct {
	Max   cpu.Core `json:"max"`
	Min   cpu.Core `json:"min"`
	Avg   float64  `json:"avg"`
	Cores cpu.CPU  `json:"cores"`
}

func newTransportApplicationHandler(running *flags.Flag) *transportApplicationHandler {
	return &transportApplicationHandler{
		appId:        "",
		appName:      "",
		appVersion:   versions.Version{},
		appRunning:   running,
		launchAT:     time.Time{},
		statsEnabled: false,
		group:        singleflight.Group{},
	}
}

type transportApplicationHandler struct {
	appId        string
	appName      string
	appVersion   versions.Version
	appRunning   *flags.Flag
	launchAT     time.Time
	statsEnabled bool
	group        singleflight.Group
}

func (handler *transportApplicationHandler) Name() (name string) {
	name = transportApplicationHandlerName
	return
}

func (handler *transportApplicationHandler) Build(options TransportHandlerOptions) (err error) {
	handler.appId = options.AppId
	handler.appName = options.AppName
	handler.appVersion = options.AppVersion
	handler.launchAT = time.Now()
	_, statsErr := options.Config.Get("statsEnable", &handler.statsEnabled)
	if statsErr != nil {
		err = errors.Warning("fns: application handler build failed").WithCause(statsErr).WithMeta("handler", handler.Name())
		return
	}
	return
}

func (handler *transportApplicationHandler) Accept(r *http.Request) (ok bool) {
	ok = r.Method == http.MethodGet && r.URL.Path == "/application/health"
	if ok {
		return
	}
	ok = r.Method == http.MethodGet && r.URL.Path == "/application/stats"
	if ok {
		return
	}
	return
}

func (handler *transportApplicationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/application/health" {
		body := fmt.Sprintf(
			"{\"name\":\"%s\", \"id\":\"%s\", \"version\":\"%s\", \"launch\":\"%s\", \"now\":\"%s\", \"deviceIp\":\"%s\"}",
			handler.appName, handler.appId, handler.appVersion.String(), handler.launchAT.Format(time.RFC3339), time.Now().Format(time.RFC3339), r.Header.Get(httpDeviceIpHeader),
		)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytex.FromString(body))
		return
	}
	if handler.statsEnabled && r.Method == http.MethodGet && r.URL.Path == "/application/stats" {
		v, _, _ := handler.group.Do(handler.Name(), func() (v interface{}, err error) {
			stat := &applicationStats{
				Id:      handler.appId,
				Name:    handler.appName,
				Running: handler.appRunning.Value(),
				Mem:     nil,
				CPU:     nil,
			}
			mem, memErr := memory.Stats()
			if memErr == nil {
				stat.Mem = mem
			}
			cpus, cpuErr := cpu.Occupancy()
			if cpuErr == nil {
				stat.CPU = &cpuOccupancy{
					Max:   cpus.Max(),
					Min:   cpus.Min(),
					Avg:   cpus.AVG(),
					Cores: cpus,
				}
			}
			v, _ = json.Marshal(stat)
			return
		})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(v.([]byte))
		return
	}
	return
}

func (handler *transportApplicationHandler) Close() {
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func newCorsHandler(config *CorsConfig) (h *cors.Cors) {
	if config == nil {
		config = &CorsConfig{
			AllowedOrigins:   []string{"*"},
			AllowedHeaders:   []string{"*"},
			ExposedHeaders:   nil,
			AllowCredentials: false,
			MaxAge:           0,
		}
	}
	if config.AllowedOrigins == nil || len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"*"}
	}
	if config.AllowedHeaders == nil || len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = make([]string, 0, 1)
		config.AllowedHeaders = append(config.AllowedHeaders, "*")
	}
	if config.AllowedHeaders[0] != "*" {
		if sort.SearchStrings(config.AllowedHeaders, httpConnectionHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpConnectionHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpUpgradeHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpUpgradeHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpXForwardedForHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpXForwardedForHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpDeviceIpHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpDeviceIpHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpDeviceIdHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpDeviceIdHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpRequestIdHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpRequestIdHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpRequestSignatureHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpRequestSignatureHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpRequestTimeoutHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpRequestTimeoutHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpRequestVersionsHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpRequestVersionsHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpDevModeHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpDevModeHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpCacheControlIfNonMatch) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpCacheControlIfNonMatch)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpPragmaHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpPragmaHeader)
		}
	}
	if config.ExposedHeaders == nil {
		config.ExposedHeaders = make([]string, 0, 1)
	}
	config.ExposedHeaders = append(
		config.ExposedHeaders,
		httpAppIdHeader, httpAppNameHeader, httpAppVersionHeader,
		httpRequestIdHeader, httpRequestSignatureHeader, httpHandleLatencyHeader,
		httpCacheControlHeader, httpETagHeader, httpClearSiteData, httpResponseRetryAfter,
	)
	h = cors.New(cors.Options{
		AllowedOrigins:         config.AllowedOrigins,
		AllowOriginFunc:        nil,
		AllowOriginRequestFunc: nil,
		AllowedMethods:         []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:         config.AllowedHeaders,
		ExposedHeaders:         config.ExposedHeaders,
		MaxAge:                 config.MaxAge,
		AllowCredentials:       config.AllowCredentials,
		AllowPrivateNetwork:    config.AllowPrivateNetwork,
		OptionsPassthrough:     false,
		OptionsSuccessStatus:   204,
		Debug:                  false,
	})
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	limiterMiddlewareName = "limiter"
)

type limiterMiddlewareConfig struct {
	Max        int64  `json:"max"`
	RetryAfter int    `json:"retryAfter"`
	TimeWindow string `json:"timeWindow"`
}

func LimiterMiddleware() TransportMiddleware {
	return &limiterMiddleware{}
}

type limiterMiddleware struct {
	log        logs.Logger
	tickets    *ratelimit.Limiter
	retryAfter string
}

func (middleware *limiterMiddleware) Name() (name string) {
	name = limiterMiddlewareName
	return
}

func (middleware *limiterMiddleware) Build(options TransportMiddlewareOptions) (err error) {
	middleware.log = options.Log
	config := limiterMiddlewareConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: limiter middleware build failed").WithCause(configErr)
		return
	}
	max := config.Max
	if max <= 0 {
		max = 5
	}
	timeWindow := 10 * time.Second
	if config.TimeWindow != "" {
		timeWindow, err = time.ParseDuration(strings.TrimSpace(config.TimeWindow))
		if err != nil {
			err = errors.Warning("fns: limiter middleware build failed").WithCause(errors.Warning("timeWindow must be time.Duration format")).WithCause(configErr)
			return
		}
	}
	if config.RetryAfter > 0 {
		middleware.retryAfter = fmt.Sprintf("%d", config.RetryAfter)
	} else {
		middleware.retryAfter = "10"
	}
	middleware.tickets = ratelimit.New(max, timeWindow)
	return
}

func (middleware *limiterMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deviceId := r.Header.Get(httpDeviceIdHeader)
		ok, takeErr := middleware.tickets.Take(deviceId)
		if takeErr != nil {
			p, _ := json.Marshal(errors.Warning("fns: request limiter take ticket failed").WithCause(takeErr))
			w.WriteHeader(555)
			_, _ = w.Write(p)
			return
		}
		if !ok {
			p, _ := json.Marshal(errors.Warning("fns: request limiter take ticket failed").WithCause(takeErr))
			w.Header().Set(httpResponseRetryAfter, middleware.retryAfter)
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write(p)
			return
		}
		next.ServeHTTP(w, r)
		repayErr := middleware.tickets.Repay(deviceId)
		if repayErr != nil && middleware.log.ErrorEnabled() {
			middleware.log.Error().Cause(
				errors.Warning("fns: request limiter repay ticket failed").WithCause(repayErr),
			).With("middleware", middleware.Name()).Message("fns: request limiter repay ticket failed")
		}
		return
	})
}

func (middleware *limiterMiddleware) Close() {
	middleware.tickets.Close()
	return
}

// +-------------------------------------------------------------------------------------------------------------------+
