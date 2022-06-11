/*
 * Copyright 2021 Wang Min Xiang
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
 */

package cluster

import (
	sc "context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	"net/http"
	"time"
)

type ClientOptions struct {
	Log                 logs.Logger
	Https               bool
	TLS                 *tls.Config
	MaxIdleConnDuration time.Duration
	MaxConnsPerHost     int
	MaxIdleConnsPerHost int
	RequestTimeout      time.Duration
}

type ClientBuilder func(options ClientOptions) (client Client, err error)

type Client interface {
	Do(ctx context.Context, method string, host string, uri string, header http.Header, body []byte) (status int, respHeader http.Header, respBody []byte, err error)
	Close()
}

func FastHttpClientBuilder(options ClientOptions) (client Client, err error) {
	schema := "http"
	if options.Https {
		schema = "https"
	}
	client = &FastHttpClient{
		log: options.Log.With("fns", "client"),
		core: &fasthttp.Client{
			Name:                     "fns",
			NoDefaultUserAgentHeader: true,
			Dial:                     nil,
			DialDualStack:            false,
			TLSConfig:                options.TLS,
			MaxConnsPerHost:          options.MaxIdleConnsPerHost,
		},
		schema:  schema,
		timeout: options.RequestTimeout,
	}
	return
}

type FastHttpClient struct {
	log     logs.Logger
	core    *fasthttp.Client
	schema  string
	timeout time.Duration
}

func (client *FastHttpClient) Do(_ context.Context, method string, host string, uri string, header http.Header, body []byte) (status int, respHeader http.Header, respBody []byte, err error) {
	req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}()
	req.Header.SetMethod(method)
	req.Header.SetRequestURI(fmt.Sprintf("%s://%s%s", client.schema, host, uri))
	if header != nil {
		for hk, hvv := range header {
			for _, hv := range hvv {
				req.Header.Add(hk, hv)
			}
		}
	}
	if body != nil && len(body) > 0 {
		req.SetBody(body)
	}
	doErr := client.core.DoTimeout(req, res, client.timeout)
	if doErr != nil {
		err = errors.Warning("fns: http client do failed").WithMeta("method", method).WithMeta("url", url).WithCause(doErr)
		return
	}
	status = res.StatusCode()
	respHeader = http.Header{}
	res.Header.VisitAll(func(key, value []byte) {
		respHeader.Add(string(key), string(value))
	})
	respBody = res.Body()
	return
}

func (client *FastHttpClient) Close() {
	client.core.CloseIdleConnections()
}

func join(ctx sc.Context, client Client, address string, node *Node) (result *joinResult, err errors.CodeError) {
	var url string
	if node.SSL {
		url = fmt.Sprintf("https://%s%s", address, joinPath)
	} else {
		url = fmt.Sprintf("http://%s%s", address, joinPath)
	}
	body, encodeErr := json.Marshal(node)
	if encodeErr != nil {
		err = errors.BadRequest("fns: encode node failed").WithCause(encodeErr)
		return
	}
	header := http.Header{}
	header.Set("Content-Type", contentType)
	resp, doErr := client.Do(ctx, http.MethodPost, url, header, encodeRequestBody(body))
	if doErr != nil {
		err = errors.Warning("fns: invoke join node failed").WithCause(doErr)
		return
	}
	data, handleErr := decodeResponseBody(resp)
	if handleErr != nil {
		err = handleErr
		return
	}
	result = &joinResult{}
	decodeErr := json.Unmarshal(data, result)
	if decodeErr != nil {
		err = errors.Warning("fns: invoke join node failed").WithCause(decodeErr)
		return
	}
	return
}

func leave(ctx sc.Context, client Client, address string, node *Node) (err errors.CodeError) {
	var url string
	if node.SSL {
		url = fmt.Sprintf("https://%s%s", address, leavePath)
	} else {
		url = fmt.Sprintf("http://%s%s", address, leavePath)
	}
	body, encodeErr := json.Marshal(node)
	if encodeErr != nil {
		err = errors.BadRequest("fns: encode node failed").WithCause(encodeErr)
		return
	}
	header := http.Header{}
	header.Set("Content-Type", contentType)
	resp, doErr := client.Do(ctx, http.MethodPost, url, header, encodeRequestBody(body))
	if doErr != nil {
		err = errors.Warning("fns: invoke leave node failed").WithCause(doErr)
		return
	}
	_, handleErr := decodeResponseBody(resp)
	if handleErr != nil {
		err = handleErr
		return
	}
	return
}
