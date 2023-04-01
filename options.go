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
	"github.com/aacfactory/configures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/fns/service/validators"
	"os"
	"strings"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

type Option func(*Options) error

var (
	defaultOptions = &Options{
		id:                    "",
		name:                  "fns",
		version:               versions.New(0, 0, 1),
		proxyMode:             false,
		configRetrieverOption: service.DefaultConfigRetrieverOption(),
		transportOptions:      UseTransport(service.FastHttpTransport()),
		shutdownTimeout:       60 * time.Second,
	}
)

func UseTransport(transport service.Transport) *TransportOptions {
	return &TransportOptions{
		transport:   transport,
		handlers:    make([]service.TransportHandler, 0, 1),
		middlewares: make([]service.TransportMiddleware, 0, 1),
	}
}

type TransportOptions struct {
	transport   service.Transport
	handlers    []service.TransportHandler
	middlewares []service.TransportMiddleware
}

func (options *TransportOptions) Append(handlers ...service.TransportHandler) *TransportOptions {
	if handlers == nil || len(handlers) == 0 {
		return options
	}
	options.handlers = append(options.handlers, handlers...)
	return options
}

func (options *TransportOptions) Use(middlewares ...service.TransportMiddleware) *TransportOptions {
	if middlewares == nil || len(middlewares) == 0 {
		return options
	}
	options.middlewares = append(options.middlewares, middlewares...)
	return options
}

type Options struct {
	id                    string
	name                  string
	version               versions.Version
	proxyMode             bool
	configRetrieverOption configures.RetrieverOption
	transportOptions      *TransportOptions
	shutdownTimeout       time.Duration
}

// +-------------------------------------------------------------------------------------------------------------------+

func ConfigRetriever(path string, format string, active string, prefix string, splitter byte) Option {
	return func(o *Options) error {
		path = strings.TrimSpace(path)
		if path == "" {
			return fmt.Errorf("path is empty")
		}
		active = strings.TrimSpace(active)
		format = strings.ToUpper(strings.TrimSpace(format))
		store := configures.NewFileStore(path, prefix, splitter)
		o.configRetrieverOption = configures.RetrieverOption{
			Active: active,
			Format: format,
			Store:  store,
		}
		return nil
	}
}

func ConfigActiveFromENV(key string) (active string) {
	v, has := os.LookupEnv(key)
	if !has {
		return
	}
	active = strings.ToLower(strings.TrimSpace(v))
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func Id(id string) Option {
	return func(options *Options) error {
		id = strings.TrimSpace(id)
		if id == "" {
			return fmt.Errorf("set id failed for empty")
		}
		options.id = id
		return nil
	}
}

func Name(name string) Option {
	return func(options *Options) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("set name failed for empty")
		}
		options.name = name
		return nil
	}
}

func Version(version string) Option {
	return func(options *Options) error {
		version = strings.TrimSpace(version)
		if version == "" {
			return fmt.Errorf("set version failed for empty")
		}
		ver, parseErr := versions.Parse(version)
		if parseErr != nil {
			return parseErr
		}
		options.version = ver
		return nil
	}
}

func ProxyMode() Option {
	return func(options *Options) error {
		options.proxyMode = true
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func RegisterValidator(register validators.ValidateRegister) Option {
	return func(options *Options) error {
		if register == nil {
			return fmt.Errorf("customize validator failed for nil")
		}
		validators.AddValidateRegister(register)
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func ShutdownTimeout(timeout time.Duration) Option {
	return func(options *Options) error {
		if timeout < 1 {
			return fmt.Errorf("set application shutdown timeout failed for nil")
		}
		options.shutdownTimeout = timeout
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func Transport(tr *TransportOptions) Option {
	return func(options *Options) error {
		if tr == nil || tr.transport == nil {
			return fmt.Errorf("customize transport failed for it is nil")
		}
		if tr.transport.Name() == "" {
			return fmt.Errorf("customize transport failed for name of transport is nil")
		}
		options.transportOptions = tr
		return nil
	}
}

func Handlers(handlers ...service.TransportHandler) Option {
	return func(options *Options) error {
		if handlers == nil || len(handlers) == 0 {
			return nil
		}
		options.transportOptions.Append(handlers...)
		return nil
	}
}

func Middlewares(middlewares ...service.TransportMiddleware) Option {
	return func(options *Options) error {
		if middlewares == nil || len(middlewares) == 0 {
			return nil
		}
		options.transportOptions.Use(middlewares...)
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+
