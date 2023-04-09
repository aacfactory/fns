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

package transports

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	fastHttpTransportName = "fasthttp"
)

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

type FastHttpTransportOptions struct {
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

// +-------------------------------------------------------------------------------------------------------------------+

type fastClient struct {
	ssl     bool
	address string
	tr      *fasthttp.Client
}

func (client *fastClient) Do(ctx context.Context, request *Request) (response *Response, err error) {
	req := fasthttp.AcquireRequest()
	// method
	req.Header.SetMethodBytes(request.method)
	// header
	if request.header != nil && len(request.header) > 0 {
		for k, vv := range request.header {
			if vv == nil || len(vv) == 0 {
				continue
			}
			for _, v := range vv {
				req.Header.Add(k, v)
			}
		}
	}
	// uri
	uri := req.URI()
	if client.ssl {
		uri.SetSchemeBytes(bytex.FromString("https"))
	} else {
		uri.SetSchemeBytes(bytex.FromString("http"))
	}
	uri.SetHostBytes(bytex.FromString(client.address))
	uri.SetPathBytes(request.path)
	if request.params != nil && len(request.params) > 0 {
		uri.SetQueryStringBytes(bytex.FromString(request.params.String()))
	}
	// body
	if request.body != nil && len(request.body) > 0 {
		req.SetBodyRaw(request.body)
	}
	// resp
	resp := fasthttp.AcquireResponse()
	// do
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		err = client.tr.DoDeadline(req, resp, deadline)
	} else {
		err = client.tr.Do(req, resp)
	}
	if err != nil {
		err = errors.Warning("fns: transport client do failed").
			WithCause(err).
			WithMeta("transport", fastHttpTransportName).WithMeta("method", bytex.ToString(request.method)).WithMeta("path", bytex.ToString(request.path))
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
		return
	}
	response = &Response{
		Status: resp.StatusCode(),
		Header: make(Header),
		Body:   resp.Body(),
	}
	resp.Header.VisitAll(func(key, value []byte) {
		response.Header.Add(bytex.ToString(key), bytex.ToString(value))
	})
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)
	return
}

func (client *fastClient) Close() {
	client.tr.CloseIdleConnections()
}

// +-------------------------------------------------------------------------------------------------------------------+

type fastHttpTransport struct {
	log     logs.Logger
	ssl     bool
	address string
	client  *fasthttp.Client
	server  *fasthttp.Server
}

func (srv *fastHttpTransport) Name() (name string) {
	name = fastHttpTransportName
	return
}

func (srv *fastHttpTransport) Build(options Options) (err error) {
	srv.log = options.Log
	srv.address = fmt.Sprintf(":%d", options.Port)
	srv.ssl = options.ServerTLS != nil

	opt := &FastHttpTransportOptions{}
	optErr := options.Config.As(opt)
	if optErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(optErr).WithMeta("transport", fastHttpTransportName)
		return
	}
	readBufferSize := uint64(0)
	if opt.ReadBufferSize != "" {
		readBufferSize, err = bytex.ParseBytes(strings.TrimSpace(opt.ReadBufferSize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("readBufferSize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	readTimeout := 10 * time.Second
	if opt.ReadTimeout != "" {
		readTimeout, err = time.ParseDuration(strings.TrimSpace(opt.ReadTimeout))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("readTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	writeBufferSize := uint64(0)
	if opt.WriteBufferSize != "" {
		writeBufferSize, err = bytex.ParseBytes(strings.TrimSpace(opt.WriteBufferSize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("writeBufferSize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	writeTimeout := 10 * time.Second
	if opt.WriteTimeout != "" {
		writeTimeout, err = time.ParseDuration(strings.TrimSpace(opt.WriteTimeout))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("writeTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	maxIdleWorkerDuration := time.Duration(0)
	if opt.MaxIdleWorkerDuration != "" {
		maxIdleWorkerDuration, err = time.ParseDuration(strings.TrimSpace(opt.MaxIdleWorkerDuration))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("maxIdleWorkerDuration must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	tcpKeepalivePeriod := time.Duration(0)
	if opt.TCPKeepalivePeriod != "" {
		tcpKeepalivePeriod, err = time.ParseDuration(strings.TrimSpace(opt.TCPKeepalivePeriod))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("tcpKeepalivePeriod must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}

	maxRequestBodySize := uint64(4 * bytex.MEGABYTE)
	if opt.MaxRequestBodySize != "" {
		maxRequestBodySize, err = bytex.ParseBytes(strings.TrimSpace(opt.MaxRequestBodySize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("maxRequestBodySize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}

	reduceMemoryUsage := opt.ReduceMemoryUsage

	srv.server = &fasthttp.Server{
		Handler:                            FastHttpTransportHandlerAdaptor(options.Handler),
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
	if err != nil {
		err = errors.Warning("fns: fasthttp build failed").WithCause(err).WithMeta("transport", fastHttpTransportName)
		return
	}
	return
}

func (srv *fastHttpTransport) buildClient(opt FastHttpClientOptions, cliConfig *tls.Config) (err error) {
	maxIdleWorkerDuration := time.Duration(0)
	if opt.MaxIdleConnDuration != "" {
		maxIdleWorkerDuration, err = time.ParseDuration(strings.TrimSpace(opt.MaxIdleConnDuration))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxIdleWorkerDuration must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	maxConnDuration := time.Duration(0)
	if opt.MaxConnDuration != "" {
		maxConnDuration, err = time.ParseDuration(strings.TrimSpace(opt.MaxConnDuration))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxConnDuration must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	readBufferSize := uint64(0)
	if opt.ReadBufferSize != "" {
		readBufferSize, err = bytex.ParseBytes(strings.TrimSpace(opt.ReadBufferSize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("readBufferSize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	readTimeout := 10 * time.Second
	if opt.ReadTimeout != "" {
		readTimeout, err = time.ParseDuration(strings.TrimSpace(opt.ReadTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("readTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	writeBufferSize := uint64(0)
	if opt.WriteBufferSize != "" {
		writeBufferSize, err = bytex.ParseBytes(strings.TrimSpace(opt.WriteBufferSize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("writeBufferSize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	writeTimeout := 10 * time.Second
	if opt.WriteTimeout != "" {
		writeTimeout, err = time.ParseDuration(strings.TrimSpace(opt.WriteTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("writeTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	maxResponseBodySize := uint64(4 * bytex.MEGABYTE)
	if opt.MaxResponseBodySize != "" {
		maxResponseBodySize, err = bytex.ParseBytes(strings.TrimSpace(opt.MaxResponseBodySize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxResponseBodySize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	maxConnWaitTimeout := time.Duration(0)
	if opt.MaxConnWaitTimeout != "" {
		maxConnWaitTimeout, err = time.ParseDuration(strings.TrimSpace(opt.MaxConnWaitTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxConnWaitTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
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

func (srv *fastHttpTransport) Dial(address string) (client Client, err error) {
	client = &fastClient{
		ssl:     srv.ssl,
		address: address,
		tr:      srv.client,
	}
	return
}

func (srv *fastHttpTransport) ListenAndServe() (err error) {
	if srv.ssl {
		err = srv.server.ListenAndServeTLS(srv.address, "", "")
	} else {
		err = srv.server.ListenAndServe(srv.address)
	}
	if err != nil {
		err = errors.Warning("fns: server listen and serve failed").WithCause(err).WithMeta("transport", fastHttpTransportName)
		return
	}
	return
}

func (srv *fastHttpTransport) Close() (err error) {
	err = srv.server.Shutdown()
	if err != nil {
		err = errors.Warning("fns: server close failed").WithCause(err).WithMeta("transport", fastHttpTransportName)
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func fastHttpErrorHandler(ctx *fasthttp.RequestCtx, err error) {
	ctx.SetStatusCode(555)
	ctx.SetContentType(contentTypeJsonHeaderValue)
	p, _ := json.Marshal(errors.Warning("fns: transport receiving or parsing the request failed").WithCause(err).WithMeta("transport", fastHttpTransportName))
	ctx.SetBody(p)
}

// +-------------------------------------------------------------------------------------------------------------------+

func FastHttpTransportHandlerAdaptor(h Handler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		r, convertErr := convertFastHttpRequestCtxToRequest(ctx)
		if convertErr != nil {
			p, _ := json.Marshal(errors.Warning("fns: fasthttp handler adapt failed ").WithCause(convertErr))
			ctx.Response.Reset()
			ctx.SetStatusCode(555)
			ctx.SetContentTypeBytes(bytex.FromString(contentTypeJsonHeaderValue))
			ctx.SetBody(p)
			return
		}

		buf := bytebufferpool.Get()
		w := convertFastHttpRequestCtxToResponseWriter(ctx, buf)

		h.Handle(w, r)

		for k, vv := range w.Header() {
			for _, v := range vv {
				ctx.Response.Header.Add(k, v)
			}
		}

		ctx.SetStatusCode(w.Status())

		if bodyLen := buf.Len(); bodyLen > 0 {
			body := buf.Bytes()
			n := 0
			for n < bodyLen {
				nn, writeErr := ctx.Write(body[n:])
				if writeErr != nil {
					break
				}
				n += nn
			}
		}

		bytebufferpool.Put(buf)
	}
}

func convertFastHttpRequestCtxToRequest(ctx *fasthttp.RequestCtx) (r *Request, err error) {
	r, err = NewRequest(ctx, ctx.Method(), ctx.RequestURI())
	if err != nil {
		err = errors.Warning("fns: convert fasthttp request to transport request failed").WithCause(err)
		return
	}

	r.SetHost(ctx.Host())

	if ctx.IsTLS() {
		r.UseTLS()
		r.tlsConnectionState = ctx.TLSConnectionState()
	}

	r.SetProto(ctx.Request.Header.Protocol())

	ctx.Request.Header.VisitAll(func(key, value []byte) {
		sk := bytex.ToString(key)
		sv := bytex.ToString(value)
		r.Header().Set(sk, sv)
	})

	if ctx.IsPost() || ctx.IsPut() {
		r.SetBody(ctx.PostBody())
	}

	return
}

func convertFastHttpRequestCtxToResponseWriter(ctx *fasthttp.RequestCtx, writer WriteBuffer) (w ResponseWriter) {
	w = &fastHttpResponseWriter{
		ctx:    ctx,
		status: 0,
		header: make(Header),
		body:   writer,
	}
	return
}

type fastHttpResponseWriter struct {
	ctx    *fasthttp.RequestCtx
	status int
	header Header
	body   WriteBuffer
}

func (w *fastHttpResponseWriter) Status() int {
	return w.status
}

func (w *fastHttpResponseWriter) SetStatus(status int) {
	w.status = status
}

func (w *fastHttpResponseWriter) Header() Header {
	return w.header
}

func (w *fastHttpResponseWriter) Succeed(v interface{}) {
	if v == nil {
		w.status = http.StatusOK
		return
	}
	body, encodeErr := json.Marshal(v)
	if encodeErr != nil {
		w.Failed(errors.Warning("fns: transport write succeed result failed").WithCause(encodeErr))
		return
	}

	w.status = http.StatusOK

	bodyLen := len(body)
	if bodyLen > 0 {
		w.Header().Set(contentLengthHeaderName, strconv.Itoa(bodyLen))
		w.Header().Set(contentTypeHeaderName, contentTypeJsonHeaderValue)
		w.write(body, bodyLen)
	}
	return
}

func (w *fastHttpResponseWriter) Failed(cause errors.CodeError) {
	if cause == nil {
		cause = errors.Warning("fns: error is lost")
	}
	body, encodeErr := json.Marshal(cause)
	if encodeErr != nil {
		body = []byte(`{"message": "fns: transport write failed result failed"}`)
		return
	}
	w.status = cause.Code()
	bodyLen := len(body)
	if bodyLen > 0 {
		w.Header().Set(contentLengthHeaderName, strconv.Itoa(bodyLen))
		w.Header().Set(contentTypeHeaderName, contentTypeJsonHeaderValue)
		w.write(body, bodyLen)
	}
	return
}

func (w *fastHttpResponseWriter) Write(body []byte) (int, error) {
	if body == nil {
		return 0, nil
	}
	bodyLen := len(body)
	w.write(body, bodyLen)
	return bodyLen, nil
}

func (w *fastHttpResponseWriter) Body() []byte {
	return w.body.Bytes()
}

func (w *fastHttpResponseWriter) write(body []byte, bodyLen int) {
	n := 0
	for n < bodyLen {
		nn, writeErr := w.body.Write(body[n:])
		if writeErr != nil {
			break
		}
		n += nn
	}
	return
}

func (w *fastHttpResponseWriter) Hijack(f func(conn net.Conn)) (err error) {
	if f == nil {
		err = errors.Warning("fns: hijack function is nil")
		return
	}
	w.ctx.Hijack(func(c net.Conn) {
		f(c)
	})
	return
}

func (w *fastHttpResponseWriter) Hijacked() bool {
	return w.ctx.Hijacked()
}
