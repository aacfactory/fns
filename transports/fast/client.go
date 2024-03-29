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
	"bytes"
	"crypto/tls"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/fns/transports/ssl"
	"github.com/dgrr/http2"
	"github.com/valyala/fasthttp"
	"net"
	"strings"
	"time"
)

type ClientHttp2Config struct {
	Enabled            bool `json:"enabled"`
	PingSeconds        int  `json:"pingSeconds"`
	MaxResponseSeconds int  `json:"maxResponseSeconds"`
}

type DialerConfig struct {
	CacheSize     int `json:"cacheSize"`
	ExpireSeconds int `json:"expireSeconds"`
}

type ClientConfig struct {
	DialDualStack             bool         `json:"dialDualStack"`
	MaxConnsPerHost           int          `json:"maxConnsPerHost"`
	MaxIdleConnDuration       string       `json:"maxIdleConnDuration"`
	MaxConnDuration           string       `json:"maxConnDuration"`
	MaxIdemponentCallAttempts int          `json:"maxIdemponentCallAttempts"`
	ReadBufferSize            string       `json:"readBufferSize"`
	ReadTimeout               string       `json:"readTimeout"`
	WriteBufferSize           string       `json:"writeBufferSize"`
	WriteTimeout              string       `json:"writeTimeout"`
	MaxResponseBodySize       string       `json:"maxResponseBodySize"`
	MaxConnWaitTimeout        string       `json:"maxConnWaitTimeout"`
	Dialer                    DialerConfig `json:"dialer"`
	IsTLS                     bool         `json:"isTLS"`
	http2                     ClientHttp2Config
	TLSConfig                 *tls.Config `json:"-"`
	TLSDialer                 ssl.Dialer  `json:"-"`
}

func NewClient(address string, config ClientConfig) (client *Client, err error) {
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
			return config.TLSDialer.DialContext(context.TODO(), "tcp", addr)
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
	if config.http2.Enabled && isTLS {
		configErr := http2.ConfigureClient(hc, http2.ClientOpts{
			PingInterval:    time.Duration(config.http2.PingSeconds) * time.Second,
			MaxResponseTime: time.Duration(config.http2.MaxResponseSeconds) * time.Second,
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
	if header != nil {
		header.Foreach(func(key []byte, values [][]byte) {
			for _, value := range values {
				req.Header.AddBytesKV(key, value)
			}
		})
	}
	// uri
	uri := req.URI()
	if client.secured {
		uri.SetSchemeBytes(bytex.FromString("https"))
	} else {
		uri.SetSchemeBytes(bytex.FromString("http"))
	}
	uri.SetHostBytes(bytex.FromString(client.address))
	queryIdx := bytes.IndexByte(path, '?')
	if queryIdx > -1 {
		if len(path) > queryIdx {
			uri.SetQueryStringBytes(path[queryIdx+1:])
		}
		path = path[0:queryIdx]
	}
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

	if err != nil {
		err = errors.Warning("fns: transport client do failed").
			WithCause(err).
			WithMeta("transport", transportName).WithMeta("method", bytex.ToString(method)).WithMeta("path", bytex.ToString(path))
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
		return
	}

	status = resp.StatusCode()

	responseHeader = transports.NewHeader()
	resp.Header.VisitAll(func(key, value []byte) {
		responseHeader.Add(key, value)
	})

	responseBody = resp.Body()

	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)
	return
}

func (client *Client) Close() {
	client.host.CloseIdleConnections()
}
