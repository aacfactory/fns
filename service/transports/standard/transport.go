package standard

import (
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service/ssl"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/logs"
	"net/http"
	"strings"
	"time"
)

const (
	transportName = "http"
)

type Config struct {
	MaxRequestHeaderSize string        `json:"maxRequestHeaderSize"`
	MaxRequestBodySize   string        `json:"maxRequestBodySize"`
	ReadTimeout          string        `json:"readTimeout"`
	ReadHeaderTimeout    string        `json:"readHeaderTimeout"`
	WriteTimeout         string        `json:"writeTimeout"`
	IdleTimeout          string        `json:"idleTimeout"`
	Client               *ClientConfig `json:"client"`
}

func (config *Config) ClientConfig() *ClientConfig {
	if config.Client == nil {
		return &ClientConfig{}
	}
	return config.Client
}

func New() transports.Transport {
	return &Transport{}
}

type Transport struct {
	log    logs.Logger
	lnf    ssl.ListenerFunc
	srv    *http.Server
	dialer *Dialer
}

func (tr *Transport) Name() (name string) {
	name = transportName
	return
}

func (tr *Transport) Build(options transports.Options) (err error) {
	tr.log = options.Log

	var srvTLS *tls.Config
	if options.TLS != nil {
		srvTLS, tr.lnf = options.TLS.Server()
	}

	config := Config{}
	decodeErr := options.Config.As(&config)
	if decodeErr != nil {
		err = errors.Warning("http: build failed").WithCause(decodeErr)
		return
	}
	maxRequestHeaderSize := uint64(0)
	if config.MaxRequestHeaderSize != "" {
		maxRequestHeaderSize, err = bytex.ParseBytes(strings.TrimSpace(config.MaxRequestHeaderSize))
		if err != nil {
			err = errors.Warning("http: build failed").WithCause(errors.Warning("maxRequestHeaderSize is invalid").WithCause(err).WithMeta("hit", "format must be bytes"))
			return
		}
	}
	maxRequestBodySize := uint64(0)
	if config.MaxRequestBodySize != "" {
		maxRequestBodySize, err = bytex.ParseBytes(strings.TrimSpace(config.MaxRequestBodySize))
		if err != nil {
			err = errors.Warning("http: build failed").WithCause(errors.Warning("maxRequestBodySize is invalid").WithCause(err).WithMeta("hit", "format must be bytes"))
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
			err = errors.Warning("http: build failed").WithCause(errors.Warning("readTimeout is invalid").WithCause(err).WithMeta("hit", "format must time.Duration"))
			return
		}
	}
	readHeaderTimeout := 5 * time.Second
	if config.ReadHeaderTimeout != "" {
		readHeaderTimeout, err = time.ParseDuration(strings.TrimSpace(config.ReadHeaderTimeout))
		if err != nil {
			err = errors.Warning("http: build failed").WithCause(errors.Warning("readHeaderTimeout is invalid").WithCause(err).WithMeta("hit", "format must time.Duration"))
			return
		}
	}
	writeTimeout := 30 * time.Second
	if config.WriteTimeout != "" {
		writeTimeout, err = time.ParseDuration(strings.TrimSpace(config.WriteTimeout))
		if err != nil {
			err = errors.Warning("http: build failed").WithCause(errors.Warning("writeTimeout is invalid").WithCause(err).WithMeta("hit", "format must time.Duration"))
			return
		}
	}
	idleTimeout := 30 * time.Second
	if config.IdleTimeout != "" {
		idleTimeout, err = time.ParseDuration(strings.TrimSpace(config.IdleTimeout))
		if err != nil {
			err = errors.Warning("http: build failed").WithCause(errors.Warning("idleTimeout is invalid").WithCause(err).WithMeta("hit", "format must time.Duration"))
			return
		}
	}
	// Transport
	handler := HttpTransportHandlerAdaptor(options.Handler, int(maxRequestBodySize))
	tr.srv = &http.Server{
		Addr:                         fmt.Sprintf(":%d", options.Port),
		Handler:                      handler,
		DisableGeneralOptionsHandler: false,
		TLSConfig:                    srvTLS,
		ReadTimeout:                  readTimeout,
		ReadHeaderTimeout:            readHeaderTimeout,
		WriteTimeout:                 writeTimeout,
		IdleTimeout:                  idleTimeout,
		MaxHeaderBytes:               int(maxRequestHeaderSize),
		ErrorLog:                     logs.MapToLogger(tr.log, logs.DebugLevel, false),
	}
	// dialer
	clientConfig := config.ClientConfig()
	if options.TLS != nil {
		cliTLS, dialer := options.TLS.Client()
		clientConfig.TLSConfig = cliTLS
		if dialer != nil {
			clientConfig.TLSDialer = dialer
		}
	}
	dialer, dialerErr := NewDialer(clientConfig)
	if dialerErr != nil {
		err = errors.Warning("http: build failed").WithCause(dialerErr)
		return
	}
	tr.dialer = dialer
	return
}

func (tr *Transport) Dial(address string) (client transports.Client, err error) {
	client, err = tr.dialer.Dial(address)
	return
}

func (tr *Transport) ListenAndServe() (err error) {
	if tr.srv.TLSConfig == nil {
		err = tr.srv.ListenAndServe()
	} else {
		err = tr.srv.ListenAndServeTLS("", "")
	}
	if err != nil {
		err = errors.Warning("http: listen and serve failed").WithCause(err)
		return
	}
	return
}

func (tr *Transport) Close() (err error) {
	tr.dialer.Close()
	err = tr.srv.Close()
	if err != nil {
		err = errors.Warning("http: close failed").WithCause(err)
	}
	return
}
