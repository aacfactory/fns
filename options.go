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
	"github.com/aacfactory/fns/cluster"
	"github.com/aacfactory/fns/documents"
	"github.com/aacfactory/fns/internal/secret"
	"os"
	"strings"
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
		document:                   documents.New(),
		configRetrieverOption:      defaultConfigRetrieverOption(),
		workerMaxIdleTime:          60 * time.Second,
		handleRequestTimeout:       10 * time.Second,
		barrier:                    defaultBarrier(),
		validator:                  defaultValidator(),
		tracerReporter:             defaultTracerReporter(),
		hooks:                      make([]Hook, 0, 1),
		serverBuilder:              fastHttpBuilder,
		clientBuilder:              fastHttpClientBuilder,
		httpHandlerWrapperBuilders: defaultHttpHandlerWrapperBuilders(),
		shutdownTimeout:            60 * time.Second,
	}
)

type Options struct {
	procs                      *procsOption
	document                   *documents.Application
	configRetrieverOption      configuares.RetrieverOption
	workerMaxIdleTime          time.Duration
	handleRequestTimeout       time.Duration
	barrier                    Barrier
	validator                  Validator
	tracerReporter             TracerReporter
	hooks                      []Hook
	serverBuilder              HttpServerBuilder
	clientBuilder              cluster.ClientBuilder
	httpHandlerWrapperBuilders []HttpHandlerWrapperBuilder
	shutdownTimeout            time.Duration
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

// +-------------------------------------------------------------------------------------------------------------------+

func ConfigActiveFromENV(key string) (active string) {
	v, has := os.LookupEnv(key)
	if !has {
		return
	}
	active = strings.ToLower(strings.TrimSpace(v))
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func Title(title string) Option {
	return func(options *Options) error {
		title = strings.TrimSpace(title)
		if title == "" {
			return fmt.Errorf("title is empty")
		}
		options.document.Title = title
		return nil
	}
}

func Description(description string) Option {
	return func(options *Options) error {
		description = strings.TrimSpace(description)
		if description == "" {
			return fmt.Errorf("description is empty")
		}
		options.document.Description = description
		return nil
	}
}
func Terms(terms string) Option {
	return func(options *Options) error {
		terms = strings.TrimSpace(terms)
		if terms == "" {
			return fmt.Errorf("terms is empty")
		}
		options.document.Terms = terms
		return nil
	}
}

func Contact(name string, email string, url string) Option {
	return func(options *Options) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("name is empty")
		}
		email = strings.TrimSpace(email)
		if email == "" {
			return fmt.Errorf("email is empty")
		}
		url = strings.TrimSpace(url)
		if url == "" {
			return fmt.Errorf("url is empty")
		}
		options.document.Contact.Name = name
		options.document.Contact.Email = email
		options.document.Contact.Url = url
		return nil
	}
}

func License(name string, url string) Option {
	return func(options *Options) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("name is empty")
		}
		url = strings.TrimSpace(url)
		if url == "" {
			return fmt.Errorf("url is empty")
		}
		options.document.License.Name = name
		options.document.License.Url = url
		return nil
	}
}

func Version(version string) Option {
	return func(options *Options) error {
		version = strings.TrimSpace(version)
		if version == "" {
			return fmt.Errorf("set version failed for empty")
		}
		options.document.Version = version
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
		secret.Key([]byte(data))
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

func CustomizeWorkerMaxIdleTime(v time.Duration) Option {
	return func(options *Options) error {
		if v < 1 {
			v = 60 * time.Second
		}
		options.workerMaxIdleTime = v
		return nil
	}
}

func CustomizeHandleRequestTimeout(v time.Duration) Option {
	return func(options *Options) error {
		if v < 1 {
			v = 10 * time.Second
		}
		options.handleRequestTimeout = v
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func CustomizeHttp(server HttpServerBuilder) Option {
	return func(options *Options) error {
		if server == nil {
			return fmt.Errorf("customize http failed for server builder is nil")
		}
		options.serverBuilder = server
		return nil
	}
}

func CustomizeClusterClientBuilder(client cluster.ClientBuilder) Option {
	return func(options *Options) error {
		if client == nil {
			return fmt.Errorf("customize cluster client builder failed for client builder is nil")
		}
		options.clientBuilder = client
		return nil
	}
}

func AppendHttpHandlerWrapper(wrappers ...HttpHandlerWrapperBuilder) Option {
	return func(options *Options) error {
		if wrappers == nil {
			return fmt.Errorf("append http handler wrapper builder failed for it is nil")
		}
		for _, wrapper := range wrappers {
			if wrapper == nil {
				continue
			}
			options.httpHandlerWrapperBuilders = append(options.httpHandlerWrapperBuilders, wrapper)
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
		options.procs.min = min
		options.procs.max = max
		return nil
	}
}
