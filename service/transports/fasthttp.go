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
	"bufio"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service/ssl"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/dgrr/http2"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/prefork"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	fastHttpTransportName = "fasthttp"
)

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
	Prefork               bool                  `json:"prefork"`
	DisableHttp2          bool                  `json:"disableHttp2"`
	Client                FastHttpClientOptions `json:"client"`
}

// +-------------------------------------------------------------------------------------------------------------------+

// +-------------------------------------------------------------------------------------------------------------------+

type fastHttpTransport struct {
	log       logs.Logger
	sslConfig ssl.Config
	address   string
	preforked bool
	server    *fasthttp.Server
	dialer    Dialer
}

func (srv *fastHttpTransport) Name() (name string) {
	name = fastHttpTransportName
	return
}

func (srv *fastHttpTransport) Build(options Options) (err error) {
	srv.log = options.Log
	srv.address = fmt.Sprintf(":%d", options.Port)
	srv.sslConfig = options.TLS
	srvTLS, cliTLS, tlsErr := srv.sslConfig.TLS()
	if tlsErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(tlsErr).WithMeta("transport", fastHttpTransportName)
		return
	}

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

	srv.preforked = opt.Prefork

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
		TLSConfig:                          srvTLS,
		FormValueFunc:                      nil,
	}
	// http2
	if !opt.DisableHttp2 && srvTLS != nil {
		http2.ConfigureServer(srv.server, http2.ServerConfig{})
	}

	// dialer
	clientOPTS := &opt.Client
	clientOPTS.TLSConfig = cliTLS
	clientOPTS.DisableHttp2 = opt.DisableHttp2
	dialer, dialerErr := NewDialer(clientOPTS)
	if dialerErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(dialerErr)
		return
	}
	srv.dialer = dialer
	return
}

func (srv *fastHttpTransport) Dial(address string) (client Client, err error) {
	client, err = srv.dialer.Dial(address)
	return
}

func (srv *fastHttpTransport) preforkServe(ln net.Listener) (err error) {
	if srv.sslConfig != nil {
		err = srv.server.Serve(srv.sslConfig.NewListener(ln))
	} else {
		err = srv.server.Serve(ln)
	}
	return
}

func (srv *fastHttpTransport) ListenAndServe() (err error) {
	if srv.preforked {
		pf := prefork.New(srv.server)
		pf.ServeFunc = srv.preforkServe
		err = pf.ListenAndServe(srv.address)
		if err != nil {
			err = errors.Warning("fns: transport perfork listen and serve failed").WithCause(err)
			return
		}
		return
	}
	if srv.sslConfig != nil {
		ln, lnErr := net.Listen("tcp4", srv.address)
		if lnErr != nil {
			err = errors.Warning("fns: transport listen and serve failed").WithCause(lnErr).WithMeta("transport", fastHttpTransportName)
			return
		}
		err = srv.server.Serve(srv.sslConfig.NewListener(ln))
	} else {
		err = srv.server.ListenAndServe(srv.address)
	}
	if err != nil {
		err = errors.Warning("fns: transport listen and serve failed").WithCause(err).WithMeta("transport", fastHttpTransportName)
		return
	}
	return
}

func (srv *fastHttpTransport) Close() (err error) {
	err = srv.server.Shutdown()
	if err != nil {
		err = errors.Warning("fns: transport close failed").WithCause(err).WithMeta("transport", fastHttpTransportName)
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

	r.remoteAddr = bytex.FromString(ctx.RemoteAddr().String())

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

func (w *fastHttpResponseWriter) Hijack(f func(conn net.Conn, rw *bufio.ReadWriter) (err error)) (async bool, err error) {
	if f == nil {
		err = errors.Warning("fns: hijack function is nil")
		return
	}
	w.ctx.Hijack(func(c net.Conn) {
		_ = f(c, nil)
	})
	async = true
	return
}

func (w *fastHttpResponseWriter) Hijacked() bool {
	return w.ctx.Hijacked()
}
