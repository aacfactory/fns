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

const (
	httpContentType            = "Content-Type"
	httpContentTypeJson        = "application/json"
	httpConnectionHeader       = "Connection"
	httpUpgradeHeader          = "Upgrade"
	httpCloseHeader            = "close"
	httpXForwardedForHeader    = "X-Forwarded-For"
	httpAppIdHeader            = "X-Fns-Id"
	httpAppNameHeader          = "X-Fns-Name"
	httpAppVersionHeader       = "X-Fns-Version"
	httpRequestIdHeader        = "X-Fns-Request-Id"
	httpRequestSignatureHeader = "X-Fns-Request-Signature"
	httpRequestTimeoutHeader   = "X-Fns-Request-Timeout"
	httpRequestVersionsHeader  = "X-Fns-Request-Version"
	httpHandleLatencyHeader    = "X-Fns-Handle-Latency"
	httpDeviceIdHeader         = "X-Fns-Device-Id"
	httpDeviceIpHeader         = "X-Fns-Device-Ip"
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
	Accept(request *http.Request) (ok bool)
	Close()
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
	counter       sync.WaitGroup
	group         singleflight.Group
	requestCounts int64
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

func (handlers *HttpHandlers) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handlers.counter.Add(1)
	atomic.AddInt64(&handlers.requestCounts, 1)
	writer.Header().Set(httpAppIdHeader, handlers.appId)
	writer.Header().Set(httpAppNameHeader, handlers.appName)
	writer.Header().Set(httpAppVersionHeader, handlers.appVersion.String())
	if handlers.running.IsOff() {
		writer.Header().Set(httpConnectionHeader, httpCloseHeader)
		writer.Header().Set(httpContentType, httpContentTypeJson)
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write(json.UnsafeMarshal(errors.Unavailable("fns: service is unavailable").WithMeta("fns", "handlers")))
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
		writer.WriteHeader(http.StatusNotAcceptable)
		_, _ = writer.Write(json.UnsafeMarshal(errors.NotAcceptable("fns: request is not accepted").WithMeta("fns", "handlers")))
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
			handlersKey = "stats"
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
	opt.Options = config.Options
	return
}

type HttpOptions struct {
	Port      int
	ServerTLS *tls.Config
	ClientTLS *tls.Config
	Handler   http.Handler
	Log       logs.Logger
	Options   json.RawMessage
}

type HttpClient interface {
	Get(path string, header http.Header) (status int, respHeader http.Header, respBody []byte, err error)
	Post(path string, header http.Header, body []byte) (status int, respHeader http.Header, respBody []byte, err error)
}

type HttpClientDialer interface {
	Dial(address string) (client HttpClient, err error)
}

type Http interface {
	Build(options HttpOptions) (err error)
	HttpClientDialer
	ListenAndServe() (err error)
	Close() (err error)
}

type FastHttpOptions struct {
	ReadTimeoutSeconds   int    `json:"readTimeoutSeconds"`
	MaxWorkerIdleSeconds int    `json:"maxWorkerIdleSeconds"`
	MaxRequestBody       string `json:"maxRequestBody"`
	ReduceMemoryUsage    bool   `json:"reduceMemoryUsage"`
}

type FastHttp struct {
	log logs.Logger
	ln  net.Listener
	srv *fasthttp.Server
}

func (srv *FastHttp) Build(options HttpOptions) (err error) {
	srv.log = options.Log
	var ln net.Listener
	if options.ServerTLS == nil {
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", options.Port))
	} else {
		ln, err = tls.Listen("tcp", fmt.Sprintf(":%d", options.Port), options.ServerTLS)
	}
	if err != nil {
		err = errors.Warning("fns: build server failed").WithCause(err).WithMeta("fns", "http")
		return
	}
	srv.ln = ln

	opt := &FastHttpOptions{}
	optErr := json.Unmarshal(options.Options, opt)
	if optErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(optErr).WithMeta("fns", "http")
		return
	}
	if opt.ReadTimeoutSeconds < 1 {
		opt.ReadTimeoutSeconds = 2
	}
	if opt.MaxWorkerIdleSeconds < 1 {
		opt.MaxWorkerIdleSeconds = 10
	}
	maxRequestBody := strings.ToUpper(strings.TrimSpace(opt.MaxRequestBody))
	if maxRequestBody == "" {
		maxRequestBody = "4MB"
	}
	maxRequestBodySize, maxRequestBodySizeErr := bytex.ToBytes(maxRequestBody)
	if maxRequestBodySizeErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(maxRequestBodySizeErr).WithMeta("fns", "http")
		return
	}
	reduceMemoryUsage := opt.ReduceMemoryUsage

	srv.srv = &fasthttp.Server{
		Handler:                            fasthttpadaptor.NewFastHTTPHandler(options.Handler),
		ErrorHandler:                       fastHttpErrorHandler,
		ReadTimeout:                        time.Duration(opt.ReadTimeoutSeconds) * time.Second,
		MaxIdleWorkerDuration:              time.Duration(opt.MaxWorkerIdleSeconds) * time.Second,
		MaxRequestBodySize:                 int(maxRequestBodySize),
		ReduceMemoryUsage:                  reduceMemoryUsage,
		DisablePreParseMultipartForm:       true,
		SleepWhenConcurrencyLimitsExceeded: 10 * time.Second,
		NoDefaultServerHeader:              true,
		NoDefaultDate:                      false,
		NoDefaultContentType:               false,
		CloseOnShutdown:                    true,
		Logger:                             logs.MapToLogger(options.Log, logs.DebugLevel, false),
	}
	return
}

func (srv *FastHttp) ListenAndServe() (err error) {
	err = srv.srv.Serve(srv.ln)
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
