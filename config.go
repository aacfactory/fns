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
	"encoding/json"
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
	Name      string          `json:"name,omitempty"`
	SecretKey string          `json:"secretKey,omitempty"`
	Http      HttpConfig      `json:"http,omitempty"`
	Workers   WorkersConfig   `json:"workers,omitempty"`
	Log       LogConfig       `json:"log,omitempty"`
	Discovery DiscoveryConfig `json:"discovery,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type HttpConfig struct {
	Host                     string    `json:"host,omitempty"`
	Port                     int       `json:"port,omitempty"`
	PublicHost               string    `json:"publicHost,omitempty"`
	PublicPort               int       `json:"publicPort,omitempty"`
	MaxConnectionsPerIP      int       `json:"maxConnectionsPerIp,omitempty"`
	MaxRequestsPerConnection int       `json:"maxRequestsPerConnection,omitempty"`
	KeepAlive                bool      `json:"keepAlive,omitempty"`
	KeepalivePeriodSecond    int       `json:"keepalivePeriodSecond,omitempty"`
	RequestTimeoutSeconds    int       `json:"requestTimeoutSeconds,omitempty"`
	WhiteCIDR                []string  `json:"whiteCIDR,omitempty"`
	SSL                      ServerTLS `json:"ssl,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type WorkersConfig struct {
	Concurrency       int `json:"concurrency,omitempty"`
	MaxIdleTimeSecond int `json:"maxIdleTimeSecond,omitempty"`
	// Aggressively reduces memory usage at the cost of higher CPU usage
	// if set to true.
	//
	// Try enabling this option only if the server consumes too much memory
	// serving mostly idle keep-alive connections. This may reduce memory
	// usage by more than 50%.
	//
	// Aggressive memory usage reduction is disabled by default.
	ReduceMemoryUsage bool `json:"reduceMemoryUsage,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type LogConfig struct {
	Level     string `json:"level,omitempty"`
	Formatter string `json:"formatter,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type DiscoveryConfig struct {
	Enable bool            `json:"enable,omitempty"`
	Kind   string          `json:"kind,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}
