package fast

import (
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
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

func newServer(log logs.Logger, port int, tlsConfig ssl.Config, config *Config, middlewares transports.Middlewares, handler transports.Handler) (srv *Server, err error) {
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

	if len(middlewares) > 0 {
		handler = middlewares.Handler(handler)
	}

	server := &fasthttp.Server{
		Handler:                            handlerAdaptor(handler),
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
		Logger:                             logs.MapToLogger(log, logs.DebugLevel, false),
		TLSConfig:                          srvTLS,
	}
	// http2
	if !config.DisableHttp2 && srvTLS != nil {
		http2.ConfigureServer(server, http2.ServerConfig{})
	}

	srv = &Server{
		port:        port,
		preFork:     config.Prefork,
		lnf:         lnf,
		srv:         server,
		middlewares: middlewares,
	}

	return
}

type Server struct {
	port        int
	preFork     bool
	lnf         ssl.ListenerFunc
	srv         *fasthttp.Server
	middlewares transports.Middlewares
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

func (srv *Server) Shutdown() (err error) {
	if len(srv.middlewares) > 0 {
		srv.middlewares.Close()
	}
	err = srv.srv.Shutdown()
	if err != nil {
		err = errors.Warning("fns: transport shutdown failed").WithCause(err).WithMeta("transport", transportName)
	}
	return
}
