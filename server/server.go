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

package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/fns/internal/configure"
	"github.com/aacfactory/fns/internal/logger"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	httpServerHeader      = "Server"
	httpServerHeaderValue = "FNS"
	httpContentType       = "Content-Type"
	httpContentTypeJson   = "application/json"
)

type HttpClient interface {
	Do(ctx context.Context, method string, url string, header http.Header, body []byte) (status int, respHeader http.Header, respBody []byte, err error)
	Close()
}

func NewHttpOptions(config *configure.Server, log logs.Logger, handler http.Handler) (opt HttpOptions, err error) {
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

type Http interface {
	Build(options HttpOptions) (err error)
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
	maxRequestBodySize, maxRequestBodySizeErr := commons.ToBytes(maxRequestBody)
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
		Logger:                             &logger.Printf{Core: options.Log},
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
