package fast

import (
	"context"
	"crypto/tls"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/fns/transports/ssl"
	"github.com/dgrr/http2"
	"github.com/valyala/fasthttp"
	"net"
	"strings"
	"time"
)

type ClientConfig struct {
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
	TLSDialer                 ssl.Dialer  `json:"-"`
}

func NewClient(address string, config *ClientConfig) (client *Client, err error) {
	maxIdleConnDuration := time.Duration(0)
	if config.MaxIdleConnDuration != "" {
		maxIdleConnDuration, err = time.ParseDuration(strings.TrimSpace(config.MaxIdleConnDuration))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxIdleWorkerDuration must be time.Duration format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}
	maxConnDuration := time.Duration(0)
	if config.MaxConnDuration != "" {
		maxConnDuration, err = time.ParseDuration(strings.TrimSpace(config.MaxConnDuration))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxConnDuration must be time.Duration format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}
	readBufferSize := uint64(0)
	if config.ReadBufferSize != "" {
		readBufferSize, err = bytex.ParseBytes(strings.TrimSpace(config.ReadBufferSize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("readBufferSize must be bytes format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}
	readTimeout := 10 * time.Second
	if config.ReadTimeout != "" {
		readTimeout, err = time.ParseDuration(strings.TrimSpace(config.ReadTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("readTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}
	writeBufferSize := uint64(0)
	if config.WriteBufferSize != "" {
		writeBufferSize, err = bytex.ParseBytes(strings.TrimSpace(config.WriteBufferSize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("writeBufferSize must be bytes format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}
	writeTimeout := 10 * time.Second
	if config.WriteTimeout != "" {
		writeTimeout, err = time.ParseDuration(strings.TrimSpace(config.WriteTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("writeTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}
	maxResponseBodySize := uint64(4 * bytex.MEGABYTE)
	if config.MaxResponseBodySize != "" {
		maxResponseBodySize, err = bytex.ParseBytes(strings.TrimSpace(config.MaxResponseBodySize))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxResponseBodySize must be bytes format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}
	maxConnWaitTimeout := time.Duration(0)
	if config.MaxConnWaitTimeout != "" {
		maxConnWaitTimeout, err = time.ParseDuration(strings.TrimSpace(config.MaxConnWaitTimeout))
		if err != nil {
			err = errors.Warning("fns: build client failed").WithCause(errors.Warning("maxConnWaitTimeout must be time.Duration format")).WithCause(err).WithMeta("transport", transportName)
			return
		}
	}

	isTLS := config.IsTLS
	if !isTLS {
		isTLS = config.TLSConfig != nil
	}
	var dialFunc fasthttp.DialFunc
	if config.TLSDialer != nil {
		dialFunc = func(addr string) (net.Conn, error) {
			return config.TLSDialer.DialContext(context.Background(), "tcp", addr)
		}
	}

	hc := &fasthttp.HostClient{
		Addr:                          address,
		Name:                          "",
		NoDefaultUserAgentHeader:      true,
		IsTLS:                         isTLS,
		TLSConfig:                     config.TLSConfig,
		Dial:                          dialFunc,
		MaxConns:                      config.MaxConnsPerHost,
		MaxConnDuration:               maxConnDuration,
		MaxIdleConnDuration:           maxIdleConnDuration,
		MaxIdemponentCallAttempts:     config.MaxIdemponentCallAttempts,
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
	if !config.DisableHttp2 && isTLS {
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
	client = &Client{
		address: address,
		secured: isTLS,
		host:    hc,
	}
	return
}

type Client struct {
	address string
	secured bool
	host    *fasthttp.HostClient
}

func (client *Client) Do(ctx context.Context, method []byte, path []byte, header transports.Header, body []byte) (status int, responseHeader transports.Header, responseBody []byte, err error) {
	req := fasthttp.AcquireRequest()

	// method
	req.Header.SetMethodBytes(method)
	// header
	header.Foreach(func(key []byte, values [][]byte) {
		for _, value := range values {
			req.Header.AddBytesKV(key, value)
		}
	})
	// uri
	uri := req.URI()
	if client.secured {
		uri.SetSchemeBytes(bytex.FromString("https"))
	} else {
		uri.SetSchemeBytes(bytex.FromString("http"))
	}
	uri.SetHostBytes(bytex.FromString(client.address))
	uri.SetPathBytes(path)

	// body
	if body != nil && len(body) > 0 {
		req.SetBodyRaw(body)
	}
	// resp
	resp := fasthttp.AcquireResponse()

	// do
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		err = client.host.DoDeadline(req, resp, deadline)
	} else {
		err = client.host.Do(req, resp)
	}

	fasthttp.ReleaseRequest(req)

	if err != nil {
		err = errors.Warning("fns: transport client do failed").
			WithCause(err).
			WithMeta("transport", transportName).WithMeta("method", bytex.ToString(method)).WithMeta("path", bytex.ToString(path))
		fasthttp.ReleaseResponse(resp)
		return
	}

	status = resp.StatusCode()
	responseHeader = &ResponseHeader{
		&resp.Header,
	}
	responseBody = resp.Body()

	fasthttp.ReleaseResponse(resp)
	return
}

func (client *Client) Close() {
	client.host.CloseIdleConnections()
}
