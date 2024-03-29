/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package fast

import (
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/fns/transports/ssl"
	"github.com/aacfactory/logs"
	"github.com/dgrr/http2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/prefork"
	"net"
	"strings"
	"time"
)

func newServer(log logs.Logger, port int, tlsConfig ssl.Config, config *Config, handler transports.Handler) (srv *Server, err error) {
	var srvTLS *tls.Config
	var lnf ssl.ListenerFunc
	if tlsConfig != nil {
		srvTLS, lnf = tlsConfig.Server()
	}

	readBufferSize := uint64(0)
	if config.ReadBufferSize != "" {
		readBufferSize, err = bytex.ParseBytes(strings.TrimSpace(config.ReadBufferSize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("readBufferSize must be bytes format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}
	readTimeout := 10 * time.Second
	if config.ReadTimeout != "" {
		readTimeout, err = time.ParseDuration(strings.TrimSpace(config.ReadTimeout))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("readTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}
	writeBufferSize := uint64(0)
	if config.WriteBufferSize != "" {
		writeBufferSize, err = bytex.ParseBytes(strings.TrimSpace(config.WriteBufferSize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("writeBufferSize must be bytes format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}
	writeTimeout := 10 * time.Second
	if config.WriteTimeout != "" {
		writeTimeout, err = time.ParseDuration(strings.TrimSpace(config.WriteTimeout))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("writeTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}
	maxIdleWorkerDuration := time.Duration(0)
	if config.MaxIdleWorkerDuration != "" {
		maxIdleWorkerDuration, err = time.ParseDuration(strings.TrimSpace(config.MaxIdleWorkerDuration))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("maxIdleWorkerDuration must be time.Duration format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}
	tcpKeepalivePeriod := time.Duration(0)
	if config.TCPKeepalivePeriod != "" {
		tcpKeepalivePeriod, err = time.ParseDuration(strings.TrimSpace(config.TCPKeepalivePeriod))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("tcpKeepalivePeriod must be time.Duration format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}

	maxRequestBodySize := uint64(4 * bytex.MEGABYTE)
	if config.MaxRequestBodySize != "" {
		maxRequestBodySize, err = bytex.ParseBytes(strings.TrimSpace(config.MaxRequestBodySize))
		if err != nil {
			err = errors.Warning("fns: build server failed").WithCause(errors.Warning("maxRequestBodySize must be bytes format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}

	reduceMemoryUsage := config.ReduceMemoryUsage

	server := &fasthttp.Server{
		Handler:                            handlerAdaptor(handler, writeTimeout),
		ErrorHandler:                       errorHandler,
		Name:                               "",
		Concurrency:                        0,
		ReadBufferSize:                     int(readBufferSize),
		WriteBufferSize:                    int(writeBufferSize),
		ReadTimeout:                        readTimeout,
		WriteTimeout:                       writeTimeout,
		MaxRequestsPerConn:                 config.MaxRequestsPerConn,
		MaxIdleWorkerDuration:              maxIdleWorkerDuration,
		TCPKeepalivePeriod:                 tcpKeepalivePeriod,
		MaxRequestBodySize:                 int(maxRequestBodySize),
		DisableKeepalive:                   false,
		TCPKeepalive:                       config.TCPKeepalive,
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
		KeepHijackedConns:                  config.KeepHijackedConns,
		CloseOnShutdown:                    true,
		StreamRequestBody:                  config.StreamRequestBody,
		ConnState:                          nil,
		Logger:                             logs.ConvertToStandardLogger(log, logs.DebugLevel, false),
		TLSConfig:                          srvTLS,
	}
	// http2
	if config.Http2.Enable && srvTLS != nil {
		http2.ConfigureServer(server, http2.ServerConfig{
			PingInterval:         time.Duration(config.Http2.PingSeconds) * time.Second,
			MaxConcurrentStreams: config.Http2.MaxConcurrentStreams,
			Debug:                false,
		})
	}

	srv = &Server{
		port:    port,
		preFork: config.Prefork,
		lnf:     lnf,
		srv:     server,
	}
	return
}

type Server struct {
	port    int
	preFork bool
	lnf     ssl.ListenerFunc
	srv     *fasthttp.Server
}

func (srv *Server) preforkServe(ln net.Listener) (err error) {
	if srv.lnf != nil {
		ln = srv.lnf(ln)
	}
	err = srv.srv.Serve(ln)
	return
}

func (srv *Server) ListenAndServe() (err error) {
	if srv.preFork {
		pf := prefork.New(srv.srv)
		pf.ServeFunc = srv.preforkServe
		err = pf.ListenAndServe(fmt.Sprintf(":%d", srv.port))
		if err != nil {
			err = errors.Warning("fns: transport perfork listen and serve failed").WithCause(err)
			return
		}
		return
	}
	ln, lnErr := net.Listen("tcp", fmt.Sprintf(":%d", srv.port))
	if lnErr != nil {
		err = errors.Warning("fns: transport listen and serve failed").WithCause(lnErr)
		return
	}
	if srv.lnf != nil {
		ln = srv.lnf(ln)
	}
	err = srv.srv.Serve(ln)
	if err != nil {
		err = errors.Warning("fns: transport listen and serve failed").WithCause(err).WithMeta("transport", transportName)
		return
	}
	return
}

func (srv *Server) Shutdown(ctx context.Context) (err error) {
	err = srv.srv.ShutdownWithContext(ctx)
	if err != nil {
		err = errors.Warning("fns: transport shutdown failed").WithCause(err).WithMeta("transport", transportName)
	}
	return
}
