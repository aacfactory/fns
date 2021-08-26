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
	"fmt"
	"github.com/aacfactory/configuares"
	"os"
	"path/filepath"
	"strings"
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
	Name      string         `json:"name,omitempty"`
	SecretKey string         `json:"secretKey,omitempty"`
	Http      HttpConfig     `json:"http,omitempty"`
	Log       LogConfig      `json:"log,omitempty"`
	Services  ServicesConfig `json:"services,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type HttpConfig struct {
	Host                     string      `json:"host,omitempty"`
	Port                     int         `json:"port,omitempty"`
	PublicHost               string      `json:"publicHost,omitempty"`
	PublicPort               int         `json:"publicPort,omitempty"`
	MaxConnectionsPerIP      int         `json:"maxConnectionsPerIp,omitempty"`
	MaxRequestsPerConnection int         `json:"maxRequestsPerConnection,omitempty"`
	KeepAlive                bool        `json:"keepAlive,omitempty"`
	KeepalivePeriodSecond    int         `json:"keepalivePeriodSecond,omitempty"`
	RequestTimeoutSeconds    int         `json:"requestTimeoutSeconds,omitempty"`
	Cors                     *CorsConfig `json:"cors,omitempty"`
	WhiteCIDR                []string    `json:"whiteCIDR,omitempty"`
}

type CorsConfig struct {
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
	Concurrency         int                  `json:"concurrency,omitempty"`
	HandleTimeoutSecond int                  `json:"handleTimeoutSecond,omitempty"`
	MaxIdleTimeSecond   int                  `json:"maxIdleTimeSecond,omitempty"`
	ReduceMemoryUsage   bool                 `json:"reduceMemoryUsage,omitempty"`
	Discovery           DiscoveryConfig      `json:"discovery,omitempty"`
	Authorization       AuthorizationsConfig `json:"authorization,omitempty"`
	Permission          PermissionsConfig    `json:"permission,omitempty"`
	serverId            string
	address             string
	version             string
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
	Kind   string          `json:"kind,omitempty"`
	Config configuares.Raw `json:"config,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type LogConfig struct {
	Level     string `json:"level,omitempty"`
	Formatter string `json:"formatter,omitempty"`
	Color     bool   `json:"color,omitempty"`
}
