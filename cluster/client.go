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
	"github.com/aacfactory/logs"
	"net/http"
	"time"
)

type ClientOptions struct {
	Log                 logs.Logger
	TLS                 *tls.Config
	MaxIdleConnDuration time.Duration
	MaxConnsPerHost     int
	MaxIdleConnsPerHost int
}

func newClientOptions(log logs.Logger, TLS *tls.Config, config ClientConfig) ClientOptions {
	maxIdleConnDuration := time.Duration(config.MaxIdleClientConnSeconds) * time.Second
	if maxIdleConnDuration < 1 {
		maxIdleConnDuration = 10 * time.Second
	}
	maxConnsPerHost := config.MaxClientConnsPerHost
	if maxConnsPerHost < 1 {
		maxConnsPerHost = 512
	}
	maxIdleConnsPerHost := config.MaxIdleClientConnsPerHost
	if maxIdleConnsPerHost < 1 {
		maxConnsPerHost = 512
	}
	return ClientOptions{
		Log:                 log,
		TLS:                 TLS,
		MaxIdleConnDuration: maxIdleConnDuration,
		MaxConnsPerHost:     maxConnsPerHost,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
	}
}

type ClientBuilder func(options ClientOptions) (client Client, err error)

type Client interface {
	Do(ctx sc.Context, method string, url string, header http.Header, body []byte) (status int, respHeader http.Header, respBody []byte, err error)
}
