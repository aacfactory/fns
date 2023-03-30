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
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"net/http"
	"strings"
	"time"
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
		err = errors.Warning("fns: fasthttp client do get failed").WithCause(err).WithMeta("transport", fastHttpName)
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
		err = errors.Warning("fns: fasthttp client do post failed").WithCause(err).WithMeta("transport", fastHttpName)
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
	return
}

func (client *FastHttpClient) Close() {
	client.tr.CloseIdleConnections()
}

const (
	fastHttpName = "fasthttp"
)

type FastHttp struct {
	log     logs.Logger
	ssl     bool
	address string
	client  *fasthttp.Client
	server  *fasthttp.Server
}

func (srv *FastHttp) Name() (name string) {
	name = fastHttpName
	return
}

func (srv *FastHttp) Build(options Options) (err error) {
	srv.log = options.Log
	srv.address = fmt.Sprintf(":%d", options.Port)
	srv.ssl = options.ServerTLS != nil

	opt := &FastHttpOptions{}
	optErr := options.Options.As(opt)
	if optErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(optErr).WithMeta("transport", fastHttpName)
		return
	}
	readBufferSize := uint64(0)
	if opt.ReadBufferSize != "" {
		readBufferSize, err = bytex.ToBytes(strings.TrimSpace(opt.ReadBufferSize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("readBufferSize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}
	readTimeout := 10 * time.Second
	if opt.ReadTimeout != "" {
		readTimeout, err = time.ParseDuration(strings.TrimSpace(opt.ReadTimeout))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("readTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}
	writeBufferSize := uint64(0)
	if opt.WriteBufferSize != "" {
		writeBufferSize, err = bytex.ToBytes(strings.TrimSpace(opt.WriteBufferSize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("writeBufferSize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}
	writeTimeout := 10 * time.Second
	if opt.WriteTimeout != "" {
		writeTimeout, err = time.ParseDuration(strings.TrimSpace(opt.WriteTimeout))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("writeTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}
	maxIdleWorkerDuration := time.Duration(0)
	if opt.MaxIdleWorkerDuration != "" {
		maxIdleWorkerDuration, err = time.ParseDuration(strings.TrimSpace(opt.MaxIdleWorkerDuration))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("maxIdleWorkerDuration must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}
	tcpKeepalivePeriod := time.Duration(0)
	if opt.TCPKeepalivePeriod != "" {
		tcpKeepalivePeriod, err = time.ParseDuration(strings.TrimSpace(opt.TCPKeepalivePeriod))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("tcpKeepalivePeriod must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}

	maxRequestBodySize := uint64(4 * bytex.MEGABYTE)
	if opt.MaxRequestBodySize != "" {
		maxRequestBodySize, err = bytex.ToBytes(strings.TrimSpace(opt.MaxRequestBodySize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("maxRequestBodySize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}

	reduceMemoryUsage := opt.ReduceMemoryUsage

	srv.server = &fasthttp.Server{
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
	if err != nil {
		err = errors.Warning("fns: fasthttp build failed").WithCause(err).WithMeta("transport", fastHttpName)
		return
	}
	return
}

func (srv *FastHttp) buildClient(opt FastHttpClientOptions, cliConfig *tls.Config) (err error) {
	maxIdleWorkerDuration := time.Duration(0)
	if opt.MaxIdleConnDuration != "" {
		maxIdleWorkerDuration, err = time.ParseDuration(strings.TrimSpace(opt.MaxIdleConnDuration))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxIdleWorkerDuration must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}
	maxConnDuration := time.Duration(0)
	if opt.MaxConnDuration != "" {
		maxConnDuration, err = time.ParseDuration(strings.TrimSpace(opt.MaxConnDuration))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxConnDuration must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}
	readBufferSize := uint64(0)
	if opt.ReadBufferSize != "" {
		readBufferSize, err = bytex.ToBytes(strings.TrimSpace(opt.ReadBufferSize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("readBufferSize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}
	readTimeout := 10 * time.Second
	if opt.ReadTimeout != "" {
		readTimeout, err = time.ParseDuration(strings.TrimSpace(opt.ReadTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("readTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}
	writeBufferSize := uint64(0)
	if opt.WriteBufferSize != "" {
		writeBufferSize, err = bytex.ToBytes(strings.TrimSpace(opt.WriteBufferSize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("writeBufferSize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}
	writeTimeout := 10 * time.Second
	if opt.WriteTimeout != "" {
		writeTimeout, err = time.ParseDuration(strings.TrimSpace(opt.WriteTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("writeTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}
	maxResponseBodySize := uint64(4 * bytex.MEGABYTE)
	if opt.MaxResponseBodySize != "" {
		maxResponseBodySize, err = bytex.ToBytes(strings.TrimSpace(opt.MaxResponseBodySize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxResponseBodySize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpName)
			return
		}
	}
	maxConnWaitTimeout := time.Duration(0)
	if opt.MaxConnWaitTimeout != "" {
		maxConnWaitTimeout, err = time.ParseDuration(strings.TrimSpace(opt.MaxConnWaitTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxConnWaitTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpName)
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

func (srv *FastHttp) Dial(address string) (client Client, err error) {
	client = &FastHttpClient{
		ssl:     srv.ssl,
		address: address,
		tr:      srv.client,
	}
	return
}

func (srv *FastHttp) ListenAndServe() (err error) {
	if srv.ssl {
		err = srv.server.ListenAndServeTLS(srv.address, "", "")
	} else {
		err = srv.server.ListenAndServe(srv.address)
	}
	if err != nil {
		err = errors.Warning("fns: server listen and serve failed").WithCause(err).WithMeta("transport", fastHttpName)
		return
	}
	return
}

func (srv *FastHttp) Close() (err error) {
	err = srv.server.Shutdown()
	if err != nil {
		err = errors.Warning("fns: server close failed").WithCause(err).WithMeta("transport", fastHttpName)
	}
	return
}

func fastHttpErrorHandler(ctx *fasthttp.RequestCtx, err error) {
	ctx.SetStatusCode(555)
	ctx.SetContentType("application/json")
	p, _ := json.Marshal(errors.Warning("fns: fasthttp receiving or parsing the request failed").WithCause(err).WithMeta("transport", fastHttpName))
	ctx.SetBody(p)
}
