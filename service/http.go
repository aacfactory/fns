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
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/internal/commons/flags"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/systems/cpu"
	"github.com/aacfactory/systems/memory"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"golang.org/x/sync/singleflight"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type HttpHandlerOptions struct {
	AppId      string
	AppName    string
	AppVersion versions.Version
	Log        logs.Logger
	Config     configures.Config
	Discovery  EndpointDiscovery
}

type HttpHandler interface {
	http.Handler
	Name() (name string)
	Build(options *HttpHandlerOptions) (err error)
	Accept(r *http.Request) (ok bool)
	Close()
}

type HttpHandlerWithServices interface {
	HttpHandler
	Services() (services []Service)
}

type HttpInterceptorOptions struct {
	AppId      string
	AppName    string
	AppVersion versions.Version
	Log        logs.Logger
	Config     configures.Config
	Discovery  EndpointDiscovery
}

type HttpInterceptor interface {
	Name() (name string)
	Build(options *HttpInterceptorOptions) (err error)
	Handler(next http.Handler) http.Handler
	Close()
}

type HttpInterceptorWithServices interface {
	HttpInterceptor
	Services() (services []Service)
}

type HttpLatencyInterceptorConfig struct {
	Enable bool `json:"enable"`
}

type HttpLatencyInterceptor struct {
	enable bool
}

func (interceptor *HttpLatencyInterceptor) Name() (name string) {
	name = "latency"
	return
}

func (interceptor *HttpLatencyInterceptor) Build(options *HttpInterceptorOptions) (err error) {
	config := HttpLatencyInterceptorConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: build latency interceptor failed").WithCause(configErr)
		return
	}
	interceptor.enable = config.Enable
	return
}

func (interceptor *HttpLatencyInterceptor) Handler(next http.Handler) http.Handler {
	if interceptor.enable {
		return http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
			beg := time.Now()
			next.ServeHTTP(writer, r)
			latency := time.Now().Sub(beg).String()
			writer.Header().Set(httpHandleLatencyHeader, latency)
		})
	}
	return next
}

func (interceptor *HttpLatencyInterceptor) Close() {
	return
}

type HandlersOptions struct {
	AppId      string
	AppName    string
	AppVersion versions.Version
	Log        logs.Logger
	Config     configures.Config
	Discovery  EndpointDiscovery
	Running    *flags.Flag
}

func NewHttpHandlers(options HandlersOptions) (handlers *HttpHandlers, err errors.CodeError) {
	handlers = &HttpHandlers{
		appId:         options.AppId,
		appName:       options.AppName,
		appVersion:    options.AppVersion,
		appLaunchAT:   time.Now(),
		log:           options.Log,
		config:        options.Config,
		discovery:     options.Discovery,
		handlers:      make([]HttpHandler, 0, 1),
		interceptors:  make([]HttpInterceptor, 0, 1),
		running:       options.Running,
		counter:       sync.WaitGroup{},
		group:         singleflight.Group{},
		requestCounts: int64(0),
	}
	return
}

type HttpHandlers struct {
	appId         string
	appName       string
	appVersion    versions.Version
	appLaunchAT   time.Time
	log           logs.Logger
	config        configures.Config
	discovery     EndpointDiscovery
	running       *flags.Flag
	handlers      []HttpHandler
	interceptors  []HttpInterceptor
	counter       sync.WaitGroup
	group         singleflight.Group
	requestCounts int64
}

func (handlers *HttpHandlers) AppendInterceptor(h HttpInterceptor) (err errors.CodeError) {
	if h == nil {
		return
	}
	name := h.Name()
	handleConfig, hasHandleConfig := handlers.config.Node(name)
	if !hasHandleConfig {
		handleConfig, _ = configures.NewJsonConfig([]byte{'{', '}'})
	}
	options := &HttpInterceptorOptions{
		AppId:      handlers.appId,
		AppName:    handlers.appName,
		AppVersion: handlers.appVersion,
		Log:        handlers.log.With("handler", name),
		Config:     handleConfig,
		Discovery:  handlers.discovery,
	}
	buildErr := h.Build(options)
	if buildErr != nil {
		err = errors.Warning("fns: build interceptor handler failed").WithMeta("name", name).WithCause(errors.Map(buildErr))
		return
	}
	handlers.interceptors = append(handlers.interceptors, h)
	return
}

func (handlers *HttpHandlers) Append(h HttpHandler) (err errors.CodeError) {
	if h == nil {
		return
	}
	name := h.Name()
	handleConfig, hasHandleConfig := handlers.config.Node(name)
	if !hasHandleConfig {
		handleConfig, _ = configures.NewJsonConfig([]byte{'{', '}'})
	}
	options := &HttpHandlerOptions{
		AppId:      handlers.appId,
		AppName:    handlers.appName,
		AppVersion: handlers.appVersion,
		Log:        handlers.log.With("handler", name),
		Config:     handleConfig,
		Discovery:  handlers.discovery,
	}
	buildErr := h.Build(options)
	if buildErr != nil {
		err = errors.Warning("fns: build handler failed").WithMeta("name", name).WithCause(errors.Map(buildErr))
		return
	}
	handlers.handlers = append(handlers.handlers, h)
	return
}

func (handlers *HttpHandlers) Build() (h http.Handler) {
	h = http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		handlers.Handle(writer, r)
	})
	if handlers.interceptors == nil || len(handlers.interceptors) == 0 {
		return h
	}
	for i := len(handlers.interceptors) - 1; i > -1; i-- {
		h = handlers.interceptors[i].Handler(h)
	}
	return h
}

func (handlers *HttpHandlers) Handle(writer http.ResponseWriter, request *http.Request) {
	handlers.counter.Add(1)
	atomic.AddInt64(&handlers.requestCounts, 1)
	writer.Header().Set(httpAppIdHeader, handlers.appId)
	writer.Header().Set(httpAppNameHeader, handlers.appName)
	writer.Header().Set(httpAppVersionHeader, handlers.appVersion.String())
	if handlers.running.IsOff() {
		writer.Header().Set(httpConnectionHeader, httpCloseHeader)
		writer.Header().Set(httpContentType, httpContentTypeJson)
		writer.WriteHeader(http.StatusRequestTimeout)
		_, _ = writer.Write(json.UnsafeMarshal(errors.Warning("fns: service is not serving").WithMeta("fns", "handlers")))
		atomic.AddInt64(&handlers.requestCounts, -1)
		handlers.counter.Done()
		return
	}
	if handlers.running.IsHalfOn() {
		writer.Header().Set(httpResponseRetryAfter, "30")
		writer.Header().Set(httpContentType, httpContentTypeJson)
		writer.WriteHeader(http.StatusTooEarly)
		_, _ = writer.Write(json.UnsafeMarshal(errors.Warning("fns: service is not serving").WithMeta("fns", "handlers")))
		atomic.AddInt64(&handlers.requestCounts, -1)
		handlers.counter.Done()
		return
	}
	if handlers.handleApplication(writer, request) {
		atomic.AddInt64(&handlers.requestCounts, -1)
		handlers.counter.Done()
		return
	}
	handled := false
	for _, handler := range handlers.handlers {
		if handler.Accept(request) {
			handler.ServeHTTP(writer, request)
			handled = true
			break
		}
	}
	if !handled {
		writer.Header().Set(httpContentType, httpContentTypeJson)
		writer.WriteHeader(http.StatusNotFound)
		_, _ = writer.Write(json.UnsafeMarshal(errors.Warning("fns: no handler accept request").WithMeta("fns", "handlers")))
		atomic.AddInt64(&handlers.requestCounts, -1)
		handlers.counter.Done()
		return
	}
	atomic.AddInt64(&handlers.requestCounts, -1)
	handlers.counter.Done()
	return
}

func (handlers *HttpHandlers) handleApplication(writer http.ResponseWriter, request *http.Request) (ok bool) {
	if request.Method == http.MethodGet && request.URL.Path == "/application/health" {
		body := fmt.Sprintf(
			"{\"name\":\"%s\", \"id\":\"%s\", \"version\":\"%s\", \"launch\":\"%s\", \"now\":\"%s\"}",
			handlers.appName, handlers.appId, handlers.appVersion, handlers.appLaunchAT.Format(time.RFC3339), time.Now().Format(time.RFC3339),
		)
		writer.Header().Set(httpContentType, httpContentTypeJson)
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(bytex.FromString(body))
		ok = true
		return
	}
	if request.Method == http.MethodGet && request.URL.Path == "/application/handlers" {
		const (
			handlersKey = "handlers"
		)
		v, _, _ := handlers.group.Do(handlersKey, func() (v interface{}, err error) {
			v, _ = json.Marshal(handlers.HandlerNames())
			return
		})
		writer.Header().Set(httpContentType, httpContentTypeJson)
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(v.([]byte))
		ok = true
		return
	}
	if request.Method == http.MethodGet && request.URL.Path == "/application/stats" {
		const (
			statsKey = "stats"
		)
		v, _, _ := handlers.group.Do(statsKey, func() (v interface{}, err error) {
			stat := &applicationStats{
				Id:       handlers.appId,
				Name:     handlers.appName,
				Running:  handlers.running.IsOn(),
				Requests: uint64(atomic.LoadInt64(&handlers.requestCounts)),
				Mem:      nil,
				CPU:      nil,
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
		writer.Header().Set(httpContentType, httpContentTypeJson)
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(v.([]byte))
		ok = true
		return
	}
	return
}

func (handlers *HttpHandlers) HandlerNames() (names []string) {
	names = make([]string, 0, 1)
	for _, handler := range handlers.handlers {
		names = append(names, handler.Name())
	}
	return
}

func (handlers *HttpHandlers) Close() {
	handlers.counter.Wait()
	for _, handler := range handlers.handlers {
		handlers.counter.Add(1)
		go func(handler HttpHandler, waiter *sync.WaitGroup) {
			handler.Close()
			waiter.Done()
		}(handler, &handlers.counter)
	}
	for _, interceptor := range handlers.interceptors {
		handlers.counter.Add(1)
		go func(interceptor HttpInterceptor, waiter *sync.WaitGroup) {
			interceptor.Close()
			waiter.Done()
		}(interceptor, &handlers.counter)
	}
	handlers.counter.Wait()
}

type applicationStats struct {
	Id       string         `json:"id"`
	Name     string         `json:"name"`
	Running  bool           `json:"running"`
	Requests uint64         `json:"requests"`
	Mem      *memory.Memory `json:"mem"`
	CPU      *cpuOccupancy  `json:"cpu"`
}

type cpuOccupancy struct {
	Max   cpu.Core `json:"max"`
	Min   cpu.Core `json:"min"`
	Avg   float64  `json:"avg"`
	Cores cpu.CPU  `json:"cores"`
}

func NewHttpOptions(config *HttpConfig, log logs.Logger, handler http.Handler) (opt HttpOptions, err error) {
	log = log.With("fns", "http")
	opt = HttpOptions{
		Port:      0,
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
			err = errors.Warning("new http options failed").WithCause(tlsErr)
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
		err = errors.Warning("new http options failed").WithCause(fmt.Errorf("port is invalid, port must great than 1024 or less than 65536"))
		return
	}
	opt.Port = port
	if config.Options == nil {
		config.Options = []byte("{}")
	}
	opt.Options, err = configures.NewJsonConfig(config.Options)
	if err != nil {
		err = errors.Warning("new http options failed").WithCause(fmt.Errorf("options is invalid")).WithCause(err)
		return
	}
	return
}

type HttpOptions struct {
	Port      int
	ServerTLS *tls.Config
	ClientTLS *tls.Config
	Handler   http.Handler
	Log       logs.Logger
	Options   configures.Config
}

type HttpClient interface {
	Get(ctx context.Context, path string, header http.Header) (status int, respHeader http.Header, respBody []byte, err error)
	Post(ctx context.Context, path string, header http.Header, body []byte) (status int, respHeader http.Header, respBody []byte, err error)
	Close()
}

type HttpClientDialer interface {
	Dial(address string) (client HttpClient, err error)
}

type Http interface {
	Name() (name string)
	Build(options HttpOptions) (err error)
	HttpClientDialer
	ListenAndServe() (err error)
	Close() (err error)
}

type FastHttpOptions struct {
	ReadBufferSize        string                `json:"readBufferSize"`
	ReadTimeout           string                `json:"readTimeout"`
	WriteBufferSize       string                `json:"writeBufferSize"`
	WriteTimeout          string                `json:"writeTimeout"`
	MaxIdleWorkerDuration string                `json:"maxIdleWorkerDuration"`
	TCPKeepalive          bool                  `json:"tcpKeepalive"`
	TCPKeepalivePeriod    string                `json:"tcpKeepalivePeriod"`
	MaxRequestBodySize    string                `json:"maxRequestBodySize"`
	ReduceMemoryUsage     bool                  `json:"reduceMemoryUsage"`
	MaxRequestsPerConn    int                   `json:"maxRequestsPerConn"`
	KeepHijackedConns     bool                  `json:"keepHijackedConns"`
	StreamRequestBody     bool                  `json:"streamRequestBody"`
	Client                FastHttpClientOptions `json:"client"`
}

type FastHttpClientOptions struct {
	DialDualStack             bool   `json:"dialDualStack"`
	MaxConnsPerHost           int    `json:"maxConnsPerHost"`
	MaxIdleConnDuration       string `json:"maxIdleConnDuration"`
	MaxConnDuration           string `json:"maxConnDuration"`
	MaxIdemponentCallAttempts int    `json:"maxIdemponentCallAttempts"`
	ReadBufferSize            string `json:"readBufferSize"`
	ReadTimeout               string `json:"readTimeout"`
	WriteBufferSize           string `json:"writeBufferSize"`
	WriteTimeout              string `json:"writeTimeout"`
	MaxResponseBodySize       string `json:"maxResponseBodySize"`
	MaxConnWaitTimeout        string `json:"maxConnWaitTimeout"`
}

type FastHttpClient struct {
	ssl     bool
	address string
	tr      *fasthttp.Client
}

func (client *FastHttpClient) Get(ctx context.Context, path string, header http.Header) (status int, respHeader http.Header, respBody []byte, err error) {
	req := client.prepareRequest(bytex.FromString(http.MethodGet), path, header)
	resp := fasthttp.AcquireResponse()
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		err = client.tr.DoDeadline(req, resp, deadline)
	} else {
		err = client.tr.Do(req, resp)
	}
	if err != nil {
		err = errors.Warning("fns: fasthttp client do get failed").WithCause(err)
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
		return
	}
	status = resp.StatusCode()
	respHeader = http.Header{}
	resp.Header.VisitAll(func(key, value []byte) {
		respHeader.Add(bytex.ToString(key), bytex.ToString(value))
	})
	respBody = resp.Body()
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)
	return
}

func (client *FastHttpClient) Post(ctx context.Context, path string, header http.Header, body []byte) (status int, respHeader http.Header, respBody []byte, err error) {
	req := client.prepareRequest(bytex.FromString(http.MethodPost), path, header)
	if body != nil && len(body) > 0 {
		req.SetBodyRaw(body)
	}
	resp := fasthttp.AcquireResponse()
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		err = client.tr.DoDeadline(req, resp, deadline)
	} else {
		err = client.tr.Do(req, resp)
	}
	if err != nil {
		err = errors.Warning("fns: fasthttp client do post failed").WithCause(err)
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
		return
	}
	status = resp.StatusCode()
	respHeader = http.Header{}
	resp.Header.VisitAll(func(key, value []byte) {
		respHeader.Add(bytex.ToString(key), bytex.ToString(value))
	})
	respBody = resp.Body()
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)
	return
}

func (client *FastHttpClient) Close() {
	client.tr.CloseIdleConnections()
}

func (client *FastHttpClient) prepareRequest(method []byte, path string, header http.Header) (req *fasthttp.Request) {
	req = fasthttp.AcquireRequest()
	uri := req.URI()
	if client.ssl {
		uri.SetSchemeBytes(bytex.FromString("https"))
	} else {
		uri.SetSchemeBytes(bytex.FromString("http"))
	}
	uri.SetPathBytes(bytex.FromString(path))
	uri.SetHostBytes(bytex.FromString(client.address))
	req.Header.SetMethodBytes(method)
	if header != nil && len(header) > 0 {
		for k, vv := range header {
			for _, v := range vv {
				req.Header.Add(k, v)
			}
		}
	}
	req.Header.SetBytesKV(bytex.FromString(httpContentType), bytex.FromString(httpContentTypeJson))
	return
}

type FastHttp struct {
	log     logs.Logger
	ssl     bool
	address string
	ln      net.Listener
	client  *fasthttp.Client
	srv     *fasthttp.Server
}

func (srv *FastHttp) Name() (name string) {
	const srvName = "fasthttp"
	name = srvName
	return
}

func (srv *FastHttp) Build(options HttpOptions) (err error) {
	srv.log = options.Log
	srv.address = fmt.Sprintf(":%d", options.Port)
	srv.ssl = options.ServerTLS != nil

	opt := &FastHttpOptions{}
	optErr := options.Options.As(opt)
	if optErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(optErr).WithMeta("fns", "http")
		return
	}
	readBufferSize := uint64(0)
	if opt.ReadBufferSize != "" {
		readBufferSize, err = bytex.ToBytes(strings.TrimSpace(opt.ReadBufferSize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("readBufferSize must be bytes format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	readTimeout := 10 * time.Second
	if opt.ReadTimeout != "" {
		readTimeout, err = time.ParseDuration(strings.TrimSpace(opt.ReadTimeout))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("readTimeout must be time.Duration format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	writeBufferSize := uint64(0)
	if opt.WriteBufferSize != "" {
		writeBufferSize, err = bytex.ToBytes(strings.TrimSpace(opt.WriteBufferSize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("writeBufferSize must be bytes format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	writeTimeout := 10 * time.Second
	if opt.WriteTimeout != "" {
		writeTimeout, err = time.ParseDuration(strings.TrimSpace(opt.WriteTimeout))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("writeTimeout must be time.Duration format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	maxIdleWorkerDuration := time.Duration(0)
	if opt.MaxIdleWorkerDuration != "" {
		maxIdleWorkerDuration, err = time.ParseDuration(strings.TrimSpace(opt.MaxIdleWorkerDuration))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("maxIdleWorkerDuration must be time.Duration format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	tcpKeepalivePeriod := time.Duration(0)
	if opt.TCPKeepalivePeriod != "" {
		tcpKeepalivePeriod, err = time.ParseDuration(strings.TrimSpace(opt.TCPKeepalivePeriod))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("tcpKeepalivePeriod must be time.Duration format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}

	maxRequestBodySize := uint64(4 * bytex.MEGABYTE)
	if opt.MaxRequestBodySize != "" {
		maxRequestBodySize, err = bytex.ToBytes(strings.TrimSpace(opt.MaxRequestBodySize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("maxRequestBodySize must be bytes format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}

	reduceMemoryUsage := opt.ReduceMemoryUsage

	srv.srv = &fasthttp.Server{
		Handler:                            fasthttpadaptor.NewFastHTTPHandler(options.Handler),
		ErrorHandler:                       fastHttpErrorHandler,
		Name:                               "",
		Concurrency:                        0,
		ReadBufferSize:                     int(readBufferSize),
		WriteBufferSize:                    int(writeBufferSize),
		ReadTimeout:                        readTimeout,
		WriteTimeout:                       writeTimeout,
		MaxRequestsPerConn:                 opt.MaxRequestsPerConn,
		MaxIdleWorkerDuration:              maxIdleWorkerDuration,
		TCPKeepalivePeriod:                 tcpKeepalivePeriod,
		MaxRequestBodySize:                 int(maxRequestBodySize),
		DisableKeepalive:                   false,
		TCPKeepalive:                       opt.TCPKeepalive,
		ReduceMemoryUsage:                  reduceMemoryUsage,
		GetOnly:                            false,
		DisablePreParseMultipartForm:       true,
		LogAllErrors:                       false,
		SecureErrorLogMessage:              false,
		DisableHeaderNamesNormalizing:      false,
		SleepWhenConcurrencyLimitsExceeded: 10 * time.Second,
		NoDefaultServerHeader:              true,
		NoDefaultDate:                      false,
		NoDefaultContentType:               false,
		KeepHijackedConns:                  opt.KeepHijackedConns,
		CloseOnShutdown:                    true,
		StreamRequestBody:                  opt.StreamRequestBody,
		ConnState:                          nil,
		Logger:                             logs.MapToLogger(options.Log, logs.DebugLevel, false),
		TLSConfig:                          options.ServerTLS,
		FormValueFunc:                      nil,
	}
	// client
	err = srv.buildClient(opt.Client, options.ClientTLS)
	return
}

func (srv *FastHttp) buildClient(opt FastHttpClientOptions, cliConfig *tls.Config) (err error) {
	maxIdleWorkerDuration := time.Duration(0)
	if opt.MaxIdleConnDuration != "" {
		maxIdleWorkerDuration, err = time.ParseDuration(strings.TrimSpace(opt.MaxIdleConnDuration))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxIdleWorkerDuration must be time.Duration format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	maxConnDuration := time.Duration(0)
	if opt.MaxConnDuration != "" {
		maxConnDuration, err = time.ParseDuration(strings.TrimSpace(opt.MaxConnDuration))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxConnDuration must be time.Duration format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	readBufferSize := uint64(0)
	if opt.ReadBufferSize != "" {
		readBufferSize, err = bytex.ToBytes(strings.TrimSpace(opt.ReadBufferSize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("readBufferSize must be bytes format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	readTimeout := 10 * time.Second
	if opt.ReadTimeout != "" {
		readTimeout, err = time.ParseDuration(strings.TrimSpace(opt.ReadTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("readTimeout must be time.Duration format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	writeBufferSize := uint64(0)
	if opt.WriteBufferSize != "" {
		writeBufferSize, err = bytex.ToBytes(strings.TrimSpace(opt.WriteBufferSize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("writeBufferSize must be bytes format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	writeTimeout := 10 * time.Second
	if opt.WriteTimeout != "" {
		writeTimeout, err = time.ParseDuration(strings.TrimSpace(opt.WriteTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("writeTimeout must be time.Duration format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	maxResponseBodySize := uint64(4 * bytex.MEGABYTE)
	if opt.MaxResponseBodySize != "" {
		maxResponseBodySize, err = bytex.ToBytes(strings.TrimSpace(opt.MaxResponseBodySize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxResponseBodySize must be bytes format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	maxConnWaitTimeout := time.Duration(0)
	if opt.MaxConnWaitTimeout != "" {
		maxConnWaitTimeout, err = time.ParseDuration(strings.TrimSpace(opt.MaxConnWaitTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxConnWaitTimeout must be time.Duration format")).WithCause(err).WithMeta("fns", "http")
			return
		}
	}
	srv.client = &fasthttp.Client{
		Name:                          "",
		NoDefaultUserAgentHeader:      true,
		Dial:                          nil,
		DialDualStack:                 false,
		TLSConfig:                     cliConfig,
		MaxConnsPerHost:               opt.MaxConnsPerHost,
		MaxIdleConnDuration:           maxIdleWorkerDuration,
		MaxConnDuration:               maxConnDuration,
		MaxIdemponentCallAttempts:     opt.MaxIdemponentCallAttempts,
		ReadBufferSize:                int(readBufferSize),
		WriteBufferSize:               int(writeBufferSize),
		ReadTimeout:                   readTimeout,
		WriteTimeout:                  writeTimeout,
		MaxResponseBodySize:           int(maxResponseBodySize),
		DisableHeaderNamesNormalizing: false,
		DisablePathNormalizing:        false,
		MaxConnWaitTimeout:            maxConnWaitTimeout,
		RetryIf:                       nil,
		ConnPoolStrategy:              0,
		ConfigureClient:               nil,
	}
	return
}

func (srv *FastHttp) Dial(address string) (client HttpClient, err error) {
	client = &FastHttpClient{
		ssl:     srv.ssl,
		address: address,
		tr:      srv.client,
	}
	return
}

func (srv *FastHttp) ListenAndServe() (err error) {
	if srv.ssl {
		err = srv.srv.ListenAndServeTLS(srv.address, "", "")
	} else {
		err = srv.srv.ListenAndServe(srv.address)
	}
	if err != nil {
		err = errors.Warning("fns: server listen and serve failed").WithCause(err).WithMeta("fns", "http")
		return
	}
	return
}

func (srv *FastHttp) Close() (err error) {
	err = srv.srv.Shutdown()
	if err != nil {
		err = errors.Warning("fns: server close failed").WithCause(err).WithMeta("fns", "http")
	}
	return
}

func fastHttpErrorHandler(ctx *fasthttp.RequestCtx, err error) {
	ctx.SetStatusCode(555)
	ctx.SetContentType(httpContentTypeJson)
	ctx.SetBody([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))
}
