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
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/fns/commons"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	activeSystemEnvKey = "FNS-ACTIVE"
)

func defaultConfigRetrieverOption() (option configuares.RetrieverOption) {
	path, pathErr := filepath.Abs("./config")
	if pathErr != nil {
		panic(fmt.Sprintf("fns create default config retriever failed, cant not get './config'"))
		return
	}
	active, _ := os.LookupEnv(activeSystemEnvKey)
	active = strings.TrimSpace(active)
	store := configuares.NewFileStore(path, "fns", '-')
	option = configuares.RetrieverOption{
		Active: active,
		Format: "YAML",
		Store:  store,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type ApplicationConfig struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Terms       string         `json:"terms,omitempty"`
	Contact     *appContact    `json:"contact,omitempty"`
	License     *appLicense    `json:"license,omitempty"`
	Concurrency int            `json:"concurrency,omitempty"`
	Http        HttpConfig     `json:"http,omitempty"`
	Log         LogConfig      `json:"log,omitempty"`
	Services    ServicesConfig `json:"services,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type HttpConfig struct {
	Host                     string         `json:"host,omitempty"`
	Port                     int            `json:"port,omitempty"`
	PublicHost               string         `json:"publicHost,omitempty"`
	PublicPort               int            `json:"publicPort,omitempty"`
	MaxConnectionsPerIP      int            `json:"maxConnectionsPerIp,omitempty"`
	MaxRequestsPerConnection int            `json:"maxRequestsPerConnection,omitempty"`
	KeepAlive                bool           `json:"keepAlive,omitempty"`
	KeepalivePeriodSecond    int            `json:"keepalivePeriodSecond,omitempty"`
	RequestTimeoutSeconds    int            `json:"requestTimeoutSeconds,omitempty"`
	ReadBufferSize           string         `json:"readBufferSize"`
	WriteBufferSize          string         `json:"writeBufferSize"`
	Cors                     CorsConfig     `json:"cors"`
	TLS                      *HttpTlsConfig `json:"tls,omitempty"`
}

type HttpTlsConfig struct {
	Enable     bool   `json:"enable,omitempty"`
	PublicKey  string `json:"publicKey,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
}

func (c *HttpTlsConfig) mapToTLS() (config *tls.Config, err error) {
	// pub
	pub, pubErr := c.readFile(c.PublicKey)
	if pubErr != nil {
		err = fmt.Errorf("fns: read public key failed, %v", pubErr)
		return
	}
	// pri
	pri, priErr := c.readFile(c.PrivateKey)
	if priErr != nil {
		err = fmt.Errorf("fns: read private key failed, %v", priErr)
		return
	}

	certificate, certificateErr := tls.X509KeyPair(pub, pri)
	if certificateErr != nil {
		err = fmt.Errorf("fns: parse key pair failed, %v", certificateErr)
		return
	}
	config = &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}

	return
}

func (c *HttpTlsConfig) readFile(s string) (p []byte, err error) {
	u, urlErr := url.Parse(s)
	if urlErr != nil {
		err = fmt.Errorf("parse url failed, %v", urlErr)
		return
	}
	switch strings.ToLower(u.Scheme) {
	case "file", "":
		pubFilePath := filepath.Join(u.Host, u.Path)
		p, err = ioutil.ReadFile(pubFilePath)
	case "env":
		pubFilePath, has := os.LookupEnv(u.Host)
		if !has {
			err = fmt.Errorf("url is invalid, %s was not found", u.Host)
			return
		}
		p, err = ioutil.ReadFile(pubFilePath)
	case "http", "https":
		status, body, getErr := fasthttp.GetTimeout(make([]byte, 0, 1), s, 30*time.Second)
		if getErr != nil {
			err = fmt.Errorf("url is invalid, get from %s failed, %v", s, getErr)
			return
		}
		if status != 200 {
			err = fmt.Errorf("url is invalid, get from %s failed, %v", s, status)
			return
		}
		p = body
	default:
		err = fmt.Errorf("url is invalid, schema is not supported")
		return
	}
	return
}

const (
	publicHostEnv = "PUBLIC_HOST"
	publicPortEnv = "PUBLIC_PORT"
)

func getPublicHostFromEnv() (host string, has bool) {
	host, has = os.LookupEnv(publicHostEnv)
	if has {
		host = strings.TrimSpace(host)
		has = host != ""
	}
	return
}

func getPublicPortFromEnv() (port int, has bool) {
	portStr, ok := os.LookupEnv(publicPortEnv)
	if ok {
		portInt, parseErr := strconv.Atoi(strings.TrimSpace(portStr))
		if parseErr == nil {
			port = portInt
			has = true
		}
	}
	return
}

func getPublicHostFromHostname() (host string, has bool) {
	ip, err := commons.IpFromHostname(false)
	if err != nil {
		return
	}
	host = ip
	has = true
	return
}

type CorsConfig struct {
	Enable           bool     `json:"enable,omitempty"`
	AllowedOrigins   []string `json:"allowedOrigins,omitempty"`
	AllowedMethods   []string `json:"allowedMethods,omitempty"`
	AllowedHeaders   []string `json:"allowedHeaders,omitempty"`
	ExposedHeaders   []string `json:"exposedHeaders,omitempty"`
	AllowCredentials bool     `json:"allowCredentials,omitempty"`
	MaxAge           int      `json:"maxAge,omitempty"`
}

func (cors *CorsConfig) fill() {
	if cors.ExposedHeaders == nil || len(cors.ExposedHeaders) == 0 {
		cors.ExposedHeaders = make([]string, 0, 1)
	}
	cors.ExposedHeaders = append(cors.ExposedHeaders, string(requestIdHeader))
	cors.ExposedHeaders = append(cors.ExposedHeaders, string(responseLatencyHeader))
	cors.ExposedHeaders = append(cors.ExposedHeaders, "Server")
	return
}

func (cors *CorsConfig) originAllowed(origin string) (ok bool) {
	origin = strings.ToLower(origin)
	for _, allowedOrigin := range cors.AllowedOrigins {
		if allowedOrigin == "*" {
			ok = true
			return
		}
		if allowedOrigin == origin {
			ok = true
			return
		}
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type ServicesConfig struct {
	HandleTimeoutSecond int                  `json:"handleTimeoutSecond,omitempty"`
	MaxIdleTimeSecond   int                  `json:"maxIdleTimeSecond,omitempty"`
	ReduceMemoryUsage   bool                 `json:"reduceMemoryUsage,omitempty"`
	Discovery           DiscoveryConfig      `json:"discovery,omitempty"`
	Authorization       AuthorizationsConfig `json:"authorization,omitempty"`
	Permission          PermissionsConfig    `json:"permission,omitempty"`
	HttpClientPoolSize  int                  `json:"httpClientPoolSize,omitempty"`
}

type DiscoveryConfig struct {
	Enable bool            `json:"enable,omitempty"`
	Kind   string          `json:"kind,omitempty"`
	Config configuares.Raw `json:"config,omitempty"`
}

type AuthorizationsConfig struct {
	Enable bool            `json:"enable,omitempty"`
	Kind   string          `json:"kind,omitempty"`
	Config configuares.Raw `json:"config,omitempty"`
}

type PermissionsConfig struct {
	Enable bool            `json:"enable,omitempty"`
	Loader string          `json:"loader,omitempty"`
	Config configuares.Raw `json:"config,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type LogConfig struct {
	Level     string `json:"level,omitempty"`
	Formatter string `json:"formatter,omitempty"`
	Color     bool   `json:"color,omitempty"`
}
