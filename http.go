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

package fns

import (
	"bytes"
	sc "context"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons"
	"github.com/aacfactory/fns/cors"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"io/ioutil"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

func defaultHttpHandlerWrappers() (wrappers []HttpHandlerWrapper) {
	wrappers = make([]HttpHandlerWrapper, 0, 1)
	wrappers = append(wrappers, &corsHttpHandlerWrapper{})
	return
}

type HttpHandlerWrapper interface {
	Build(env Environments) (err error)
	Handler(h http.Handler) http.Handler
}

type HttpServer interface {
	Build(env Environments, handler http.Handler) (err error)
	Listen() (err error)
	Close() (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

type corsHttpHandlerWrapper struct {
	cors *cors.Cors
}

func (wrapper *corsHttpHandlerWrapper) Build(env Environments) (err error) {
	httpConfig, hasHttp := env.Config("http")
	if !hasHttp {
		wrapper.cors = cors.AllowAll()
		return
	}
	config := &httpCorsConfig{}
	hasCors, getErr := httpConfig.Get("cors", config)
	if getErr != nil {
		err = errors.Warning("fns: create cors http handler wrapper failed").WithCause(getErr)
		return
	}
	if !hasCors {
		wrapper.cors = cors.AllowAll()
		return
	}
	allowedOrigins := config.AllowedOrigins
	if allowedOrigins == nil {
		allowedOrigins = make([]string, 0, 1)
	}
	if len(allowedOrigins) == 0 {
		allowedOrigins = append(allowedOrigins, "*")
	}
	allowedHeaders := config.AllowedHeaders
	if allowedHeaders == nil {
		allowedHeaders = make([]string, 0, 1)
	}
	if sort.SearchStrings(allowedHeaders, "Connection") < 0 {
		allowedHeaders = append(allowedHeaders, "Connection")
	}
	if sort.SearchStrings(allowedHeaders, "Upgrade") < 0 {
		allowedHeaders = append(allowedHeaders, "Upgrade")
	}
	if sort.SearchStrings(allowedHeaders, httpXForwardedFor) < 0 {
		allowedHeaders = append(allowedHeaders, httpXForwardedFor)
	}
	if sort.SearchStrings(allowedHeaders, httpXRealIp) < 0 {
		allowedHeaders = append(allowedHeaders, httpXRealIp)
	}
	exposedHeaders := config.ExposedHeaders
	if exposedHeaders == nil {
		exposedHeaders = make([]string, 0, 1)
	}
	exposedHeaders = append(exposedHeaders, httpIdHeader, httpLatencyHeader, httpConnectionHeader, "Server")
	opt := cors.Options{
		AllowedOrigins:       config.AllowedOrigins,
		AllowedMethods:       []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:       allowedHeaders,
		ExposedHeaders:       exposedHeaders,
		MaxAge:               config.MaxAge,
		AllowCredentials:     config.AllowCredentials,
		AllowPrivateNetwork:  true,
		OptionsPassthrough:   false,
		OptionsSuccessStatus: http.StatusNoContent,
	}
	wrapper.cors = cors.New(opt)
	return
}

func (wrapper *corsHttpHandlerWrapper) Handler(h http.Handler) http.Handler {
	if wrapper.cors == nil {
		return h
	}
	return wrapper.cors.Handler(h)
}

// +-------------------------------------------------------------------------------------------------------------------+

type fastHttpConfig struct {
	Concurrency              int    `json:"concurrency,omitempty"`
	ReduceMemoryUsage        bool   `json:"reduceMemoryUsage"`
	MaxConnectionsPerIP      int    `json:"maxConnectionsPerIp,omitempty"`
	MaxRequestsPerConnection int    `json:"maxRequestsPerConnection,omitempty"`
	KeepAlive                bool   `json:"keepAlive,omitempty"`
	KeepalivePeriodSecond    int    `json:"keepalivePeriodSecond,omitempty"`
	RequestTimeoutSeconds    int    `json:"requestTimeoutSeconds,omitempty"`
	MaxRequestHeaderSize     string `json:"maxRequestHeaderSize"`
	MaxRequestBodySize       string `json:"maxRequestBodySize"`
}

type fastHttp struct {
	handler http.Handler
	port    int
	ssl     *tls.Config
	srv     *fasthttp.Server
}

func (srv *fastHttp) Build(env Environments, handler http.Handler) (err error) {
	port := 80
	opt := &fastHttpConfig{
		Concurrency:              workers.DefaultConcurrency,
		ReduceMemoryUsage:        true,
		MaxConnectionsPerIP:      0,
		MaxRequestsPerConnection: 0,
		KeepAlive:                true,
		KeepalivePeriodSecond:    10,
		RequestTimeoutSeconds:    10,
		MaxRequestHeaderSize:     "2KB",
		MaxRequestBodySize:       "2KB",
	}

	config, hasConfig := env.Config("http")
	if hasConfig {
		httpConfig := &HttpConfig{}
		decodeErr := config.As(httpConfig)
		if decodeErr != nil {
			err = errors.Warning("fns: server build failed").WithCause(decodeErr).WithMeta("scope", "server")
			return
		}
		port = httpConfig.Port
		if port < 1024 || port > 65535 {
			err = errors.Warning("fns: server build failed").WithCause(fmt.Errorf("port is out of range")).WithMeta("scope", "server")
			return
		}
		if httpConfig.Options != nil && len(httpConfig.Options) > 1 {
			decodeOptErr := json.Unmarshal(httpConfig.Options, opt)
			if decodeOptErr != nil {
				err = errors.Warning("fns: server build failed").WithCause(decodeOptErr).WithMeta("scope", "server")
				return
			}
		}
		if httpConfig.TLS != nil {
			srv.ssl, err = httpConfig.TLS.Config()
			if err != nil {
				err = errors.Warning("fns: server build failed").WithCause(err).WithMeta("scope", "server")
				return
			}
		}
	}
	maxRequestHeaderSizeSTR := strings.TrimSpace(opt.MaxRequestHeaderSize)
	if maxRequestHeaderSizeSTR == "" {
		maxRequestHeaderSizeSTR = "2KB"
	}
	maxRequestHeaderSize, maxRequestHeaderSizeErr := commons.ToBytes(maxRequestHeaderSizeSTR)
	if maxRequestHeaderSizeErr != nil {
		err = errors.Warning("fns: server build failed").WithCause(maxRequestHeaderSizeErr).WithMeta("scope", "server")
		return
	}
	MaxRequestBodySizeSTR := strings.TrimSpace(opt.MaxRequestBodySize)
	if MaxRequestBodySizeSTR == "" {
		MaxRequestBodySizeSTR = "2KB"
	}
	maxRequestBodySize, maxRequestBodySizeErr := commons.ToBytes(MaxRequestBodySizeSTR)
	if maxRequestBodySizeErr != nil {
		err = errors.Warning("fns: server build failed").WithCause(maxRequestBodySizeErr).WithMeta("scope", "server")
		return
	}
	srv.srv = &fasthttp.Server{
		Handler:                            fasthttpadaptor.NewFastHTTPHandler(handler),
		ErrorHandler:                       srv.ErrorHandler,
		Concurrency:                        opt.Concurrency,
		ReadBufferSize:                     int(maxRequestHeaderSize + maxRequestBodySize),
		ReadTimeout:                        time.Duration(opt.RequestTimeoutSeconds) * time.Second,
		MaxConnsPerIP:                      opt.MaxConnectionsPerIP,
		MaxRequestsPerConn:                 opt.MaxRequestsPerConnection,
		MaxIdleWorkerDuration:              10 * time.Second,
		TCPKeepalivePeriod:                 time.Duration(opt.KeepalivePeriodSecond) * time.Second,
		MaxRequestBodySize:                 int(maxRequestBodySize),
		DisableKeepalive:                   !opt.KeepAlive,
		ReduceMemoryUsage:                  opt.ReduceMemoryUsage,
		DisablePreParseMultipartForm:       true,
		SleepWhenConcurrencyLimitsExceeded: 10 * time.Second,
		NoDefaultServerHeader:              true,
		NoDefaultDate:                      false,
		NoDefaultContentType:               false,
		CloseOnShutdown:                    true,
		Logger:                             &printf{core: env.Log().With("fns", "fasthttp")},
	}
	return
}

func (srv *fastHttp) ErrorHandler(ctx *fasthttp.RequestCtx, err error) {
	ctx.SetStatusCode(555)
	ctx.SetContentType(httpContentTypeJson)
	ctx.SetBody([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))
}

func (srv *fastHttp) Listen() (err error) {
	var ln net.Listener
	if srv.ssl == nil {
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", srv.port))
	} else {
		ln, err = tls.Listen("tcp", fmt.Sprintf(":%d", srv.port), srv.ssl)
	}
	if err != nil {
		err = errors.Warning("fns: server listen failed").WithCause(err).WithMeta("scope", "server")
		return
	}
	err = srv.srv.Serve(ln)
	if err != nil {
		err = errors.Warning("fns: server listen failed").WithCause(err).WithMeta("scope", "server")
		return
	}
	return
}

func (srv *fastHttp) Close() (err error) {
	err = srv.srv.Shutdown()
	if err != nil {
		err = errors.Warning("fns: server close failed").WithCause(err).WithMeta("scope", "server")
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	// path
	httpHealthPath      = "/health"
	httpDocumentRawPath = "/documents/raw"
	httpDocumentOASPath = "/documents/oas.json"
	httpWebsocketPath   = "/websocket"

	// header
	httpServerHeader          = "Server"
	httpServerHeaderValue     = "FNS"
	httpContentType           = "Content-Type"
	httpContentTypeProxy      = "application/fns+proxy"
	httpContentTypeJson       = "application/json"
	httpAuthorizationHeader   = "Authorization"
	httpConnectionHeader      = "Connection"
	httpConnectionHeaderClose = "close"
	httpIdHeader              = "X-Fns-Request-Id"
	httpLatencyHeader         = "X-Fns-Latency"
	httpXForwardedFor         = "X-Forwarded-For"
	httpXRealIp               = "X-Real-Ip"
)

var (
	nullJson = []byte{'n', 'u', 'l', 'l'}
)

func newHttpHandlerErrBody() *httpHandlerErrBody {
	return &httpHandlerErrBody{
		methodNotAllowed: json.UnsafeMarshal(errors.New(405, "***METHOD NOT ALLOWED***", "fns: method is not allowed").WithMeta("fns", "http")),
		outOfRange:       json.UnsafeMarshal(errors.New(416, "***REQUEST RANGE NOT SATISFIABLE***", "fns: requested range not satisfiable").WithMeta("fns", "http")),
		unavailable:      json.UnsafeMarshal(errors.Unavailable("fns: service is unavailable").WithMeta("fns", "http")),
		timeout:          json.UnsafeMarshal(errors.Timeout("fns: request is timeout").WithMeta("fns", "http")),
		notAcceptable:    json.UnsafeMarshal(errors.NotAcceptable("fns: request is not acceptable").WithMeta("fns", "http")),
	}
}

type httpHandlerErrBody struct {
	methodNotAllowed []byte
	outOfRange       []byte
	unavailable      []byte
	timeout          []byte
	notAcceptable    []byte
}

type httpHandlerOptions struct {
	env                  Environments
	documents            *Documents
	barrier              Barrier
	requestHandleTimeout time.Duration
	websocketDiscovery   WebsocketDiscovery
	runtime              Runtime
	tracerReporter       TracerReporter
	hooks                *hooks
}

func newHttpHandler(env Environments, opt httpHandlerOptions) (handler *httpHandler, err error) {
	websocketUpgrader, websocketUpgraderErr := newWebsocketUpgrader(env)
	if websocketUpgrader != nil {
		err = errors.Warning("fns: create http handler failed").WithCause(websocketUpgraderErr)
		return
	}
	handler = &httpHandler{
		env:                  env,
		log:                  env.Log().With("fns", "http"),
		requestCounter:       sync.WaitGroup{},
		documents:            opt.documents,
		barrier:              opt.barrier,
		requestHandleTimeout: opt.requestHandleTimeout,
		websocketDiscovery:   opt.websocketDiscovery,
		websocketUpgrader:    websocketUpgrader,
		runtime:              opt.runtime,
		tracerReporter:       opt.tracerReporter,
		hooks:                opt.hooks,
		errBody:              newHttpHandlerErrBody(),
	}
	return
}

type httpHandler struct {
	env                  Environments
	log                  logs.Logger
	requestCounter       sync.WaitGroup
	documents            *Documents
	barrier              Barrier
	requestHandleTimeout time.Duration
	websocketDiscovery   WebsocketDiscovery
	websocketUpgrader    *websocket.Upgrader
	runtime              Runtime
	tracerReporter       TracerReporter
	hooks                *hooks
	errBody              *httpHandlerErrBody
}

func (h *httpHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	requestPath := request.URL.Path
	switch request.Method {
	case http.MethodGet:
		switch requestPath {
		case httpHealthPath, "":
			h.health(response)
			return
		case httpDocumentRawPath:
			h.documentRAW(response)
			return
		case httpDocumentOASPath:
			h.documentOAS(response)
			return
		case httpWebsocketPath:
			h.handleWebsocket(response, request)
			return
		default:
			response.Header().Set(httpServerHeader, httpServerHeaderValue)
			response.Header().Set(httpContentType, httpContentTypeJson)
			response.WriteHeader(416)
			_, _ = response.Write(h.errBody.outOfRange)
			return
		}
	case http.MethodPost:
		if !h.env.Running() {
			h.unavailable(response)
			return
		}
		// path /{service}/{fn}
		pathItems := strings.Split(requestPath, "/")
		if len(pathItems) != 3 {
			h.notAcceptable(response)
			return
		}
		service := pathItems[1]
		fn := pathItems[2]
		// content type
		contentType := request.Header.Get(httpContentType)
		switch contentType {
		case httpContentTypeJson:
			h.handleRequest(response, request, service, fn)
		case httpContentTypeProxy:
			h.handleInternalRequest(response, request, service, fn)
		default:
			h.notAcceptable(response)
			return
		}
	default:
		response.Header().Set(httpServerHeader, httpServerHeaderValue)
		response.Header().Set(httpContentType, httpContentTypeJson)
		response.WriteHeader(405)
		_, _ = response.Write(h.errBody.methodNotAllowed)
		return
	}
}

func (h *httpHandler) unavailable(response http.ResponseWriter) {
	response.Header().Set(httpServerHeader, httpServerHeaderValue)
	response.Header().Set(httpContentType, httpContentTypeJson)
	response.Header().Set(httpConnectionHeader, httpConnectionHeaderClose)
	response.WriteHeader(503)
	_, _ = response.Write(h.errBody.unavailable)
}

func (h *httpHandler) timeout(response http.ResponseWriter) {
	response.Header().Set(httpServerHeader, httpServerHeaderValue)
	response.Header().Set(httpContentType, httpContentTypeJson)
	response.WriteHeader(408)
	_, _ = response.Write(h.errBody.timeout)
}

func (h *httpHandler) notAcceptable(response http.ResponseWriter) {
	response.Header().Set(httpServerHeader, httpServerHeaderValue)
	response.Header().Set(httpContentType, httpContentTypeJson)
	response.WriteHeader(406)
	_, _ = response.Write(h.errBody.notAcceptable)
}

func (h *httpHandler) health(response http.ResponseWriter) {
	body := fmt.Sprintf(
		"{\"appId\":\"%s\", \"version\":\"%s\", \"title\":\"%s\", \"running\":\"%v\", \"now\":\"%s\"}",
		h.env.AppId(), h.documents.Version, h.documents.Title, h.env.Running(), time.Now().Format(time.RFC3339Nano),
	)
	h.succeed(response, []byte(body))
}

func (h *httpHandler) documentRAW(response http.ResponseWriter) {
	raw, rawErr := h.documents.json()
	if rawErr != nil {
		h.failed(response, rawErr)
		return
	}
	h.succeed(response, raw)
}

func (h *httpHandler) documentOAS(response http.ResponseWriter) {
	raw, rawErr := h.documents.oas()
	if rawErr != nil {
		h.failed(response, rawErr)
		return
	}
	h.succeed(response, raw)
}

func (h *httpHandler) succeed(response http.ResponseWriter, body []byte) {
	response.Header().Set(httpServerHeader, httpServerHeaderValue)
	response.Header().Set(httpContentType, httpContentTypeJson)
	response.WriteHeader(200)
	if body == nil || len(body) == 0 {
		body = nullJson
	}
	_, _ = response.Write(body)
}

func (h *httpHandler) failed(response http.ResponseWriter, codeErr errors.CodeError) {
	response.Header().Set(httpServerHeader, httpServerHeaderValue)
	response.Header().Set(httpContentType, httpContentTypeJson)
	response.WriteHeader(codeErr.Code())
	p, _ := json.Marshal(codeErr)
	_, _ = response.Write(p)
}

func (h *httpHandler) hashRequest(header http.Header, body []byte) (v string) {
	hash := md5.New()
	authorization := header.Get(httpAuthorizationHeader)
	if authorization != "" {
		hash.Write([]byte(authorization))
	}
	hash.Write(body)
	v = hex.EncodeToString(hash.Sum(nil))
	return
}

func (h *httpHandler) handleRequest(response http.ResponseWriter, request *http.Request, service string, fn string) {
	body, readBodyErr := ioutil.ReadAll(request.Body)
	if readBodyErr != nil {
		h.notAcceptable(response)
		return
	}
	var arg Argument
	if body == nil || len(body) == 0 || bytes.Equal(body, nullJson) {
		arg = EmptyArgument()
	} else {
		if !json.Validate(body) {
			h.notAcceptable(response)
			return
		}
		arg = NewArgument(body)
	}

	timeoutCtx, cancel := sc.WithTimeout(request.Context(), h.requestHandleTimeout)
	ctx := newContext(timeoutCtx, newRequest(request), newContextData(json.NewObject()), h.runtime)
	// handle
	h.requestCounter.Add(1)
	// endpoint
	endpoint, getEndpointErr := h.runtime.Endpoints().Get(ctx, service)
	if getEndpointErr != nil {
		h.failed(response, getEndpointErr)
		h.requestCounter.Done()
		cancel()
		return
	}

	// result
	barrierKey := fmt.Sprintf("%s:%s:%s", service, fn, h.hashRequest(request.Header, body))
	handleResult, handleErr, _ := h.barrier.Do(ctx, barrierKey, func() (v interface{}, err error) {
		result := endpoint.Request(ctx, fn, arg)
		resultBytes := json.RawMessage{}
		err = result.Get(ctx, &resultBytes)
		if err != nil {
			return
		}
		v = resultBytes
		return
	})
	h.barrier.Forget(ctx, barrierKey)
	cancel()
	// latency
	response.Header().Set(httpLatencyHeader, ctx.tracer.RootSpan().Latency().String())
	// requestId
	response.Header().Set(httpIdHeader, ctx.Request().Id())
	// write result
	var codeErr errors.CodeError = nil
	if handleErr != nil {
		codeErr0, ok := handleErr.(errors.CodeError)
		if ok {
			codeErr = codeErr0
		} else {
			codeErr = errors.Warning("fns: handle request failed").WithCause(handleErr)
		}
	}
	if codeErr == nil {
		h.succeed(response, handleResult.([]byte))
	} else {
		h.failed(response, codeErr)
	}
	// report tracer
	h.tracerReporter.Report(ctx.Tracer())
	// hook
	h.hooks.send(newHookUnit(service, fn, ctx.Request(), body, codeErr, ctx.tracer.RootSpan().Latency()))
	// done
	h.requestCounter.Done()
}

func (h *httpHandler) handleInternalRequest(response http.ResponseWriter, request *http.Request, service string, fn string) {
	body, readBodyErr := ioutil.ReadAll(request.Body)
	if readBodyErr != nil {
		h.notAcceptable(response)
		return
	}
	// proxy request
	proxyRequest := &serviceProxyRequest{}
	decodeBodyErr := proxyRequest.Decode(body)
	if decodeBodyErr != nil {
		h.notAcceptable(response)
		return
	}

	var arg Argument
	if proxyRequest.Argument == nil || len(proxyRequest.Argument) == 0 || bytes.Equal(proxyRequest.Argument, nullJson) {
		arg = EmptyArgument()
	} else {
		if !json.Validate(proxyRequest.Argument) {
			h.notAcceptable(response)
			return
		}
		arg = NewArgument(proxyRequest.Argument)
	}

	timeoutCtx, cancel := sc.WithTimeout(request.Context(), h.requestHandleTimeout)
	ctx := newContext(timeoutCtx, newRequest(request), proxyRequest.ContextData, h.runtime)

	// handle
	h.requestCounter.Add(1)
	// endpoint
	endpoint, getEndpointErr := h.runtime.Endpoints().Get(ctx, service)
	if getEndpointErr != nil {
		h.failed(response, getEndpointErr)
		h.requestCounter.Done()
		cancel()
		return
	}
	// result
	result := endpoint.Request(ctx, fn, arg)
	resultBytes := json.RawMessage{}
	handleErr := result.Get(ctx, &resultBytes)
	cancel()
	// latency
	response.Header().Set(httpLatencyHeader, ctx.tracer.RootSpan().Latency().String())
	// requestId
	response.Header().Set(httpIdHeader, ctx.Request().Id())

	proxyResponse := &serviceProxyResponse{
		Failed:      handleErr != nil,
		ContextData: ctx.Data(),
		Span:        ctx.Tracer().RootSpan(),
		Result:      resultBytes,
		Error:       handleErr,
	}
	proxyResponseBytes := json.UnsafeMarshal(proxyResponse)
	h.succeed(response, proxyResponseBytes)
	// hook
	h.hooks.send(newHookUnit(service, fn, ctx.Request(), body, handleErr, ctx.tracer.RootSpan().Latency()))
	// done
	h.requestCounter.Done()
}

func (h *httpHandler) handleWebsocket(response http.ResponseWriter, request *http.Request) {

}

func (h *httpHandler) Close() (err error) {
	h.requestCounter.Wait()
	// todo close all websocket
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

// +-------------------------------------------------------------------------------------------------------------------+