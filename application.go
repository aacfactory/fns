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
	sc "context"
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/valyala/fasthttp"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	B = 1 << (10 * iota)
	KB
	MB
	GB
	TB
	PB
	EB
)

type Application interface {
	Deploy(service ...Service)
	Run(ctx sc.Context) (err error)
	Sync()
	SyncWithTimeout(timeout time.Duration)
}

// +-------------------------------------------------------------------------------------------------------------------+

func New(options ...Option) (app Application, err error) {
	opt := defaultOptions
	if options != nil {
		for _, option := range options {
			optErr := option(opt)
			if optErr != nil {
				err = optErr
				return
			}
		}
	}

	configRetriever, configRetrieverErr := configuares.NewRetriever(opt.ConfigRetrieverOption)
	if configRetrieverErr != nil {
		err = configRetrieverErr
		return
	}

	config, configErr := configRetriever.Get()
	if configErr != nil {
		err = configErr
		return
	}

	appConfig := ApplicationConfig{}
	mappingErr := config.As(&appConfig)
	if mappingErr != nil {
		err = mappingErr
		return
	}

	// name
	name := strings.TrimSpace(appConfig.Name)
	if name == "" {
		err = fmt.Errorf("fns create failed, no name in config")
		return
	}

	// logs
	logFormatter := logs.ConsoleFormatter
	logFormatterValue := strings.ToLower(strings.TrimSpace(appConfig.Log.Formatter))
	if logFormatterValue == "json" {
		logFormatter = logs.JsonFormatter
	}
	logLevel := logs.InfoLevel
	logLevelValue := strings.ToLower(strings.TrimSpace(appConfig.Log.Level))
	if logLevelValue == "debug" {
		logLevel = logs.DebugLevel
	} else if logLevelValue == "info" {
		logLevel = logs.InfoLevel
	} else if logLevelValue == "warn" {
		logLevel = logs.WarnLevel
	} else if logLevelValue == "error" {
		logLevel = logs.ErrorLevel
	}

	log, logErr := logs.New(
		logs.WithFormatter(logFormatter),
		logs.Name(name),
		logs.WithLevel(logLevel),
		logs.Writer(os.Stdout),
		logs.Color(appConfig.Log.Color),
	)

	if logErr != nil {
		err = logErr
		return
	}

	app0 := &application{
		id:         UID(),
		name:       name,
		version:    opt.Version,
		address:    "",
		running:    0,
		config:     config,
		log:        log,
		serviceMap: make(map[string]Service),
		svc:        nil,
		ln:         nil,
		server:     nil,
	}

	// build
	buildErr := app0.build(appConfig)
	if buildErr != nil {
		err = buildErr
		return
	}

	// succeed
	app = app0

	return
}

type application struct {
	id         string
	name       string
	version    string
	address    string
	running    int64
	config     configuares.Config
	log        logs.Logger
	serviceMap map[string]Service
	svc        Services
	ln         net.Listener
	server     *fasthttp.Server
}

func (app *application) Deploy(services ...Service) {
	if services == nil || len(services) == 0 {
		return
	}
	for _, service := range services {
		if service == nil {
			continue
		}
		_, has := app.serviceMap[service.Namespace()]
		if has {
			panic(fmt.Sprintf("fns deploy service failed for service %s is duplicated", service.Namespace()))
			return
		}
		app.serviceMap[service.Namespace()] = service
	}
	return
}

func (app *application) Run(ctx sc.Context) (err error) {

	// build services
	for _, service := range app.serviceMap {
		serviceErr := service.Build(app.config)
		if serviceErr != nil {
			err = fmt.Errorf("fns Run: build %s service failed, %v", service.Namespace(), serviceErr)
			return
		}
	}
	// serve http
	serveErr := app.serve()
	if serveErr != nil {
		err = serveErr
		err = fmt.Errorf("fns Run: start http server failed, %v", serveErr)
		return
	}
	// mount
	for _, service := range app.serviceMap {
		mountErr := app.svc.Mount(service)
		if mountErr != nil {
			err = fmt.Errorf("fns Run: mount %s service failed, %v", service.Namespace(), mountErr)
			app.stop(10 * time.Second)
			return
		}
		delete(app.serviceMap, service.Namespace())
	}
	atomic.StoreInt64(&app.running, int64(1))

	return
}

func (app *application) build(config ApplicationConfig) (err error) {

	err = app.buildListener(config)
	if err != nil {
		return
	}

	err = app.buildHttpServer(config)
	if err != nil {
		return
	}

	err = app.buildServices(config)
	if err != nil {
		return
	}

	return
}

func (app *application) buildServices(_config ApplicationConfig) (err error) {

	config := _config.Services
	config.serverId = app.id
	config.address = app.address
	config.version = app.version

	svc := &services{}

	buildErr := svc.Build(config)
	if buildErr != nil {
		err = buildErr
		return
	}

	app.svc = svc

	return
}

func (app *application) buildListener(_config ApplicationConfig) (err error) {
	// config
	httpConfig := _config.Http

	serverHost := strings.TrimSpace(httpConfig.Host)
	if serverHost == "" {
		serverHost = "0.0.0.0"
	}
	serverPort := httpConfig.Port
	if serverPort <= 0 {
		serverPort = 80
	}
	if serverPort < 1 || serverPort > 65535 {
		err = fmt.Errorf("fns get http config failed for bad port, %v", serverPort)
		return
	}
	serverAddr := fmt.Sprintf("%s:%d", serverHost, serverPort)

	ln, lnErr := net.Listen("tcp", serverAddr)
	if lnErr != nil {
		err = fmt.Errorf("fns build http server failed, %v", lnErr)
		return
	}

	app.ln = ln
	app.address = serverAddr

	return
}

func (app *application) buildHttpServer(_config ApplicationConfig) (err error) {
	// config
	config := _config.Http
	concurrency := _config.Services.Concurrency
	reduceMemoryUsage := _config.Services.ReduceMemoryUsage
	maxIdleTimeSecond := _config.Services.MaxIdleTimeSecond

	// server
	app.server = &fasthttp.Server{
		Handler:        fasthttp.CompressHandler(app.handleHttpRequest),
		ReadBufferSize: 64 * KB,
		ErrorHandler: func(ctx *fasthttp.RequestCtx, err error) {
			ctx.ResetBody()
			ctx.SetStatusCode(555)
			p, _ := json.Marshal(errors.New(555, "***NON EXHAUSTIVE***", err.Error()))
			ctx.SetBody(p)
		},
		ContinueHandler:                    nil,
		Name:                               "FNS",
		Concurrency:                        concurrency,
		IdleTimeout:                        time.Duration(maxIdleTimeSecond) * time.Second,
		MaxConnsPerIP:                      config.MaxConnectionsPerIP,
		MaxRequestsPerConn:                 config.MaxRequestsPerConnection,
		TCPKeepalive:                       config.KeepAlive,
		TCPKeepalivePeriod:                 time.Duration(config.KeepalivePeriodSecond) * time.Second,
		ReduceMemoryUsage:                  reduceMemoryUsage,
		DisablePreParseMultipartForm:       true,
		SleepWhenConcurrencyLimitsExceeded: 0,
		NoDefaultDate:                      true,
		NoDefaultContentType:               true,
		ReadTimeout:                        30 * time.Second,
	}

	return
}

func (app *application) serve() (err error) {

	errCh := make(chan error, 1)
	ctx, cancel := sc.WithTimeout(sc.TODO(), 3*time.Second)

	go func(a *application, errCh chan error) {
		serveErr := a.server.Serve(a.ln)
		if serveErr != nil {
			errCh <- fmt.Errorf("fns http serve failed, %v", serveErr)
			close(errCh)
			a.stop(10 * time.Second)
		}
	}(app, errCh)

	select {
	case <-ctx.Done():
		cancel()
		return
	case serveErr := <-errCh:
		cancel()
		err = serveErr
		return
	}
}

func parseHttpRequestURL(uri []byte) (namespace string, fn string, ok bool) {
	// todo
	return
}

func (app *application) handleHttpRequest(request *fasthttp.RequestCtx) {
	// todo white list NOT ACCEPTED

	// todo timeout
}

func (app *application) Sync() {
	app.SyncWithTimeout(10 * time.Second)
	return
}

func (app *application) SyncWithTimeout(timeout time.Duration) {

	ch := make(chan os.Signal, 1)
	signal.Notify(ch,
		os.Interrupt,
		syscall.SIGINT,
		os.Kill,
		syscall.SIGKILL,
		syscall.SIGTERM,
	)
	app.stop(timeout)
	return
}

func (app *application) stop(timeout time.Duration) {
	if timeout < 10*time.Second {
		timeout = 10 * time.Second
	}
	cancelCTX, cancel := sc.WithTimeout(sc.TODO(), timeout)
	closeCh := make(chan struct{}, 1)
	go func(ctx sc.Context, app *application, closeCh chan struct{}) {
		atomic.StoreInt64(&app.running, int64(0))
		// unmount services
		app.svc.Close()

		// http close
		_ = app.server.Shutdown()

		closeCh <- struct{}{}
		close(closeCh)
	}(cancelCTX, app, closeCh)
	select {
	case <-closeCh:
		cancel()
		break
	case <-cancelCTX.Done():
		cancel()
		break
	}
}
