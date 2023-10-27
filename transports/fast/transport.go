package fast

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/valyala/fasthttp"
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
	server      *Server
	dialer      *Dialer
	middlewares transports.Middlewares
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
	middleware, middlewareErr := transports.WaveMiddlewares(options.Log, options.Config, options.Middlewares)
	if middlewareErr != nil {
		err = errors.Warning("fns: fast transport build failed").WithCause(middlewareErr).WithMeta("transport", transportName)
		return
	}
	tr.middlewares = middleware
	// handler
	if options.Handler == nil {
		err = errors.Warning("fns: fast transport build failed").WithCause(fmt.Errorf("handler is nil")).WithMeta("transport", transportName)
		return
	}
	handler := middleware.Handler(options.Handler)

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
	srv, srvErr := newServer(log, port, tlsConfig, config, handler)
	if srvErr != nil {
		err = errors.Warning("fns: build transport failed").WithCause(srvErr).WithMeta("transport", transportName)
		return
	}
	tr.server = srv

	// dialer
	clientConfig := &config.Client
	if tlsConfig != nil {
		cliTLS, dialer := tlsConfig.Client()
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

func (tr *Transport) Port() (port int) {
	port = tr.server.port
	return
}

func (tr *Transport) ListenAndServe() (err error) {
	err = tr.server.ListenAndServe()
	return
}

func (tr *Transport) Shutdown() (err error) {
	tr.middlewares.Close()
	tr.dialer.Close()
	err = tr.server.Shutdown()
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func errorHandler(ctx *fasthttp.RequestCtx, err error) {
	ctx.SetStatusCode(555)
	ctx.SetContentType(transports.ContentTypeJsonHeaderValue)
	p, _ := json.Marshal(errors.Warning("fns: transport receiving or parsing the request failed").WithCause(err).WithMeta("transport", transportName))
	ctx.SetBody(p)
}
