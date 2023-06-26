package fast

import (
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service/ssl"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/dgrr/http2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/prefork"
	"net"
	"strings"
	"time"
)

const (
	transportName = "fasthttp"
)

type Config struct {
	ReadBufferSize        string       `json:"readBufferSize"`
	ReadTimeout           string       `json:"readTimeout"`
	WriteBufferSize       string       `json:"writeBufferSize"`
	WriteTimeout          string       `json:"writeTimeout"`
	MaxIdleWorkerDuration string       `json:"maxIdleWorkerDuration"`
	TCPKeepalive          bool         `json:"tcpKeepalive"`
	TCPKeepalivePeriod    string       `json:"tcpKeepalivePeriod"`
	MaxRequestBodySize    string       `json:"maxRequestBodySize"`
	ReduceMemoryUsage     bool         `json:"reduceMemoryUsage"`
	MaxRequestsPerConn    int          `json:"maxRequestsPerConn"`
	KeepHijackedConns     bool         `json:"keepHijackedConns"`
	StreamRequestBody     bool         `json:"streamRequestBody"`
	Prefork               bool         `json:"prefork"`
	DisableHttp2          bool         `json:"disableHttp2"`
	Client                ClientConfig `json:"client"`
}

func New() transports.Transport {
	return &Transport{}
}

type Transport struct {
	log       logs.Logger
	lnf       ssl.ListenerFunc
	address   string
	preforked bool
	server    *fasthttp.Server
	dialer    *Dialer
}

func (tr *Transport) Name() (name string) {
	name = transportName
	return
}

func (tr *Transport) Build(options transports.Options) (err error) {
	tr.log = options.Log
	tr.address = fmt.Sprintf(":%d", options.Port)
	var srvTLS *tls.Config
	if options.TLS != nil {
		srvTLS, tr.lnf = options.TLS.Server()
	}
	config := &Config{}
	optErr := options.Config.As(config)
	if optErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(optErr).WithMeta("transport", transportName)
		return
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

	tr.preforked = config.Prefork

	tr.server = &fasthttp.Server{
		Handler:                            handlerAdaptor(options.Handler),
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
		Logger:                             logs.MapToLogger(options.Log, logs.DebugLevel, false),
		TLSConfig:                          srvTLS,
	}
	// http2
	if !config.DisableHttp2 && srvTLS != nil {
		http2.ConfigureServer(tr.server, http2.ServerConfig{})
	}

	// dialer
	clientConfig := &config.Client
	if options.TLS != nil {
		cliTLS, dialer := options.TLS.Client()
		clientConfig.TLSConfig = cliTLS
		clientConfig.DisableHttp2 = config.DisableHttp2
		if dialer != nil {
			clientConfig.TLSDialer = dialer
		}
	}
	dialer, dialerErr := NewDialer(clientConfig)
	if dialerErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(dialerErr)
		return
	}
	tr.dialer = dialer
	return
}

func (tr *Transport) Dial(address string) (client transports.Client, err error) {
	client, err = tr.dialer.Dial(address)
	return
}

func (tr *Transport) preforkServe(ln net.Listener) (err error) {
	if tr.lnf != nil {
		ln = tr.lnf(ln)
	}
	err = tr.server.Serve(ln)
	return
}

func (tr *Transport) ListenAndServe() (err error) {
	if tr.preforked {
		pf := prefork.New(tr.server)
		pf.ServeFunc = tr.preforkServe
		err = pf.ListenAndServe(tr.address)
		if err != nil {
			err = errors.Warning("fns: transport perfork listen and serve failed").WithCause(err)
			return
		}
		return
	}
	ln, lnErr := net.Listen("tcp", tr.address)
	if lnErr != nil {
		err = errors.Warning("fns: transport listen and serve failed").WithCause(lnErr)
		return
	}
	if tr.lnf != nil {
		ln = tr.lnf(ln)
	}
	err = tr.server.Serve(ln)
	if err != nil {
		err = errors.Warning("fns: transport listen and serve failed").WithCause(err).WithMeta("transport", transportName)
		return
	}
	return
}

func (tr *Transport) Close() (err error) {
	tr.dialer.Close()
	err = tr.server.Shutdown()
	if err != nil {
		err = errors.Warning("fns: transport close failed").WithCause(err).WithMeta("transport", transportName)
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func errorHandler(ctx *fasthttp.RequestCtx, err error) {
	ctx.SetStatusCode(555)
	ctx.SetContentType(transports.ContentTypeJsonHeaderValue)
	p, _ := json.Marshal(errors.Warning("fns: transport receiving or parsing the request failed").WithCause(err).WithMeta("transport", transportName))
	ctx.SetBody(p)
}
