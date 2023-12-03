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

package tests

import (
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/commons/switchs"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/configs"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/fns/transports/fast"
	"github.com/aacfactory/workers"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type machine struct {
	rt      *runtime.Runtime
	manager services.EndpointsManager
}

func (m *machine) Shutdown() {
	ctx := context.TODO()
	m.manager.Shutdown(ctx)
}

var (
	app *machine = nil
)

type Options struct {
	deps                  []services.Service
	config                *configs.Config
	configRetrieverOption configures.RetrieverOption
	transport             transports.Transport
}

type Option func(options *Options) (err error)

func WithDependence(dep ...services.Service) Option {
	return func(options *Options) (err error) {
		options.deps = append(options.deps, dep...)
		return
	}
}

func WithConfig(config *configs.Config) Option {
	return func(options *Options) (err error) {
		options.config = config
		return
	}
}

func WithConfigRetriever(path string, format string, active string, prefix string, splitter byte) Option {
	return func(options *Options) (err error) {
		path = strings.TrimSpace(path)
		if path == "" {
			err = fmt.Errorf("path is empty")
			return
		}
		active = strings.TrimSpace(active)
		format = strings.ToUpper(strings.TrimSpace(format))
		store := configures.NewFileStore(path, prefix, splitter)
		options.configRetrieverOption = configures.RetrieverOption{
			Active: active,
			Format: format,
			Store:  store,
		}
		return
	}
}

func WithTransport(transport transports.Transport) Option {
	return func(options *Options) (err error) {
		options.transport = transport
		return
	}
}

func getConfigDir(src string) (dir string, err error) {
	if src == "" {
		err = errors.Warning("config dir is not found")
		return
	}
	if !filepath.IsAbs(src) {
		src, err = filepath.Abs(src)
		if err != nil {
			return
		}
	}
	dirs, readErr := os.ReadDir(src)
	if readErr != nil {
		err = readErr
		return
	}
	for _, entry := range dirs {
		if entry.IsDir() && entry.Name() == "configs" {
			dir = filepath.Join(src, "configs")
			return
		}
	}
	parentDir := filepath.Dir(src)
	if parentDir == src {
		err = errors.Warning("config dir is not found")
		return
	}
	dir, err = getConfigDir(parentDir)
	return
}

// Setup
// use local config
func Setup(service services.Service, options ...Option) (err error) {
	opt := Options{
		deps:                  nil,
		configRetrieverOption: configures.RetrieverOption{},
		transport:             fast.New(),
	}
	for _, option := range options {
		err = option(&opt)
		if err != nil {
			err = errors.Warning("fns: setup testing failed").WithCause(err)
			return
		}
	}
	if service == nil {
		err = errors.Warning("fns: setup testing failed").WithCause(fmt.Errorf("service is nil"))
		return
	}
	appId := "tests"
	appVersion := versions.Origin()
	// config
	config := opt.config
	if config == nil {
		if reflect.ValueOf(opt.configRetrieverOption).IsZero() {
			configDir, configDirErr := getConfigDir(".")
			if configDirErr != nil {
				err = errors.Warning("fns: setup testing failed").WithCause(configDirErr)
				return
			}
			opt.configRetrieverOption = configures.RetrieverOption{
				Active: "local",
				Format: "YAML",
				Store:  configures.NewFileStore(configDir, "fns", '-'),
			}
		}
		configRetriever, configRetrieverErr := configures.NewRetriever(opt.configRetrieverOption)
		if configRetrieverErr != nil {
			err = errors.Warning("fns: setup testing failed").WithCause(configRetrieverErr)
			return
		}
		configure, configureErr := configRetriever.Get()
		if configureErr != nil {
			err = errors.Warning("fns: setup testing failed").WithCause(configureErr)
			return
		}
		config = &configs.Config{}
		configErr := configure.As(config)
		if configErr != nil {
			err = errors.Warning("fns: setup testing failed").WithCause(configErr)
			return
		}
	}

	// log
	logger, loggerErr := logs.New("tests", config.Log)
	if loggerErr != nil {
		err = errors.Warning("fns: setup testing failed").WithCause(loggerErr)
		return
	}
	// worker
	workerOptions := make([]workers.Option, 0, 1)
	if workersMax := config.Runtime.Workers.Max; workersMax > 0 {
		workerOptions = append(workerOptions, workers.MaxWorkers(workersMax))
	}
	if workersMaxIdleSeconds := config.Runtime.Workers.MaxIdleSeconds; workersMaxIdleSeconds > 0 {
		workerOptions = append(workerOptions, workers.MaxIdleWorkerDuration(time.Duration(workersMaxIdleSeconds)*time.Second))
	}
	worker := workers.New(workerOptions...)
	// status
	status := &switchs.Switch{}
	// manager
	var manager services.EndpointsManager

	local := services.New(appId, appVersion, logger.With("fns", "endpoints"), config.Services, worker)

	// barrier
	var barrier barriers.Barrier
	// shared
	var shared shareds.Shared

	// cluster
	if clusterConfig := config.Cluster; clusterConfig.Name != "" {
		transport := opt.transport
		transportErr := transport.Construct(transports.Options{
			Log:    logger.With("transport", transport.Name()),
			Config: config.Transport,
			Handler: transports.HandlerFunc(func(writer transports.ResponseWriter, request transports.Request) {
			}),
		})
		if transportErr != nil {
			err = errors.Warning("fns: setup testing failed").WithCause(transportErr)
			return
		}
		var clusterErr error
		manager, shared, barrier, _, clusterErr = clusters.New(clusters.Options{
			Id:      appId,
			Version: appVersion,
			Port:    transport.Port(),
			Log:     logger.With("fns", "cluster"),
			Worker:  worker,
			Local:   local,
			Dialer:  opt.transport,
			Config:  clusterConfig,
		})
		if clusterErr != nil {
			err = errors.Warning("fns: setup testing failed").WithCause(clusterErr)
			return
		}
	} else {
		var sharedErr error
		shared, sharedErr = shareds.Local(logger.With("shared", "local"), config.Runtime.Shared)
		if sharedErr != nil {
			err = errors.Warning("fns: setup testing failed").WithCause(sharedErr)
			return
		}
		barrier = barriers.New()
		manager = local
	}

	addErr := manager.Add(service)
	if addErr != nil {
		err = errors.Warning("fns: setup testing failed").WithCause(addErr)
		return
	}
	for _, dep := range opt.deps {
		depErr := manager.Add(dep)
		if depErr != nil {
			err = errors.Warning("fns: setup testing failed").WithCause(depErr)
			return
		}
	}

	// runtime
	rt := runtime.New(
		appId, "tests", appVersion,
		status, logger, worker,
		manager,
		barrier, shared,
	)

	app = &machine{
		rt:      rt,
		manager: manager,
	}

	return
}

func Teardown() {
	if app != nil {
		app.Shutdown()
	}
}

func TODO() context.Context {
	return runtime.With(context.TODO(), app.rt)
}
