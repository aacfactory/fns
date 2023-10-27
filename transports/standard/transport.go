package standard

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/transports"
)

const (
	transportName = "standard"
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
	server *Server
	dialer *Dialer
}

func (tr *Transport) Name() (name string) {
	name = transportName
	return
}

func (tr *Transport) Construct(options transports.Options) (err error) {
	// log
	log := options.Log.With("transport", transportName)
	// tls
	tlsConfig, tlsConfigErr := options.Config.TLS()
	if tlsConfig != nil {
		err = errors.Warning("fns: fast transport build failed").WithCause(tlsConfigErr).WithMeta("transport", transportName)
		return
	}
	// middlewares
	middlewares, middlewaresErr := transports.WaveMiddlewares(options.Log, options.Config, options.Middlewares)
	if middlewaresErr != nil {
		err = errors.Warning("fns: fast transport build failed").WithCause(middlewaresErr).WithMeta("transport", transportName)
		return
	}
	// handler
	if options.Handler == nil {
		err = errors.Warning("fns: fast transport build failed").WithCause(fmt.Errorf("handler is nil")).WithMeta("transport", transportName)
		return
	}

	// port
	port, portErr := options.Config.Port()
	if portErr != nil {
		err = errors.Warning("fns: fast transport build failed").WithCause(portErr).WithMeta("transport", transportName)
		return
	}
	// config
	optConfig, optConfigErr := options.Config.Options()
	if optConfigErr != nil {
		err = errors.Warning("fns: build transport failed").WithCause(optConfigErr).WithMeta("transport", transportName)
		return
	}
	config := &Config{}
	configErr := optConfig.As(config)
	if configErr != nil {
		err = errors.Warning("fns: build transport failed").WithCause(configErr).WithMeta("transport", transportName)
		return
	}
	// server
	srv, srvErr := newServer(log, port, tlsConfig, config, middlewares, options.Handler)
	if srvErr != nil {
		err = errors.Warning("fns: build transport failed").WithCause(srvErr).WithMeta("transport", transportName)
		return
	}
	tr.server = srv

	// dialer
	clientConfig := config.ClientConfig()
	if tlsConfig != nil {
		cliTLS, dialer := tlsConfig.Client()
		clientConfig.TLSConfig = cliTLS
		if dialer != nil {
			clientConfig.TLSDialer = dialer
		}
	}
	dialer, dialerErr := NewDialer(clientConfig)
	if dialerErr != nil {
		err = errors.Warning("http: build transport failed").WithCause(dialerErr)
		return
	}
	tr.dialer = dialer
	return
}

func (tr *Transport) Dial(address string) (client transports.Client, err error) {
	client, err = tr.dialer.Dial(address)
	return
}

func (tr *Transport) Port() (port int) {
	port = tr.server.port
	return
}

func (tr *Transport) ListenAndServe() (err error) {
	err = tr.server.ListenAndServe()
	return
}

func (tr *Transport) Shutdown() (err error) {
	tr.dialer.Close()
	err = tr.server.Shutdown()
	return
}
