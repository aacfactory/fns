package standard

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/fns/transports/ssl"
	"github.com/aacfactory/logs"
	"net"
	"net/http"
	"strings"
	"time"
)

func newServer(log logs.Logger, port int, tlsConfig ssl.Config, config *Config, handler transports.Handler) (srv *Server, err error) {
	var srvTLS *tls.Config
	var lnf ssl.ListenerFunc
	if tlsConfig != nil {
		srvTLS, lnf = tlsConfig.Server()
	}

	maxRequestHeaderSize := uint64(0)
	if config.MaxRequestHeaderSize != "" {
		maxRequestHeaderSize, err = bytex.ParseBytes(strings.TrimSpace(config.MaxRequestHeaderSize))
		if err != nil {
			err = errors.Warning("http: build server failed").WithCause(errors.Warning("maxRequestHeaderSize is invalid").WithCause(err).WithMeta("hit", "format must be bytes"))
			return
		}
	}
	maxRequestBodySize := uint64(0)
	if config.MaxRequestBodySize != "" {
		maxRequestBodySize, err = bytex.ParseBytes(strings.TrimSpace(config.MaxRequestBodySize))
		if err != nil {
			err = errors.Warning("http: build server failed").WithCause(errors.Warning("maxRequestBodySize is invalid").WithCause(err).WithMeta("hit", "format must be bytes"))
			return
		}
	}
	if maxRequestBodySize == 0 {
		maxRequestBodySize = 4 * bytex.MEGABYTE
	}
	readTimeout := 10 * time.Second
	if config.ReadTimeout != "" {
		readTimeout, err = time.ParseDuration(strings.TrimSpace(config.ReadTimeout))
		if err != nil {
			err = errors.Warning("http: build server failed").WithCause(errors.Warning("readTimeout is invalid").WithCause(err).WithMeta("hit", "format must time.Duration"))
			return
		}
	}
	readHeaderTimeout := 5 * time.Second
	if config.ReadHeaderTimeout != "" {
		readHeaderTimeout, err = time.ParseDuration(strings.TrimSpace(config.ReadHeaderTimeout))
		if err != nil {
			err = errors.Warning("http: build server failed").WithCause(errors.Warning("readHeaderTimeout is invalid").WithCause(err).WithMeta("hit", "format must time.Duration"))
			return
		}
	}
	writeTimeout := 30 * time.Second
	if config.WriteTimeout != "" {
		writeTimeout, err = time.ParseDuration(strings.TrimSpace(config.WriteTimeout))
		if err != nil {
			err = errors.Warning("http: build server failed").WithCause(errors.Warning("writeTimeout is invalid").WithCause(err).WithMeta("hit", "format must time.Duration"))
			return
		}
	}
	idleTimeout := 30 * time.Second
	if config.IdleTimeout != "" {
		idleTimeout, err = time.ParseDuration(strings.TrimSpace(config.IdleTimeout))
		if err != nil {
			err = errors.Warning("http: build server failed").WithCause(errors.Warning("idleTimeout is invalid").WithCause(err).WithMeta("hit", "format must time.Duration"))
			return
		}
	}

	server := &http.Server{
		Addr:                         fmt.Sprintf(":%d", port),
		Handler:                      HttpTransportHandlerAdaptor(handler, int(maxRequestBodySize)),
		DisableGeneralOptionsHandler: false,
		TLSConfig:                    srvTLS,
		ReadTimeout:                  readTimeout,
		ReadHeaderTimeout:            readHeaderTimeout,
		WriteTimeout:                 writeTimeout,
		IdleTimeout:                  idleTimeout,
		MaxHeaderBytes:               int(maxRequestHeaderSize),
		ErrorLog:                     logs.MapToLogger(log, logs.DebugLevel, false),
	}

	srv = &Server{
		port: port,
		lnf:  lnf,
		srv:  server,
	}

	return
}

type Server struct {
	port int
	lnf  ssl.ListenerFunc
	srv  *http.Server
}

func (srv *Server) ListenAndServe() (err error) {
	ln, lnErr := net.Listen("tcp", fmt.Sprintf(":%d", srv.port))
	if lnErr != nil {
		err = errors.Warning("fns: transport listen and serve failed").WithCause(lnErr)
		return
	}
	if srv.lnf != nil {
		ln = srv.lnf(ln)
	}
	if srv.srv.TLSConfig == nil {
		err = srv.srv.Serve(ln)
	} else {
		err = srv.srv.ServeTLS(ln, "", "")
	}
	if err != nil {
		err = errors.Warning("fns: transport listen and serve failed").WithCause(err).WithMeta("transport", transportName)
		return
	}
	return
}

func (srv *Server) Shutdown() (err error) {
	err = srv.srv.Shutdown(context.Background())
	if err != nil {
		err = errors.Warning("fns: transport shutdown failed").WithCause(err).WithMeta("transport", transportName)
	}
	return
}
