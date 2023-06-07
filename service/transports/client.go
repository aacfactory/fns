package transports

import (
	"context"
	"crypto/tls"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/dgrr/http2"
	"github.com/valyala/fasthttp"
	"net"
	"strings"
	"time"
)

type Client interface {
	Do(ctx context.Context, request *Request) (response *Response, err error)
	Close()
}

type DialFunc func(ctx context.Context, network string, addr string) (conn net.Conn, err error)

type FastHttpClientOptions struct {
	DialDualStack             bool        `json:"dialDualStack"`
	MaxConnsPerHost           int         `json:"maxConnsPerHost"`
	MaxIdleConnDuration       string      `json:"maxIdleConnDuration"`
	MaxConnDuration           string      `json:"maxConnDuration"`
	MaxIdemponentCallAttempts int         `json:"maxIdemponentCallAttempts"`
	ReadBufferSize            string      `json:"readBufferSize"`
	ReadTimeout               string      `json:"readTimeout"`
	WriteBufferSize           string      `json:"writeBufferSize"`
	WriteTimeout              string      `json:"writeTimeout"`
	MaxResponseBodySize       string      `json:"maxResponseBodySize"`
	MaxConnWaitTimeout        string      `json:"maxConnWaitTimeout"`
	IsTLS                     bool        `json:"isTLS"`
	DisableHttp2              bool        `json:"disableHttp2"`
	TLSConfig                 *tls.Config `json:"-"`
	Dial                      DialFunc    `json:"-"`
}

func NewFastClient(address string, opts *FastHttpClientOptions) (client Client, err error) {
	maxIdleConnDuration := time.Duration(0)
	if opts.MaxIdleConnDuration != "" {
		maxIdleConnDuration, err = time.ParseDuration(strings.TrimSpace(opts.MaxIdleConnDuration))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxIdleWorkerDuration must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	maxConnDuration := time.Duration(0)
	if opts.MaxConnDuration != "" {
		maxConnDuration, err = time.ParseDuration(strings.TrimSpace(opts.MaxConnDuration))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxConnDuration must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	readBufferSize := uint64(0)
	if opts.ReadBufferSize != "" {
		readBufferSize, err = bytex.ParseBytes(strings.TrimSpace(opts.ReadBufferSize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("readBufferSize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	readTimeout := 10 * time.Second
	if opts.ReadTimeout != "" {
		readTimeout, err = time.ParseDuration(strings.TrimSpace(opts.ReadTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("readTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	writeBufferSize := uint64(0)
	if opts.WriteBufferSize != "" {
		writeBufferSize, err = bytex.ParseBytes(strings.TrimSpace(opts.WriteBufferSize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("writeBufferSize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	writeTimeout := 10 * time.Second
	if opts.WriteTimeout != "" {
		writeTimeout, err = time.ParseDuration(strings.TrimSpace(opts.WriteTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("writeTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	maxResponseBodySize := uint64(4 * bytex.MEGABYTE)
	if opts.MaxResponseBodySize != "" {
		maxResponseBodySize, err = bytex.ParseBytes(strings.TrimSpace(opts.MaxResponseBodySize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxResponseBodySize must be bytes format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}
	maxConnWaitTimeout := time.Duration(0)
	if opts.MaxConnWaitTimeout != "" {
		maxConnWaitTimeout, err = time.ParseDuration(strings.TrimSpace(opts.MaxConnWaitTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxConnWaitTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", fastHttpTransportName)
			return
		}
	}

	isTLS := opts.IsTLS
	if !isTLS {
		isTLS = opts.TLSConfig != nil
	}
	var dialFunc fasthttp.DialFunc
	if opts.Dial != nil {
		dialFunc = func(addr string) (net.Conn, error) {
			return opts.Dial(context.Background(), "tcp", addr)
		}
	}

	hc := &fasthttp.HostClient{
		Addr:                          address,
		Name:                          "",
		NoDefaultUserAgentHeader:      true,
		IsTLS:                         isTLS,
		TLSConfig:                     opts.TLSConfig,
		Dial:                          dialFunc,
		MaxConns:                      opts.MaxConnsPerHost,
		MaxConnDuration:               maxConnDuration,
		MaxIdleConnDuration:           maxIdleConnDuration,
		MaxIdemponentCallAttempts:     opts.MaxIdemponentCallAttempts,
		ReadBufferSize:                int(readBufferSize),
		WriteBufferSize:               int(writeBufferSize),
		ReadTimeout:                   readTimeout,
		WriteTimeout:                  writeTimeout,
		MaxResponseBodySize:           int(maxResponseBodySize),
		DisableHeaderNamesNormalizing: false,
		DisablePathNormalizing:        false,
		SecureErrorLogMessage:         false,
		MaxConnWaitTimeout:            maxConnWaitTimeout,
		RetryIf:                       nil,
		Transport:                     nil,
		ConnPoolStrategy:              fasthttp.FIFO,
	}
	if !opts.DisableHttp2 && isTLS {
		configErr := http2.ConfigureClient(hc, http2.ClientOpts{
			PingInterval:    0,
			MaxResponseTime: 10 * time.Second,
			OnRTT:           nil,
		})
		if configErr != nil {
			err = errors.Warning("fns: build client failed").WithCause(configErr)
			return
		}
	}
	client = &fastClient{
		address: address,
		secured: isTLS,
		core:    hc,
	}
	return
}

type fastClient struct {
	address string
	secured bool
	core    *fasthttp.HostClient
}

func (client *fastClient) Do(ctx context.Context, request *Request) (response *Response, err error) {
	req := fasthttp.AcquireRequest()
	// method
	req.Header.SetMethodBytes(request.method)
	// header
	if request.header != nil && len(request.header) > 0 {
		for k, vv := range request.header {
			if vv == nil || len(vv) == 0 {
				continue
			}
			for _, v := range vv {
				req.Header.Add(k, v)
			}
		}
	}
	// uri
	uri := req.URI()
	if client.secured {
		uri.SetSchemeBytes(bytex.FromString("https"))
	} else {
		uri.SetSchemeBytes(bytex.FromString("http"))
	}
	uri.SetHostBytes(bytex.FromString(client.address))
	uri.SetPathBytes(request.path)
	if request.params != nil && len(request.params) > 0 {
		uri.SetQueryStringBytes(bytex.FromString(request.params.String()))
	}
	// body
	if request.body != nil && len(request.body) > 0 {
		req.SetBodyRaw(request.body)
	}
	// resp
	resp := fasthttp.AcquireResponse()
	// do
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		err = client.core.DoDeadline(req, resp, deadline)
	} else {
		err = client.core.Do(req, resp)
	}
	if err != nil {
		err = errors.Warning("fns: transport client do failed").
			WithCause(err).
			WithMeta("transport", fastHttpTransportName).WithMeta("method", bytex.ToString(request.method)).WithMeta("path", bytex.ToString(request.path))
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
		return
	}
	response = &Response{
		Status: resp.StatusCode(),
		Header: make(Header),
		Body:   resp.Body(),
	}
	resp.Header.VisitAll(func(key, value []byte) {
		response.Header.Add(bytex.ToString(key), bytex.ToString(value))
	})
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)
	return
}

func (client *fastClient) Close() {
	client.core.CloseIdleConnections()
}
