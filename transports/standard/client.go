package standard

import (
	"bytes"
	sc "context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/fns/transports/ssl"
	"github.com/valyala/bytebufferpool"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type ClientConfig struct {
	MaxConnsPerHost       int         `json:"maxConnsPerHost"`
	MaxResponseHeaderSize string      `json:"maxResponseHeaderSize"`
	Timeout               string      `json:"timeout"`
	DisableKeepAlive      bool        `json:"disableKeepAlive"`
	MaxIdleConnsPerHost   int         `json:"maxIdleConnsPerHost"`
	IdleConnTimeout       string      `json:"idleConnTimeout"`
	TLSHandshakeTimeout   string      `json:"tlsHandshakeTimeout"`
	ExpectContinueTimeout string      `json:"expectContinueTimeout"`
	IsTLS                 bool        `json:"isTLS"`
	TLSConfig             *tls.Config `json:"-"`
	TLSDialer             ssl.Dialer  `json:"-"`
}

func (config *ClientConfig) MaxConnectionsPerHost() (n int) {
	if config.MaxConnsPerHost < 1 {
		config.MaxConnsPerHost = 64
	}
	n = config.MaxConnsPerHost
	return
}

func (config *ClientConfig) MaxIdleConnectionsPerHost() (n int) {
	if config.MaxIdleConnsPerHost < 1 {
		config.MaxIdleConnsPerHost = 100
	}
	n = config.MaxIdleConnsPerHost
	return
}

func (config *ClientConfig) MaxResponseHeaderByteSize() (n uint64, err error) {
	maxResponseHeaderSize := strings.TrimSpace(config.MaxResponseHeaderSize)
	if maxResponseHeaderSize == "" {
		maxResponseHeaderSize = "4KB"
	}
	n, err = bytex.ParseBytes(maxResponseHeaderSize)
	if err != nil {
		err = errors.Warning("maxResponseHeaderBytes is invalid").WithCause(err).WithMeta("hit", "format must be bytes")
		return
	}
	return
}

func (config *ClientConfig) TimeoutDuration() (n time.Duration, err error) {
	timeout := strings.TrimSpace(config.Timeout)
	if timeout == "" {
		timeout = "2s"
	}
	n, err = time.ParseDuration(timeout)
	if err != nil {
		err = errors.Warning("timeout is invalid").WithCause(err).WithMeta("hit", "format must be time.Duration")
		return
	}
	return
}

func (config *ClientConfig) IdleConnTimeoutDuration() (n time.Duration, err error) {
	timeout := strings.TrimSpace(config.IdleConnTimeout)
	if timeout == "" {
		timeout = "90s"
	}
	n, err = time.ParseDuration(timeout)
	if err != nil {
		err = errors.Warning("idle conn timeout is invalid").WithCause(err).WithMeta("hit", "format must be time.Duration")
		return
	}
	return
}

func (config *ClientConfig) TLSHandshakeTimeoutDuration() (n time.Duration, err error) {
	timeout := strings.TrimSpace(config.TLSHandshakeTimeout)
	if timeout == "" {
		timeout = "10s"
	}
	n, err = time.ParseDuration(timeout)
	if err != nil {
		err = errors.Warning("tls handshake timeout is invalid").WithCause(err).WithMeta("hit", "format must be time.Duration")
		return
	}
	return
}

func (config *ClientConfig) ExpectContinueTimeoutDuration() (n time.Duration, err error) {
	timeout := strings.TrimSpace(config.ExpectContinueTimeout)
	if timeout == "" {
		timeout = "1s"
	}
	n, err = time.ParseDuration(timeout)
	if err != nil {
		err = errors.Warning("expect continue timeout is invalid").WithCause(err).WithMeta("hit", "format must be time.Duration")
		return
	}
	return
}

func NewClient(address string, config *ClientConfig) (client *Client, err error) {
	maxResponseHeaderBytes, maxResponseHeaderBytesErr := config.MaxResponseHeaderByteSize()
	if maxResponseHeaderBytesErr != nil {
		err = maxResponseHeaderBytesErr
		return
	}
	timeout, timeoutErr := config.TimeoutDuration()
	if timeoutErr != nil {
		err = timeoutErr
		return
	}
	idleConnTimeout, idleConnTimeoutErr := config.IdleConnTimeoutDuration()
	if idleConnTimeoutErr != nil {
		err = idleConnTimeoutErr
		return
	}
	tlsHandshakeTimeout, tlsHandshakeTimeoutErr := config.TLSHandshakeTimeoutDuration()
	if tlsHandshakeTimeoutErr != nil {
		err = tlsHandshakeTimeoutErr
		return
	}
	expectContinueTimeout, expectContinueTimeoutErr := config.ExpectContinueTimeoutDuration()
	if expectContinueTimeoutErr != nil {
		err = expectContinueTimeoutErr
		return
	}
	isTLS := config.IsTLS
	if !isTLS {
		isTLS = config.TLSConfig != nil
	}
	var dialFunc func(ctx sc.Context, network, addr string) (net.Conn, error)
	if config.TLSDialer != nil {
		dialFunc = config.TLSDialer.DialContext
	}
	roundTripper := &http.Transport{
		Proxy:                  http.ProxyFromEnvironment,
		DialContext:            dialFunc,
		DialTLSContext:         nil,
		TLSClientConfig:        config.TLSConfig,
		TLSHandshakeTimeout:    tlsHandshakeTimeout,
		DisableKeepAlives:      config.DisableKeepAlive,
		DisableCompression:     false,
		MaxIdleConns:           config.MaxIdleConnectionsPerHost(),
		MaxIdleConnsPerHost:    config.MaxIdleConnectionsPerHost(),
		MaxConnsPerHost:        config.MaxConnectionsPerHost(),
		IdleConnTimeout:        idleConnTimeout,
		ResponseHeaderTimeout:  0,
		ExpectContinueTimeout:  expectContinueTimeout,
		TLSNextProto:           nil,
		MaxResponseHeaderBytes: int64(maxResponseHeaderBytes),
		WriteBufferSize:        4096,
		ReadBufferSize:         4096,
		ForceAttemptHTTP2:      true,
	}
	client = &Client{
		address: address,
		secured: isTLS,
		host: &http.Client{
			Transport:     roundTripper,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       timeout,
		},
	}
	return
}

type Client struct {
	address string
	secured bool
	host    *http.Client
}

func (c Client) Key() (key string) {
	key = c.address
	return
}

func (c Client) Do(ctx context.Context, method []byte, path []byte, header transports.Header, body []byte) (status int, responseHeader transports.Header, responseBody []byte, err error) {
	url := ""
	if c.secured {
		url = fmt.Sprintf("https://%s%s", c.address, bytex.ToString(path))
	} else {
		url = fmt.Sprintf("http://%s%s", c.address, bytex.ToString(path))
	}
	r, rErr := http.NewRequestWithContext(ctx, bytex.ToString(method), url, bytes.NewReader(body))
	if rErr != nil {
		err = errors.Warning("http: create request failed").WithCause(rErr)
		return
	}
	if header != nil {
		header.Foreach(func(key []byte, values [][]byte) {
			for _, value := range values {
				r.Header.Add(bytex.ToString(key), bytex.ToString(value))
			}
		})
	}

	resp, doErr := c.host.Do(r)
	if doErr != nil {
		if errors.Map(doErr).Contains(context.Canceled) || errors.Map(doErr).Contains(context.DeadlineExceeded) {
			err = errors.Timeout("http: do failed").WithCause(doErr)
			return
		}
		err = errors.Warning("http: do failed").WithCause(doErr)
		return
	}
	buf := bytex.Acquire4KBuffer()
	defer bytex.Release4KBuffer(buf)
	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)
	for {
		n, readErr := resp.Body.Read(buf)
		_, _ = b.Write(buf[0:n])
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			_ = resp.Body.Close()
			err = errors.Warning("http: do failed").WithCause(errors.Warning("read response body failed").WithCause(readErr))
			return
		}
	}
	status = resp.StatusCode
	responseHeader = WrapHttpHeader(resp.Header)
	responseBody = b.Bytes()

	return
}

func (c Client) Close() {
	c.host.CloseIdleConnections()
	return
}
