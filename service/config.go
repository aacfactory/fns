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
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Runtime   *RuntimeConfig   `json:"runtime"`
	Log       *LogConfig       `json:"log"`
	Transport *TransportConfig `json:"transport"`
	Cluster   *ClusterConfig   `json:"cluster"`
	Proxy     *ProxyConfig     `json:"proxy"`
}

type LogConfig struct {
	Level     string `json:"level"`
	Formatter string `json:"formatter"`
	Color     bool   `json:"color"`
}

type ProxyConfig struct {
	EnableDevMode bool `json:"enableDevMode"`
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

type TransportConfig struct {
	Port        int                    `json:"port"`
	Cors        *transports.CorsConfig `json:"cors"`
	TLS         *TLSConfig             `json:"tls"`
	Options     json.RawMessage        `json:"options"`
	Middlewares json.RawMessage        `json:"middlewares"`
	Handlers    json.RawMessage        `json:"handlers"`
}

func (config *TransportConfig) MiddlewareConfig(name string) (conf configures.Config, err error) {
	if config.Middlewares == nil || len(config.Middlewares) == 0 {
		conf, err = configures.NewJsonConfig([]byte{'{', '}'})
		return
	}
	conf, err = configures.NewJsonConfig(config.Middlewares)
	if err != nil {
		err = errors.Warning(fmt.Sprintf("get %s middleware config failed", name)).WithCause(err)
		return
	}
	has := false
	conf, has = conf.Node(name)
	if !has {
		conf, err = configures.NewJsonConfig([]byte{'{', '}'})
		return
	}
	return
}

func (config *TransportConfig) HandlerConfig(name string) (conf configures.Config, err error) {
	if config.Handlers == nil || len(config.Handlers) == 0 {
		conf, err = configures.NewJsonConfig([]byte{'{', '}'})
		return
	}
	conf, err = configures.NewJsonConfig(config.Handlers)
	if err != nil {
		err = errors.Warning(fmt.Sprintf("get %s handler config failed", name)).WithCause(err)
		return
	}
	has := false
	conf, has = conf.Node(name)
	if !has {
		conf, err = configures.NewJsonConfig([]byte{'{', '}'})
		return
	}
	return
}

func (config *TransportConfig) ConvertToTransportsOptions(log logs.Logger, handler transports.Handler) (options transports.Options, err error) {
	options = transports.Options{
		Port:      80,
		ServerTLS: nil,
		ClientTLS: nil,
		Handler:   handler,
		Log:       log.With("fns", "transport"),
		Config:    nil,
	}
	if config == nil {
		return
	}
	var serverTLS *tls.Config
	var clientTLS *tls.Config
	if config.TLS != nil {
		var tlsErr error
		serverTLS, clientTLS, tlsErr = config.TLS.Config()
		if tlsErr != nil {
			err = errors.Warning("convert to transport options failed").WithCause(tlsErr)
			return
		}
	}
	options.ServerTLS = serverTLS
	options.ClientTLS = clientTLS
	port := config.Port
	if port == 0 {
		if serverTLS == nil {
			port = 80
		} else {
			port = 443
		}
	}
	if port < 1 || port > 65535 {
		err = errors.Warning("convert to transport options failed").WithCause(fmt.Errorf("port is invalid, port must great than 1024 or less than 65536"))
		return
	}
	options.Port = port
	if config.Options == nil {
		config.Options = []byte("{}")
	}
	options.Config, err = configures.NewJsonConfig(config.Options)
	if err != nil {
		err = errors.Warning("convert to transport options failed").WithCause(fmt.Errorf("options is invalid")).WithCause(err)
		return
	}
	return
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

type ClusterConfig struct {
	Kind                 string          `json:"kind"`
	FetchMembersInterval string          `json:"fetchMembersInterval"`
	DevMode              *ClusterDevMode `json:"devMode"`
	Shared               *SharedConfig   `json:"shared"`
	Options              json.RawMessage `json:"options"`
}

type ClusterDevMode struct {
	ProxyAddress string     `json:"proxyAddress"`
	TLS          *TLSConfig `json:"tls"`
}

type SharedConfig struct {
	BarrierDisabled        bool   `json:"barrierDisabled"`
	BarrierTTLMilliseconds uint64 `json:"barrierTTLMilliseconds"`
}

type RequestLimiterConfig struct {
	MaxPerDeviceRequest int64  `json:"maxPerDeviceRequest"`
	RetryAfter          string `json:"retryAfter"`
}

const (
	activeSystemEnvKey = "FNS-ACTIVE"
)

func DefaultConfigRetrieverOption() (option configures.RetrieverOption) {
	path, pathErr := filepath.Abs("./configs")
	if pathErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: create default config retriever failed, cant not get absolute representation of './configs'").WithCause(pathErr)))
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
