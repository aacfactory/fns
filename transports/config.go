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

package transports

import (
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/transports/ssl"
	"github.com/aacfactory/json"
	"strings"
)

func FixedTLSConfig(conf ssl.Config) *TLSConfig {
	return &TLSConfig{
		conf: conf,
	}
}

type TLSConfig struct {
	// Kind
	// ACME
	// SSC(SELF-SIGN-CERT)
	// DEFAULT
	Kind    string          `json:"kind" yaml:"kind,omitempty"`
	Options json.RawMessage `json:"options" yaml:"options,omitempty"`
	conf    ssl.Config
}

func (config *TLSConfig) Config() (conf ssl.Config, err error) {
	if config.conf != nil {
		conf = config.conf
		return
	}
	kind := strings.TrimSpace(config.Kind)
	hasConf := false
	conf, hasConf = ssl.GetConfig(kind)
	if !hasConf {
		err = errors.Warning(fmt.Sprintf("fns: can not get %s tls config", kind))
		return
	}
	if len(config.Options) == 0 {
		config.Options = []byte{'{', '}'}
	}
	confOptions, confOptionsErr := configures.NewJsonConfig(config.Options)
	if confOptionsErr != nil {
		err = errors.Warning(fmt.Sprintf("fns: can not get options of %s tls config", kind)).WithCause(confOptionsErr)
		return
	}
	err = conf.Build(confOptions)
	return
}

type Config struct {
	Port        int             `json:"port,omitempty" yaml:"port,omitempty"`
	TLS         *TLSConfig      `json:"tls,omitempty" yaml:"tls,omitempty"`
	Options     json.RawMessage `json:"options,omitempty" yaml:"options,omitempty"`
	Middlewares json.RawMessage `json:"middlewares,omitempty" yaml:"middlewares,omitempty"`
	Handlers    json.RawMessage `json:"handlers,omitempty" yaml:"handlers,omitempty"`
}

func (config *Config) GetPort() (port int, err error) {
	port = config.Port
	if port == 0 {
		if config.TLS == nil {
			port = 80
		} else {
			port = 443
		}
	}
	if port < 1 || port > 65535 {
		err = errors.Warning("port is invalid, port must great than 1024 or less than 65536")
		return
	}
	return
}

func (config *Config) GetTLS() (tls ssl.Config, err error) {
	if config.TLS == nil {
		return
	}
	tls, err = config.TLS.Config()
	if err != nil {
		err = errors.Warning("tls is invalid").WithCause(err)
		return
	}
	return
}

func (config *Config) OptionsConfig() (options configures.Config, err error) {
	if len(config.Options) == 0 {
		config.Options = []byte{'{', '}'}
	}
	options, err = configures.NewJsonConfig(config.Options)
	if err != nil {
		err = errors.Warning("options is invalid").WithCause(err)
		return
	}
	return
}

func (config *Config) MiddlewareConfig(name string) (middleware configures.Config, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		err = errors.Warning("middleware is invalid").WithCause(fmt.Errorf("name is nil")).WithMeta("middleware", name)
		return
	}
	if len(config.Middlewares) == 0 {
		config.Middlewares = []byte{'{', '}'}
	}
	middlewares, middlewaresErr := configures.NewJsonConfig(config.Middlewares)
	if middlewaresErr != nil {
		err = errors.Warning("middleware is invalid").WithCause(middlewaresErr).WithMeta("middleware", name)
		return
	}
	has := false
	middleware, has = middlewares.Node(name)
	if !has {
		middleware, _ = configures.NewJsonConfig([]byte{'{', '}'})
	}
	return
}

func (config *Config) HandlerConfig(name string) (handler configures.Config, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		err = errors.Warning("middleware is invalid").WithCause(fmt.Errorf("name is nil")).WithMeta("middleware", name)
		return
	}
	if len(config.Handlers) == 0 {
		config.Handlers = []byte{'{', '}'}
	}
	handlers, handlersErr := configures.NewJsonConfig(config.Handlers)
	if handlersErr != nil {
		err = errors.Warning("middleware is invalid").WithCause(handlersErr).WithMeta("middleware", name)
		return
	}
	has := false
	handler, has = handlers.Node(name)
	if !has {
		handler, _ = configures.NewJsonConfig([]byte{'{', '}'})
	}
	return
}
