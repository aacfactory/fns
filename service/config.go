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
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service/transports/middlewares/cors"
	transports2 "github.com/aacfactory/fns/transports"
	"github.com/aacfactory/fns/transports/ssl"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Runtime   *RuntimeConfig   `json:"runtime" yaml:"runtime,omitempty"`
	Log       *LogConfig       `json:"log" yaml:"log,omitempty"`
	Transport *TransportConfig `json:"transport" yaml:"transport,omitempty"`
	Cluster   *ClusterConfig   `json:"cluster" yaml:"cluster,omitempty"`
	Proxy     *ProxyConfig     `json:"proxy" yaml:"proxy,omitempty"`
	Services  json.RawMessage  `json:"services" yaml:"services,omitempty"`
}

func (config *Config) Service(name string) (conf configures.Config, err error) {
	if name == "" {
		err = errors.Warning("fns: get service config failed").WithCause(errors.Warning("service name is nil"))
		return
	}
	if config.Services == nil || len(config.Services) == 0 {
		conf, err = configures.NewJsonConfig([]byte{'{', '}'})
		if err != nil {
			err = errors.Warning("fns: get service config failed").WithCause(err).WithMeta("service", name)
		}
		return
	}
	conf, err = configures.NewJsonConfig(config.Services)
	if err != nil {
		err = errors.Warning("fns: get service config failed").WithCause(err).WithMeta("service", name)
	}
	has := false
	conf, has = conf.Node(name)
	if !has {
		conf, err = configures.NewJsonConfig([]byte{'{', '}'})
		if err != nil {
			err = errors.Warning("fns: get service config failed").WithCause(err).WithMeta("service", name)
		}
		return
	}
	return
}

type LogConfig struct {
	Level     string `json:"level" yaml:"level,omitempty"`
	Formatter string `json:"formatter" yaml:"formatter,omitempty"`
	Color     bool   `json:"color" yaml:"color,omitempty"`
}

type ProxyConfig struct {
	TransportConfig
	EnableDevMode bool `json:"enableDevMode" yaml:"enableDevMode,omitempty"`
}

type RuntimeConfig struct {
	MaxWorkers           int                `json:"maxWorkers" yaml:"maxWorkers,omitempty"`
	WorkerMaxIdleSeconds int                `json:"workerMaxIdleSeconds" yaml:"workerMaxIdleSeconds,omitempty"`
	HandleTimeoutSeconds int                `json:"handleTimeoutSeconds" yaml:"handleTimeoutSeconds,omitempty"`
	AutoMaxProcs         AutoMaxProcsConfig `json:"autoMaxProcs" yaml:"autoMaxProcs,omitempty"`
	SecretKey            string             `json:"secretKey" yaml:"secretKey,omitempty"`
}

type AutoMaxProcsConfig struct {
	Min int `json:"min" yaml:"min,omitempty"`
	Max int `json:"max" yaml:"max,omitempty"`
}

type TransportConfig struct {
	Name        string           `json:"name" yaml:"name,omitempty"`
	Port        int              `json:"port" yaml:"port,omitempty"`
	Cors        *cors.CorsConfig `json:"cors" yaml:"cors,omitempty"`
	TLS         *TLSConfig       `json:"tls" yaml:"tls,omitempty"`
	Options     json.RawMessage  `json:"options" yaml:"options,omitempty"`
	Middlewares json.RawMessage  `json:"middlewares" yaml:"middlewares,omitempty"`
	Handlers    json.RawMessage  `json:"handlers" yaml:"handlers,omitempty"`
}

func (config *TransportConfig) MiddlewaresConfig() (conf configures.Config, err error) {
	if config.Middlewares == nil || len(config.Middlewares) == 0 {
		conf, err = configures.NewJsonConfig([]byte{'{', '}'})
		if err != nil {
			err = errors.Warning("fns: get middleware config failed").WithCause(err)
		}
		return
	}
	conf, err = configures.NewJsonConfig(config.Middlewares)
	if err != nil {
		err = errors.Warning("fns: get middleware config failed").WithCause(err)
		return
	}
	return
}

func (config *TransportConfig) HandlersConfig() (conf configures.Config, err error) {
	if config.Handlers == nil || len(config.Handlers) == 0 {
		conf, err = configures.NewJsonConfig([]byte{'{', '}'})
		if err != nil {
			err = errors.Warning("fns: get handler config failed").WithCause(err)
		}
		return
	}
	conf, err = configures.NewJsonConfig(config.Handlers)
	if err != nil {
		err = errors.Warning("fns: get handler config failed").WithCause(err)
		return
	}
	return
}

func (config *TransportConfig) ConvertToTransportsOptions(log logs.Logger, handler transports2.Handler) (options transports2.Options, err error) {
	options = transports2.Options{
		Port:    80,
		TLS:     nil,
		Handler: handler,
		Log:     log.With("fns", "transport"),
		Config:  nil,
	}
	if config == nil {
		return
	}
	if config.TLS != nil {
		var tlsErr error
		options.TLS, tlsErr = config.TLS.Config()
		if tlsErr != nil {
			err = errors.Warning("convert to transport options failed").WithCause(tlsErr)
			return
		}
	}
	port := config.Port
	if port == 0 {
		if options.TLS == nil {
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
	Kind    string          `json:"kind" yaml:"kind,omitempty"`
	Options json.RawMessage `json:"options" yaml:"options,omitempty"`
}

func (config *TLSConfig) Config() (conf ssl.Config, err error) {
	kind := strings.TrimSpace(config.Kind)
	hasConf := false
	conf, hasConf = ssl.GetConfig(kind)
	if !hasConf {
		err = errors.Warning(fmt.Sprintf("fns: can not get %s tls config", kind))
		return
	}
	confOptions, confOptionsErr := configures.NewJsonConfig(config.Options)
	if confOptionsErr != nil {
		err = errors.Warning(fmt.Sprintf("fns: can not get options of %s tls config", kind)).WithCause(confOptionsErr)
		return
	}
	err = conf.Build(confOptions)
	return
}

type ClusterConfig struct {
	Kind                 string          `json:"kind" yaml:"kind,omitempty"`
	FetchMembersInterval string          `json:"fetchMembersInterval" yaml:"fetchMembersInterval,omitempty"`
	Options              json.RawMessage `json:"options" yaml:"options,omitempty"`
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
