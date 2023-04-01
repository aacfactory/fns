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
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/valyala/fasthttp"
	"io"
	"net/http"
	"net/url"
	"strings"
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

func newTransportOptions(rt *Runtime, config *HttpConfig, log logs.Logger, handler http.Handler) (opt TransportOptions, err error) {
	log = log.With("fns", "transport")
	opt = TransportOptions{
		Port:      80,
		ServerTLS: nil,
		ClientTLS: nil,
		Handler:   handler,
		Log:       log,
		Runtime:   rt,
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

type TransportOptions struct {
	Port      int
	ServerTLS *tls.Config
	ClientTLS *tls.Config
	Handler   http.Handler
	Log       logs.Logger
	Runtime   *Runtime
	Options   configures.Config
}

type Transport interface {
	Name() (name string)
	Build(options TransportOptions) (err error)
	Dialer
	ListenAndServe() (err error)
	io.Closer
}

type Dialer interface {
	Dial(address string) (client Client, err error)
}

type Client interface {
	Get(ctx context.Context, path string, header http.Header) (status int, respHeader http.Header, respBody []byte, err error)
	Post(ctx context.Context, path string, header http.Header, body []byte) (status int, respHeader http.Header, respBody []byte, err error)
	Close()
}

// +-------------------------------------------------------------------------------------------------------------------+

type fastHttpTransportClientOptions struct {
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

type fastHttpTransportOptions struct {
	ReadBufferSize        string                         `json:"readBufferSize"`
	ReadTimeout           string                         `json:"readTimeout"`
	WriteBufferSize       string                         `json:"writeBufferSize"`
	WriteTimeout          string                         `json:"writeTimeout"`
	MaxIdleWorkerDuration string                         `json:"maxIdleWorkerDuration"`
	TCPKeepalive          bool                           `json:"tcpKeepalive"`
	TCPKeepalivePeriod    string                         `json:"tcpKeepalivePeriod"`
	MaxRequestBodySize    string                         `json:"maxRequestBodySize"`
	ReduceMemoryUsage     bool                           `json:"reduceMemoryUsage"`
	MaxRequestsPerConn    int                            `json:"maxRequestsPerConn"`
	KeepHijackedConns     bool                           `json:"keepHijackedConns"`
	StreamRequestBody     bool                           `json:"streamRequestBody"`
	Client                fastHttpTransportClientOptions `json:"client"`
}

type fastHttpTransportClient struct {
	ssl     bool
	address string
	tr      *fasthttp.Client
}

func (client *fastHttpTransportClient) Get(ctx context.Context, path string, header http.Header) (status int, respHeader http.Header, respBody []byte, err error) {
	req := client.prepareRequest(bytex.FromString(http.MethodGet), path, header)
	resp := fasthttp.AcquireResponse()
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		err = client.tr.DoDeadline(req, resp, deadline)
	} else {
		err = client.tr.Do(req, resp)
	}
	if err != nil {
		err = errors.Warning("fns: fasthttp client do get failed").WithCause(err).WithMeta("transport", fastHttpTransportName)
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

func (client *fastHttpTransportClient) Post(ctx context.Context, path string, header http.Header, body []byte) (status int, respHeader http.Header, respBody []byte, err error) {
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
		err = errors.Warning("fns: fasthttp client do post failed").WithCause(err).WithMeta("transport", fastHttpTransportName)
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

func (client *fastHttpTransportClient) prepareRequest(method []byte, path string, header http.Header) (req *fasthttp.Request) {
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
	return
}

func (client *fastHttpTransportClient) Close() {
	client.tr.CloseIdleConnections()
}

const (
	fastHttpTransportName = "fasthttp"
)

func FastHttpTransport() Transport {
	return &fastHttpTransport{}
}

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

func (srv *fastHttpTransport) Build(options TransportOptions) (err error) {
	srv.log = options.Log
	srv.address = fmt.Sprintf(":%d", options.Port)
	srv.ssl = options.ServerTLS != nil

	opt := &fastHttpTransportOptions{}
	optErr := options.Options.As(opt)
	if optErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(optErr).WithMeta("transport", fastHttpTransportName)
		return
	}
	readBufferSize := uint64(0)
	if opt.ReadBufferSize != "" {
		readBufferSize, err = bytex.ToBytes(strings.TrimSpace(opt.ReadBufferSize))
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
		writeBufferSize, err = bytex.ToBytes(strings.TrimSpace(opt.WriteBufferSize))
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
		maxRequestBodySize, err = bytex.ToBytes(strings.TrimSpace(opt.MaxRequestBodySize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("maxRequestBodySize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}

	reduceMemoryUsage := opt.ReduceMemoryUsage

	adaptor := &fastHttpTransportHandlerAdaptor{
		runtime: options.Runtime,
	}

	srv.server = &fasthttp.Server{
		Handler:                            adaptor.Handler(options.Handler),
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

func (srv *fastHttpTransport) buildClient(opt fastHttpTransportClientOptions, cliConfig *tls.Config) (err error) {
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
		readBufferSize, err = bytex.ToBytes(strings.TrimSpace(opt.ReadBufferSize))
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
		writeBufferSize, err = bytex.ToBytes(strings.TrimSpace(opt.WriteBufferSize))
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
		maxResponseBodySize, err = bytex.ToBytes(strings.TrimSpace(opt.MaxResponseBodySize))
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
	client = &fastHttpTransportClient{
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

func fastHttpErrorHandler(ctx *fasthttp.RequestCtx, err error) {
	ctx.SetStatusCode(555)
	ctx.SetContentType("application/json")
	p, _ := json.Marshal(errors.Warning("fns: fasthttp receiving or parsing the request failed").WithCause(err).WithMeta("transport", fastHttpTransportName))
	ctx.SetBody(p)
}

type fastHttpTransportHandlerAdaptor struct {
	runtime *Runtime
}

func (adaptor *fastHttpTransportHandlerAdaptor) Handler(h http.Handler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var r http.Request
		if err := convertFastHttpRequestToHttpRequest(ctx, &r, true); err != nil {
			p, _ := json.Marshal(errors.Warning("fns: cannot parse requestURI").WithMeta("uri", r.RequestURI).WithMeta("transport", fastHttpTransportName).WithCause(err))
			ctx.Response.Reset()
			ctx.SetStatusCode(555)
			ctx.SetContentTypeBytes(bytex.FromString(httpContentTypeJson))
			ctx.SetBody(p)
			return
		}

		w := netHTTPResponseWriter{w: ctx.Response.BodyWriter()}
		h.ServeHTTP(&w, r.WithContext(adaptor.runtime.SetIntoContext(ctx)))

		ctx.SetStatusCode(w.StatusCode())
		haveContentType := false
		for k, vv := range w.Header() {
			if k == fasthttp.HeaderContentType {
				haveContentType = true
			}

			for _, v := range vv {
				ctx.Response.Header.Add(k, v)
			}
		}
		if !haveContentType {
			l := 512
			b := ctx.Response.Body()
			if len(b) < 512 {
				l = len(b)
			}
			ctx.Response.Header.Set(fasthttp.HeaderContentType, http.DetectContentType(b[:l]))
		}
	}
}

func convertFastHttpRequestToHttpRequest(ctx *fasthttp.RequestCtx, r *http.Request, forServer bool) error {
	body := ctx.PostBody()
	strRequestURI := bytex.ToString(ctx.RequestURI())

	rURL, err := url.ParseRequestURI(strRequestURI)
	if err != nil {
		return err
	}

	r.Method = bytex.ToString(ctx.Method())
	r.Proto = bytex.ToString(ctx.Request.Header.Protocol())
	if r.Proto == "HTTP/2" {
		r.ProtoMajor = 2
	} else {
		r.ProtoMajor = 1
	}
	r.ProtoMinor = 1
	r.ContentLength = int64(len(body))
	r.RemoteAddr = ctx.RemoteAddr().String()
	r.Host = bytex.ToString(ctx.Host())
	r.TLS = ctx.TLSConnectionState()
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.URL = rURL

	if forServer {
		r.RequestURI = strRequestURI
	}

	if r.Header == nil {
		r.Header = make(http.Header)
	} else if len(r.Header) > 0 {
		for k := range r.Header {
			delete(r.Header, k)
		}
	}

	ctx.Request.Header.VisitAll(func(k, v []byte) {
		sk := bytex.ToString(k)
		sv := bytex.ToString(v)

		switch sk {
		case "Transfer-Encoding":
			r.TransferEncoding = append(r.TransferEncoding, sv)
		default:
			r.Header.Set(sk, sv)
		}
	})

	return nil
}

type netHTTPResponseWriter struct {
	statusCode int
	h          http.Header
	w          io.Writer
}

func (w *netHTTPResponseWriter) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

func (w *netHTTPResponseWriter) Header() http.Header {
	if w.h == nil {
		w.h = make(http.Header)
	}
	return w.h
}

func (w *netHTTPResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *netHTTPResponseWriter) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

func (w *netHTTPResponseWriter) Flush() {}
