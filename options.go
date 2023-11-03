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
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/configs"
	"github.com/aacfactory/fns/hooks"
	"github.com/aacfactory/fns/proxies"
	"github.com/aacfactory/fns/services/validators"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/fns/transports/fast"
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
		configRetrieverOption: configs.DefaultConfigRetrieverOption(),
		transport:             fast.New(),
		middlewares:           make([]transports.Middleware, 0, 1),
		handlers:              make([]transports.MuxHandler, 0, 1),
		hooks:                 nil,
		shutdownTimeout:       60 * time.Second,
		proxyOptions:          make([]proxies.Option, 0, 1),
	}
)

// +-------------------------------------------------------------------------------------------------------------------+

type Options struct {
	id                    string
	name                  string
	version               versions.Version
	configRetrieverOption configures.RetrieverOption
	transport             transports.Transport
	middlewares           []transports.Middleware
	handlers              []transports.MuxHandler
	hooks                 []hooks.Hook
	shutdownTimeout       time.Duration
	proxyOptions          []proxies.Option
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
			return fmt.Errorf("customize id failed for empty")
		}
		options.id = id
		return nil
	}
}

func Name(name string) Option {
	return func(options *Options) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("customize name failed for empty")
		}
		options.name = name
		return nil
	}
}

func Version(version string) Option {
	return func(options *Options) error {
		version = strings.TrimSpace(version)
		if version == "" {
			return fmt.Errorf("customize version failed for empty")
		}
		ver, parseErr := versions.Parse(bytex.FromString(version))
		if parseErr != nil {
			return parseErr
		}
		options.version = ver
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

func Hooks(h ...hooks.Hook) Option {
	return func(options *Options) error {
		if h == nil || len(h) == 0 {
			return fmt.Errorf("customize hooks failed for nil")
		}
		if options.hooks == nil {
			options.hooks = make([]hooks.Hook, 0, 1)
		}
		for _, hook := range h {
			if hook == nil {
				continue
			}
			options.hooks = append(options.hooks, hook)
		}
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func ShutdownTimeout(timeout time.Duration) Option {
	return func(options *Options) error {
		if timeout < 1 {
			return fmt.Errorf("customize application shutdown timeout failed for nil")
		}
		options.shutdownTimeout = timeout
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func Transport(transport transports.Transport) Option {
	return func(options *Options) error {
		options.transport = transport
		return nil
	}
}

func Middleware(middleware transports.Middleware) Option {
	return func(options *Options) error {
		options.middlewares = append(options.middlewares, middleware)
		return nil
	}
}

func Handler(handler transports.MuxHandler) Option {
	return func(options *Options) error {
		options.handlers = append(options.handlers, handler)
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func Proxy(options ...proxies.Option) Option {
	return func(opts *Options) error {
		opts.proxyOptions = append(opts.proxyOptions, options...)
		return nil
	}
}
