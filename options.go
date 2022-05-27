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
	"github.com/aacfactory/workers"
	"os"
	"strings"
	"sync"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

type Option func(*Options) error

var (
	defaultOptions = &Options{
		procs: &procsOption{
			min: 0,
			max: 0,
		},
		documents: &Documents{
			Title:         "FNS",
			Description:   "",
			Terms:         "",
			Contact:       Contact{},
			License:       License{},
			Version:       "0.0.0",
			Services:      make(map[string]*ServiceDocument),
			URL:           "",
			once:          sync.Once{},
			oasRAW:        nil,
			convertOasErr: nil,
			raw:           nil,
			encodeErr:     nil,
		},
		configRetrieverOption: defaultConfigRetrieverOption(),
		concurrency:           workers.DefaultConcurrency,
		workerMaxIdleTime:     workers.DefaultMaxIdleTime,
		serviceRequestTimeout: 10 * time.Second,
		barrier:               defaultBarrier(),
		validator:             defaultValidator(),
		tracerReporter:        defaultTracerReporter(),
		hooks:                 make([]Hook, 0, 1),
		server:                &fastHttp{},
		httpVersion:           "HTTP/1.1",
		httpHandlerWrappers:   make([]HttpHandlerWrapper, 0, 1),
		websocketDiscovery:    &memoryWebsocketDiscovery{},
		shutdownTimeout:       60 * time.Second,
	}
	secretKey = []byte("+-fns")
)

type Options struct {
	procs                 *procsOption
	documents             *Documents
	configRetrieverOption configuares.RetrieverOption
	concurrency           int
	workerMaxIdleTime     time.Duration
	serviceRequestTimeout time.Duration
	barrier               Barrier
	validator             Validator
	tracerReporter        TracerReporter
	hooks                 []Hook
	server                HttpServer
	httpVersion           string
	httpHandlerWrappers   []HttpHandlerWrapper
	websocketDiscovery    WebsocketDiscovery
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
		store := configuares.NewFileStore(path, prefix, splitter)
		o.configRetrieverOption = configuares.RetrieverOption{
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

func Document(title string, description string, terms string, contact Contact, license License) Option {
	return func(options *Options) error {
		title = strings.TrimSpace(title)
		if title == "" {
			return fmt.Errorf("set title failed for empty")
		}
		options.documents.Title = title
		options.documents.Description = description
		options.documents.Terms = terms
		options.documents.Contact = contact
		options.documents.License = license
		return nil
	}
}

func Version(version string) Option {
	return func(options *Options) error {
		version = strings.TrimSpace(version)
		if version == "" {
			return fmt.Errorf("set version failed for empty")
		}
		options.documents.Version = version
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func Hooks(hooks ...Hook) Option {
	return func(o *Options) error {
		if hooks == nil || len(hooks) == 0 {
			return fmt.Errorf("hooks is empty")
		}
		copy(o.hooks, hooks)
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
		secretKey = []byte(data)
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func Concurrency(concurrency int) Option {
	return func(options *Options) error {
		if concurrency < 1 {
			return fmt.Errorf("set concurrency failed for empty")
		}
		options.concurrency = concurrency
		return nil
	}
}

func ServiceRequestHandleTimeout(timeout time.Duration) Option {
	return func(options *Options) error {
		if timeout < 1 {
			return fmt.Errorf("set service request timeout failed for empty")
		}
		options.serviceRequestTimeout = timeout
		return nil
	}
}

func ServiceHandlerMaxIdleTime(idle time.Duration) Option {
	return func(options *Options) error {
		if idle < 1 {
			return fmt.Errorf("set service handler max idle time failed for time is invalid")
		}
		options.workerMaxIdleTime = idle
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func CustomizeValidator(validator Validator) Option {
	return func(options *Options) error {
		if validator == nil {
			return fmt.Errorf("set validator failed for nil")
		}
		options.validator = validator
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func CustomizeBarrier(barrier Barrier) Option {
	return func(options *Options) error {
		if barrier == nil {
			return fmt.Errorf("set barrier failed for nil")
		}
		options.barrier = barrier
		return nil
	}
}

func CustomizeShutdownTimeout(timeout time.Duration) Option {
	return func(options *Options) error {
		if timeout < 1 {
			return fmt.Errorf("set application shutdown timeout failed for nil")
		}
		options.shutdownTimeout = timeout
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func CustomizeTraceReporter(reporter TracerReporter) Option {
	return func(options *Options) error {
		if reporter == nil {
			return fmt.Errorf("set tracer reporter failed for nil")
		}
		options.tracerReporter = reporter
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func CustomizeHttpServer(httpVersion string, server HttpServer) Option {
	return func(options *Options) error {
		httpVersion = strings.ToUpper(strings.TrimSpace(httpVersion))
		if httpVersion == "" {
			return fmt.Errorf("set http server failed for httpVersion is invalid")
		}
		if server == nil {
			return fmt.Errorf("set http server failed for it is nil")
		}
		options.server = server
		options.httpVersion = httpVersion
		return nil
	}
}

func AppendHttpHandlerWrapper(wrappers ...HttpHandlerWrapper) Option {
	return func(options *Options) error {
		if wrappers == nil {
			return fmt.Errorf("append http handler wrappers failed for it is nil")
		}
		for _, wrapper := range wrappers {
			if wrapper == nil {
				continue
			}
			options.httpHandlerWrappers = append(options.httpHandlerWrappers, wrapper)
		}
		return nil
	}
}

func EnableCors() Option {
	return func(options *Options) error {
		options.httpHandlerWrappers = append(options.httpHandlerWrappers, &corsHttpHandlerWrapper{})
		return nil
	}
}

func CustomizeWebsocketDiscovery(discovery WebsocketDiscovery) Option {
	return func(options *Options) error {
		if discovery == nil {
			return fmt.Errorf("set websocket discovery failed for nil")
		}
		options.websocketDiscovery = discovery
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
		options.procs.min = min
		options.procs.max = max
		return nil
	}
}
