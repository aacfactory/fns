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

package fns

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"net/http"
	"net/url"
	"time"
)

type httpClient struct {
	client *fasthttp.Client
}

func (c *httpClient) acquireRequest(_url string, head http.Header, body []byte) (req *fasthttp.Request, err error) {
	u, urlErr := url.Parse(_url)
	if urlErr != nil {
		err = urlErr
		return
	}
	req = fasthttp.AcquireRequest()
	// url
	req.URI().SetScheme(u.Scheme)
	if u.User != nil {
		req.URI().SetUsername(u.User.Username())
		password, hasPassword := u.User.Password()
		if hasPassword {
			req.URI().SetPassword(password)
		}
	}
	req.URI().SetHost(u.Host)
	req.URI().SetPath(u.Path)
	req.URI().SetQueryString(u.Query().Encode())
	if u.Fragment != "" {
		req.URI().SetHash(u.Fragment)
	}
	// head
	if head != nil && len(head) > 0 {
		headBuf := bytebufferpool.Get()
		wErr := head.Write(headBuf)
		if wErr != nil {
			err = wErr
			bytebufferpool.Put(headBuf)
			return
		}
		reader := bytes.NewReader(headBuf.Bytes())
		bytebufferpool.Put(headBuf)
		headReader := bufio.NewReader(reader)
		rErr := req.Header.Read(headReader)
		if rErr != nil {
			err = rErr
			return
		}
	}
	if body != nil && len(body) > 0 {
		req.SetBodyRaw(body)
	}
	return
}

func (c *httpClient) Get(url string, head http.Header, timeout time.Duration) (err error) {
	req, reqErr := c.acquireRequest(url, head, nil)
	if reqErr != nil {
		err = reqErr
		return
	}
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod("GET")
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if timeout < 1*time.Second {
		timeout = 5 * time.Second
	}

	doErr := c.client.DoTimeout(req, resp, timeout)
	if doErr != nil {
		if doErr == fasthttp.ErrTimeout {
			err = fmt.Errorf("timeout")
		} else {
			err = doErr
		}
		return
	}

	httpHeader := http.Header{}
	resp.Header.VisitAll(func(key, value []byte) {
		httpHeader.Add(string(key), string(value))
	})

	return
}

func (c *httpClient) Post(url string, head http.Header, body []byte, timeout time.Duration) (err error) {
	req, reqErr := c.acquireRequest(url, head, body)
	if reqErr != nil {
		err = reqErr
		return
	}
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod("POST")
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if timeout < 1*time.Second {
		timeout = 5 * time.Second
	}

	doErr := c.client.DoTimeout(req, resp, timeout)
	if doErr != nil {
		if doErr == fasthttp.ErrTimeout {
			err = fmt.Errorf("timeout")
		} else {
			err = doErr
		}
		return
	}

	httpHeader := http.Header{}
	resp.Header.VisitAll(func(key, value []byte) {
		httpHeader.Add(string(key), string(value))
	})

	return
}

func (c *httpClient) Put(url string, head http.Header, body []byte, timeout time.Duration) (err error) {
	req, reqErr := c.acquireRequest(url, head, body)
	if reqErr != nil {
		err = reqErr
		return
	}
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod("PUT")
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if timeout < 1*time.Second {
		timeout = 5 * time.Second
	}

	doErr := c.client.DoTimeout(req, resp, timeout)
	if doErr != nil {
		if doErr == fasthttp.ErrTimeout {
			err = fmt.Errorf("timeout")
		} else {
			err = doErr
		}
		return
	}

	httpHeader := http.Header{}
	resp.Header.VisitAll(func(key, value []byte) {
		httpHeader.Add(string(key), string(value))
	})

	return
}

func (c *httpClient) Close() {
	c.client.CloseIdleConnections()
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

