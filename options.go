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
	"github.com/aacfactory/fns/cluster"
	"github.com/aacfactory/fns/internal/configure"
	"github.com/aacfactory/fns/internal/secret"
	"github.com/aacfactory/fns/listeners"
	"github.com/aacfactory/fns/server"
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
		version:                   "0.0.0",
		autoMaxProcsMin:           0,
		autoMaxProcsMax:           0,
		configRetrieverOption:     configure.DefaultConfigRetrieverOption(),
		barrier:                   nil,
		server:                    &server.FastHttp{},
		serverInterceptorHandlers: make([]server.InterceptorHandler, 0, 1),
		extraListeners:            make([]listeners.Listener, 0, 1),
		shutdownTimeout:           60 * time.Second,
	}
)

type Options struct {
	version                   string
	autoMaxProcsMin           int
	autoMaxProcsMax           int
	configRetrieverOption     configures.RetrieverOption
	barrier                   service.Barrier
	server                    server.Http
	serverInterceptorHandlers []server.InterceptorHandler
	clientBuilder             cluster.ClientBuilder
	extraListeners            []listeners.Listener
	shutdownTimeout           time.Duration
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

func Version(version string) Option {
	return func(options *Options) error {
		version = strings.TrimSpace(version)
		if version == "" {
			return fmt.Errorf("set version failed for empty")
		}
		options.version = version
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func SecretKey(data string) Option {
	return func(options *Options) error {
		data = strings.TrimSpace(data)
		if data == "" {
			return fmt.Errorf("set secret key failed for empty data")
		}
		secret.Key([]byte(data))
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

func Barrier(barrier service.Barrier) Option {
	return func(options *Options) error {
		if barrier == nil {
			return fmt.Errorf("customize barrier failed for nil")
		}
		options.barrier = barrier
		return nil
	}
}

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

func Server(server server.Http) Option {
	return func(options *Options) error {
		if server == nil {
			return fmt.Errorf("customize http failed for server is nil")
		}
		options.server = server
		return nil
	}
}

func InterceptorHandlers(handlers ...server.InterceptorHandler) Option {
	return func(options *Options) error {
		if handlers == nil || len(handlers) == 0 {
			return nil
		}
		options.serverInterceptorHandlers = append(options.serverInterceptorHandlers, handlers...)
		return nil
	}
}

func ClusterClientBuilder(builder cluster.ClientBuilder) Option {
	return func(options *Options) error {
		if builder == nil {
			return fmt.Errorf("customize cluster client failed for builder is nil")
		}
		options.clientBuilder = builder
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func ExtraListeners(lns ...listeners.Listener) Option {
	return func(options *Options) error {
		if lns == nil || len(lns) == 0 {
			return nil
		}
		for _, ln := range lns {
			if ln == nil {
				return fmt.Errorf("add extra listener failed for one of them is nil")
			}
			lnName := strings.TrimSpace(ln.Name())
			if lnName == "" {
				return fmt.Errorf("add extra listener failed for one of them has no name")
			}
			for _, listener := range options.extraListeners {
				if listener.Name() == lnName {
					return fmt.Errorf("add extra listener failed for one of them has same name")
				}
			}
			options.extraListeners = append(options.extraListeners, ln)
		}
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func MAXPROCS(min int, max int) Option {
	return func(options *Options) error {
		if min < 1 {
			min = 1
		}
		if max < 1 {
			max = 0
		}
		options.autoMaxProcsMin = min
		options.autoMaxProcsMax = max
		return nil
	}
}
