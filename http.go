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
	"github.com/aacfactory/fns/cluster"
	"github.com/aacfactory/fns/documents"
	"github.com/aacfactory/fns/internal/cors"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
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

type HttpHandlerWrapperBuilder func(env Environments) (wrapper HttpHandlerWrapper, err error)

type HttpHandlerWrapper interface {
	Wrap(h http.Handler) http.Handler
}

func defaultHttpHandlerWrapperBuilders() (builders []HttpHandlerWrapperBuilder) {
	builders = make([]HttpHandlerWrapperBuilder, 0, 1)
	builders = append(builders, httpCorsHandlerWrapperBuilder, healthHandlerWrapperBuilder)
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func healthHandlerWrapperBuilder(env Environments) (wrapper HttpHandlerWrapper, err error) {
	wrapper = &healthHandlerWrapper{
		env:         env,
		unavailable: json.UnsafeMarshal(errors.Unavailable("fns: service is unavailable").WithMeta("fns", "http")),
	}
	return
}

type healthHandlerWrapper struct {
	env         Environments
	unavailable []byte
}

func (handler healthHandlerWrapper) Wrap(h http.Handler) http.Handler {
	if !handler.env.Running() {
		return http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
			writer.Header().Set(httpServerHeader, httpServerHeaderValue)
			writer.Header().Set(httpContentType, httpContentTypeJson)
			writer.Header().Set(httpConnectionHeader, httpConnectionHeaderClose)
			writer.WriteHeader(http.StatusServiceUnavailable)
			_, _ = writer.Write(handler.unavailable)
		})
	}
	return h
}

// +-------------------------------------------------------------------------------------------------------------------+

type HttpServerOptions struct {
	Port    int
	TLS     *tls.Config
	Handler http.Handler
	Log     logs.Logger
	raw     *json.Object
}

func (options HttpServerOptions) Get(key string, value interface{}) (err error) {
	err = options.raw.Get(key, value)
	if err != nil {
		err = errors.Warning(fmt.Sprintf("fns: http server options get %s failed", key)).WithCause(err)
		return
	}
	return
}

type HttpServerBuilder func(options HttpServerOptions) (server HttpServer, err error)

type HttpServer interface {
	ListenAndServe() (err error)
	Close() (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

func httpCorsHandlerWrapperBuilder(env Environments) (wrapper HttpHandlerWrapper, err error) {
	corsConfig, hasCorsConfig := env.Config("cors")
	if !hasCorsConfig {
		wrapper = &httpCorsHandlerWrapper{
			cors: cors.AllowAll(),
		}
		return
	}
	config := &HttpCorsConfig{}
	asErr := corsConfig.As(config)
	if asErr != nil {
		err = errors.Warning("fns: create cors http handler wrapper failed").WithCause(asErr)
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
	wrapper = &httpCorsHandlerWrapper{
		cors: cors.New(opt),
	}
	return
}

type httpCorsHandlerWrapper struct {
	cors *cors.Cors
}

func (wrapper *httpCorsHandlerWrapper) Wrap(h http.Handler) http.Handler {
	return wrapper.cors.Handler(h)
}

// +-------------------------------------------------------------------------------------------------------------------+

func fastHttpBuilder(options HttpServerOptions) (srv HttpServer, err error) {
	var ln net.Listener
	if options.TLS == nil {
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", options.Port))
	} else {
		ln, err = tls.Listen("tcp", fmt.Sprintf(":%d", options.Port), options.TLS)
	}
	if err != nil {
		err = errors.Warning("fns: create net listener failed").WithCause(err)
		return
	}
	srv = &fastHttp{
		log: options.Log,
		ln:  ln,
		srv: &fasthttp.Server{
			Handler:                            fasthttpadaptor.NewFastHTTPHandler(options.Handler),
			ErrorHandler:                       fastHttpErrorHandler,
			ReadTimeout:                        2 * time.Second,
			MaxIdleWorkerDuration:              10 * time.Second,
			MaxRequestBodySize:                 4 * MB,
			ReduceMemoryUsage:                  true,
			DisablePreParseMultipartForm:       true,
			SleepWhenConcurrencyLimitsExceeded: 10 * time.Second,
			NoDefaultServerHeader:              true,
			NoDefaultDate:                      false,
			NoDefaultContentType:               false,
			CloseOnShutdown:                    true,
			Logger:                             &printf{core: options.Log},
		},
	}
	return
}

func fastHttpErrorHandler(ctx *fasthttp.RequestCtx, err error) {
	ctx.SetStatusCode(555)
	ctx.SetContentType(httpContentTypeJson)
	ctx.SetBody([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))
}

type fastHttp struct {
	log logs.Logger
	ln  net.Listener
	srv *fasthttp.Server
}

func (srv *fastHttp) ListenAndServe() (err error) {
	err = srv.srv.Serve(srv.ln)
	if err != nil {
		err = errors.Warning("fns: server listen failed").WithCause(err).WithMeta("scope", "http")
		return
	}
	return
}

func (srv *fastHttp) Close() (err error) {
	err = srv.srv.Shutdown()
	if err != nil {
		err = errors.Warning("fns: server close failed").WithCause(err).WithMeta("scope", "http")
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func fastHttpClientBuilder(options cluster.ClientOptions) (client cluster.Client, err error) {
	maxIdleConnDuration := options.MaxIdleConnDuration
	if maxIdleConnDuration < 1 {
		maxIdleConnDuration = fasthttp.DefaultMaxIdleConnDuration
	}
	maxConnsPerHost := options.MaxConnsPerHost
	if maxConnsPerHost < 1 {
		maxConnsPerHost = fasthttp.DefaultMaxConnsPerHost
	}
	client = &fastHttpClient{
		log: options.Log.With("fns", "client"),
		client: &fasthttp.Client{
			Name:                     "FNS",
			NoDefaultUserAgentHeader: false,
			TLSConfig:                options.TLS,
			MaxConnsPerHost:          maxConnsPerHost,
			MaxIdleConnDuration:      maxIdleConnDuration,
			ReadBufferSize:           4 * KB,
			WriteBufferSize:          4 * MB,
			ReadTimeout:              5 * time.Second,
			MaxResponseBodySize:      4 * MB,
		},
	}
	return
}

type fastHttpClient struct {
	log    logs.Logger
	client *fasthttp.Client
}

func (client *fastHttpClient) Do(ctx sc.Context, method string, url string, header http.Header, body []byte) (respBody []byte, err error) {
	deadline, hasDeadline := ctx.Deadline()
	req := fasthttp.AcquireRequest()
	if !hasDeadline {
		deadline = time.Now().Add(10 * time.Second)
	}
	req.Header.SetMethod(method)
	if header != nil {
		for k, v := range header {
			if v != nil {
				for _, vv := range v {
					req.Header.Add(k, vv)
				}
			}
		}
	}
	req.SetRequestURI(url)
	if body != nil && len(body) > 0 {
		req.SetBody(body)
	}
	resp := fasthttp.AcquireResponse()
	doErr := client.client.DoDeadline(req, resp, deadline)
	fasthttp.ReleaseRequest(req)
	if doErr != nil {
		fasthttp.ReleaseResponse(resp)
		err = errors.Warning("fns: fasthttp client do failed").WithMeta("url", url).WithCause(doErr)
		return
	}
	respBody = resp.Body()
	fasthttp.ReleaseResponse(resp)
	return
}

func (client *fastHttpClient) Close() {
	client.client.CloseIdleConnections()
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
	document             *documents.Application
	barrier              Barrier
	requestHandleTimeout time.Duration
	wsm                  *websocketManager
	runtime              Runtime
	tracerReporter       TracerReporter
	hooks                *hooks
}

func newHttpHandler(env Environments, opt httpHandlerOptions) (handler *httpHandler) {
	handler = &httpHandler{
		env:                  env,
		log:                  env.Log().With("fns", "http"),
		requestCounter:       sync.WaitGroup{},
		document:             opt.document,
		barrier:              opt.barrier,
		requestHandleTimeout: opt.requestHandleTimeout,
		wsm:                  opt.wsm,
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
	document             *documents.Application
	barrier              Barrier
	requestHandleTimeout time.Duration
	wsm                  *websocketManager
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
	response.WriteHeader(http.StatusServiceUnavailable)
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
		h.env.AppId(), h.document.Version, h.document.Title, h.env.Running(), time.Now().Format(time.RFC3339Nano),
	)
	h.succeed(response, []byte(body))
}

func (h *httpHandler) documentRAW(response http.ResponseWriter) {
	raw, rawErr := h.document.Json()
	if rawErr != nil {
		h.failed(response, rawErr)
		return
	}
	h.succeed(response, raw)
}

func (h *httpHandler) documentOAS(response http.ResponseWriter) {
	raw, rawErr := h.document.OAS()
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
	if body == nil || len(body) == 0 || bytes.Equal(body, nullJson) {
		return
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
		has, getErr := result.Get(ctx, &resultBytes)
		if getErr != nil {
			err = getErr
			return
		}
		if has {
			v = resultBytes
		}
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
		if handleResult == nil {
			h.succeed(response, nil)
		} else {
			h.succeed(response, handleResult.([]byte))
		}
	} else {
		h.failed(response, codeErr)
	}
	// done
	h.requestCounter.Done()
	// report tracer
	h.tracerReporter.Report(ctx.Fork(sc.TODO()), ctx.Tracer())
	// hook
	h.hooks.send(newHookUnit(ctx, service, fn, body, codeErr, ctx.tracer.RootSpan().Latency()))
}

func (h *httpHandler) handleInternalRequest(response http.ResponseWriter, request *http.Request, service string, fn string) {
	body, readBodyErr := ioutil.ReadAll(request.Body)
	if readBodyErr != nil {
		h.notAcceptable(response)
		return
	}
	// proxy request
	ctxData, arg, decodeErr := decodeProxyRequest(body)
	if decodeErr != nil {
		h.notAcceptable(response)
		return
	}

	timeoutCtx, cancel := sc.WithTimeout(request.Context(), h.requestHandleTimeout)
	ctx := newContext(timeoutCtx, newRequest(request), ctxData, h.runtime)

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
	has, handleErr := result.Get(ctx, &resultBytes)
	cancel()
	// latency
	response.Header().Set(httpLatencyHeader, ctx.tracer.RootSpan().Latency().String())
	// requestId
	response.Header().Set(httpIdHeader, ctx.Request().Id())
	if !has {
		resultBytes = nil
	}
	responseBody, responseErr := encodeProxyResponse(handleErr != nil, ctx.Tracer().RootSpan(), resultBytes, handleErr)
	if responseErr != nil {
		h.failed(response, responseErr)
		h.requestCounter.Done()
		return
	}
	h.succeed(response, responseBody)
	// done
	h.requestCounter.Done()
	// hook
	h.hooks.send(newHookUnit(ctx, service, fn, body, handleErr, ctx.tracer.RootSpan().Latency()))
}

func (h *httpHandler) handleWebsocket(response http.ResponseWriter, request *http.Request) {
	conn, connErr := h.wsm.Upgrader().Upgrade(response, request, nil)

}

func (h *httpHandler) Close() (err error) {
	h.requestCounter.Wait()
	h.wsm.Close()
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

// +-------------------------------------------------------------------------------------------------------------------+
