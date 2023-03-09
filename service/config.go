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

package service

import (
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service/ssl"
	"github.com/aacfactory/json"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Runtime *RuntimeConfig `json:"runtime"`
	Log     *LogConfig     `json:"log"`
	Http    *HttpConfig    `json:"http"`
	Cluster *ClusterConfig `json:"cluster"`
}

type LogConfig struct {
	Level     string `json:"level"`
	Formatter string `json:"formatter"`
	Color     bool   `json:"color"`
}

type RuntimeConfig struct {
	MaxWorkers                int                `json:"maxWorkers"`
	WorkerMaxIdleSeconds      int                `json:"workerMaxIdleSeconds"`
	HandleTimeoutSeconds      int                `json:"handleTimeoutSeconds"`
	LocalSharedStoreCacheSize string             `json:"localSharedStoreCacheSize"`
	AutoMaxProcs              AutoMaxProcsConfig `json:"autoMaxProcs"`
	SecretKey                 string             `json:"secretKey"`
}

type AutoMaxProcsConfig struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type HttpConfig struct {
	Port     int             `json:"port"`
	Cors     *CorsConfig     `json:"cors"`
	TLS      *TLSConfig      `json:"tls"`
	Options  json.RawMessage `json:"options"`
	Handlers json.RawMessage `json:"handlers"`
}

type TLSConfig struct {
	// Kind
	// ACME
	// SSC(SELF-SIGN-CERT)
	// DEFAULT
	Kind    string          `json:"kind"`
	Options json.RawMessage `json:"options"`
}

func (config *TLSConfig) Config() (serverTLS *tls.Config, clientTLS *tls.Config, err error) {
	kind := strings.TrimSpace(config.Kind)
	loader, hasLoader := ssl.GetLoader(kind)
	if !hasLoader {
		err = errors.Warning(fmt.Sprintf("fns: can not get %s tls loader", kind))
		return
	}
	loaderConfig, loaderConfigErr := configures.NewJsonConfig(config.Options)
	if loaderConfigErr != nil {
		err = errors.Warning(fmt.Sprintf("fns: can not get options of %s tls loader", kind)).WithCause(loaderConfigErr)
		return
	}
	serverTLS, clientTLS, err = loader(loaderConfig)
	return
}

type CorsConfig struct {
	AllowedOrigins      []string `json:"allowedOrigins"`
	AllowedHeaders      []string `json:"allowedHeaders"`
	ExposedHeaders      []string `json:"exposedHeaders"`
	AllowCredentials    bool     `json:"allowCredentials"`
	MaxAge              int      `json:"maxAge"`
	AllowPrivateNetwork bool     `json:"allowPrivateNetwork"`
}

type ClusterConfig struct {
	Kind    string          `json:"kind"`
	DevMode *ClusterDevMode `json:"devMode"`
	Shared  *SharedConfig   `json:"shared"`
	Options json.RawMessage `json:"options"`
}

type ClusterDevMode struct {
	ProxyAddress string     `json:"proxyAddress"`
	TLS          *TLSConfig `json:"tls"`
}

type SharedConfig struct {
	BarrierDisabled        bool   `json:"barrierDisabled"`
	BarrierTTLMilliseconds uint64 `json:"barrierTTLMilliseconds"`
}

const (
	activeSystemEnvKey = "FNS-ACTIVE"
)

func DefaultConfigRetrieverOption() (option configures.RetrieverOption) {
	path, pathErr := filepath.Abs("./config")
	if pathErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: create default config retriever failed, cant not get absolute representation of './config'").WithCause(pathErr)))
		return
	}
	active, _ := os.LookupEnv(activeSystemEnvKey)
	active = strings.TrimSpace(active)
	store := configures.NewFileStore(path, "fns", '-')
	option = configures.RetrieverOption{
		Active: active,
		Format: "YAML",
		Store:  store,
	}
	return
}
