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
	"bytes"
	sc "context"
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/go-playground/validator/v10"
	"github.com/valyala/fasthttp"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
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
	Log() (log logs.Logger)
	Deploy(service ...Service) (err error)
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

	secretKey = opt.SecretKey

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

	if opt.Version != "" && opt.Version != defaultVersion {
		log = log.With("ver", opt.Version)
	}

	// timeout
	handleTimeout := 30 * time.Second
	if appConfig.Services.HandleTimeoutSecond > 0 {
		handleTimeout = time.Duration(appConfig.Services.HandleTimeoutSecond) * time.Second
	}

	// validate
	validate := opt.Validate
	if validate == nil {
		validate = validator.New()
		commons.ValidateRegisterRegex(validate)
	}

	appConfig.Services.concurrency = appConfig.Concurrency

	app0 := &application{
		id:              UID(),
		name:            name,
		version:         opt.Version,
		address:         "",
		publicAddress:   "",
		running:         0,
		config:          config,
		log:             log,
		validate:        validate,
		serviceMap:      make(map[string]Service),
		svc:             nil,
		fnHandleTimeout: handleTimeout,
		requestCounter:  sync.WaitGroup{},
		ln:              nil,
		server:          nil,
		hasHook:         false,
		hookUnitCh:      nil,
		hookStopCh:      nil,
		hooks:           opt.Hooks,
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
	id              string
	name            string
	version         string
	address         string
	publicAddress   string
	running         int64
	config          configuares.Config
	log             logs.Logger
	validate        *validator.Validate
	serviceMap      map[string]Service
	svc             *services
	fnHandleTimeout time.Duration
	requestCounter  sync.WaitGroup
	ln              net.Listener
	server          *fasthttp.Server
	hasHook         bool
	hookUnitPool    sync.Pool
	hookUnitCh      chan *HookUnit
	hookStopCh      chan chan struct{}
	hooks           []Hook
}

func (app *application) Log() (log logs.Logger) {
	log = app.log
	return
}

func (app *application) Deploy(services ...Service) (err error) {
	if services == nil || len(services) == 0 {
		return
	}
	for _, service := range services {
		if service == nil {
			continue
		}
		_, has := app.serviceMap[service.Namespace()]
		if has {
			err = fmt.Errorf("fns Deploy: service %s is duplicated", service.Namespace())
			return
		}
		app.serviceMap[service.Namespace()] = service
		if app.Log().DebugEnabled() {
			app.Log().Debug().Message(fmt.Sprintf("fns Deploy: deploy %s service succeed", service.Namespace()))
		}
	}
	return
}

func (app *application) Run(ctx sc.Context) (err error) {

	// build services
	if len(app.serviceMap) == 0 {
		err = fmt.Errorf("fns Run: no services")
		return
	}
	for _, service := range app.serviceMap {
		serviceErr := service.Build(app.config)
		if serviceErr != nil {
			err = fmt.Errorf("fns Run: build %s service failed, %v", service.Namespace(), serviceErr)
			return
		}
		if app.Log().DebugEnabled() {
			app.Log().Debug().Message(fmt.Sprintf("fns Run: build %s service succeed", service.Namespace()))
		}
	}
	// serve http
	serveErr := app.serve(ctx)
	if serveErr != nil {
		err = serveErr
		err = fmt.Errorf("fns Run: start http server failed, %v", serveErr)
		return
	}
	if app.Log().DebugEnabled() {
		app.Log().Debug().Message(fmt.Sprintf("fns Run: listen %s succeed", app.address))
	}
	// mount
	for _, service := range app.serviceMap {
		mountErr := app.svc.Mount(service)
		if mountErr != nil {
			err = fmt.Errorf("fns Run: mount %s service failed, %v", service.Namespace(), mountErr)
			app.stop(10 * time.Second)
			return
		}
		if app.Log().DebugEnabled() {
			app.Log().Debug().Message(fmt.Sprintf("fns Run: mount %s service succeed", service.Namespace()))
		}
		delete(app.serviceMap, service.Namespace())
	}

	atomic.StoreInt64(&app.running, int64(1))

	if app.Log().DebugEnabled() {
		app.Log().Debug().Message("fns Run: succeed")
	}

	return
}

func (app *application) build(config ApplicationConfig) (err error) {

	err = app.mountHooks()
	if err != nil {
		return
	}

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
	config.address = app.publicAddress
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

	// public address
	publicHost := strings.TrimSpace(httpConfig.PublicHost)
	if publicHost == "" {
		// from env
		publicHost, _ = getPublicHostFromEnv()
		// from hostname
		if publicHost == "" {
			publicHost, _ = getPublicHostFromHostname()
		}
	}
	if publicHost != "" {
		publicPort := httpConfig.PublicPort
		if publicPort == 0 {
			publicPort, _ = getPublicPortFromEnv()
			if publicPort == 0 {
				publicPort = serverPort
			}
		}
		if publicPort < 1 || publicPort > 65535 {
			err = fmt.Errorf("invalid public port, %v", publicPort)
			return
		}
		app.publicAddress = fmt.Sprintf("%s:%d", publicHost, publicPort)
	}

	return
}

func (app *application) buildHttpServer(_config ApplicationConfig) (err error) {
	// config
	config := _config.Http
	concurrency := _config.Concurrency
	reduceMemoryUsage := _config.Services.ReduceMemoryUsage

	// server
	requestHandler := fasthttp.CompressHandler(app.handleHttpRequest)
	if config.Cors.Enabled {
		config.Cors.fill()
		requestHandler = newCors(config.Cors).handler(requestHandler)
	}
	app.server = &fasthttp.Server{
		Handler:        requestHandler,
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
		MaxConnsPerIP:                      config.MaxConnectionsPerIP,
		MaxRequestsPerConn:                 config.MaxRequestsPerConnection,
		TCPKeepalive:                       config.KeepAlive,
		TCPKeepalivePeriod:                 time.Duration(config.KeepalivePeriodSecond) * time.Second,
		ReduceMemoryUsage:                  reduceMemoryUsage,
		DisablePreParseMultipartForm:       true,
		SleepWhenConcurrencyLimitsExceeded: 0,
		NoDefaultDate:                      true,
		NoDefaultContentType:               true,
		ReadTimeout:                        10 * time.Second,
	}

	return
}

func (app *application) serve(ctx sc.Context) (err error) {

	errCh := make(chan error, 1)
	var cancel sc.CancelFunc

	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		ctx, cancel = sc.WithTimeout(ctx, 1*time.Second)
	} else {
		ctx, cancel = sc.WithCancel(ctx)
	}

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
	case serveErr := <-errCh:
		err = serveErr
	}
	cancel()
	return
}

func (app *application) handleHttpRequest(request *fasthttp.RequestCtx) {
	if atomic.LoadInt64(&app.running) != int64(1) {
		sendError(request, errors.New(555, "***WARNING***", "fns is not ready or closing"))
		return
	}
	if request.IsGet() {
		uri := request.URI()
		if uri == nil || uri.Path() == nil || len(uri.Path()) == 0 {
			sendError(request, errors.New(555, "***WARNING***", "uri is invalid"))
			return
		}
		p := uri.Path()
		// health check
		if string(p) == healthCheckPath {
			request.SetStatusCode(200)
			request.SetContentTypeBytes(jsonUTF8ContentType)
			request.SetBody([]byte(fmt.Sprintf("{\"name\": \"%s\", \"version\": \"%s\"}", app.name, app.version)))
			return
		}
		// description
		items := bytes.Split(p[1:], pathSplitter)
		if len(items) != 2 {
			sendError(request, errors.New(555, "***WARNING***", "uri is invalid"))
			return
		}
		namespace := string(items[0])
		if app.svc.IsInternal(namespace) {
			sendError(request, errors.New(555, "***WARNING***", "can not access an internal service"))
			return
		}
		if string(items[1]) == descriptionPathItem {
			description := app.svc.Description(namespace)
			if description == nil || len(description) == 0 {
				request.SetStatusCode(200)
				request.SetContentTypeBytes(jsonUTF8ContentType)
				request.SetBody(emptyBody)
			} else {
				request.SetStatusCode(200)
				request.SetContentTypeBytes(jsonUTF8ContentType)
				request.SetBody(description)
			}
			return
		} else {
			sendError(request, errors.New(555, "***WARNING***", "uri is invalid"))
			return
		}
	} else if request.IsPost() {
		app.requestCounter.Add(1)
		defer app.requestCounter.Done()

		uri := request.URI()
		if uri == nil || uri.Path() == nil || len(uri.Path()) == 0 {
			sendError(request, errors.New(555, "***WARNING***", "uri is invalid"))
			return
		}
		p := uri.Path()
		items := bytes.Split(p[1:], pathSplitter)
		if len(items) != 2 {
			sendError(request, errors.New(555, "***WARNING***", "uri is invalid"))
			return
		}
		namespace := string(items[0])
		if !app.svc.Exist(namespace) {
			sendError(request, errors.NotFound(fmt.Sprintf("%s was not found", namespace)))
			return
		}

		// body
		arg, argErr := NewArgument(request.PostBody())
		if argErr != nil {
			sendError(request, errors.BadRequest("request body must be json content"))
			return
		}

		// user
		var requestUser User
		authorization := request.Request.Header.PeekBytes(authorizationHeader)
		if authorization != nil && len(authorization) > 0 {
			requestUser = newUser(authorization, app.svc.Authorizations(), app.svc.Permissions())
		} else {
			requestUser = newUser(nil, app.svc.Authorizations(), app.svc.Permissions())
		}

		// ctx
		timeoutCtx, cancel := sc.WithTimeout(sc.TODO(), app.fnHandleTimeout)
		var ctx *context
		requestId := request.Request.Header.PeekBytes(requestIdHeader)
		if requestId != nil && len(requestId) > 0 {
			metaValue := request.Request.Header.PeekBytes(requestMetaHeader)
			if metaValue == nil || len(metaValue) == 0 {
				sendError(request, errors.New(555, "***WARNING***", "meta is required in internal request"))
				cancel()
				return
			}
			ctx = newContext(timeoutCtx, string(requestId), requestUser)
			if !ctx.Meta().Decode(metaValue) {
				sendError(request, errors.New(555, "***WARNING***", "meta is invalid in internal request"))
				cancel()
				return
			}
		} else {
			if app.svc.IsInternal(namespace) {
				sendError(request, errors.New(555, "***WARNING***", "can not access an internal service"))
				cancel()
				return
			}
			ctx = newContext(timeoutCtx, UID(), requestUser)
		}
		// ctx app
		ctx.app = &appRuntime{
			publicAddress: app.publicAddress,
			log:           app.log,
			validate:      app.validate,
		}

		// fn
		fn := string(items[1])

		// request
		handleBeg := time.Now()
		result := app.svc.Request(ctx, namespace, fn, arg)
		latency := time.Now().Sub(handleBeg)
		data := json.RawMessage{}
		handleErr := result.Get(ctx.Context, &data)
		if handleErr != nil {
			sendError(request, handleErr)
			cancel()
			return
		}
		request.SetStatusCode(200)
		request.SetContentTypeBytes(jsonUTF8ContentType)
		request.Response.Header.SetBytesK(requestIdHeader, ctx.RequestId())
		request.Response.Header.SetBytesK(responseLatencyHeader, latency.String())
		request.SetBody(data)

		// hook
		if app.hasHook {
			unit := app.hookUnitPool.Get().(*HookUnit)
			unit.Namespace = namespace
			unit.FnName = fn
			unit.RequestSize = int64(len(request.PostBody()))
			unit.ResponseSize = int64(len(data))
			unit.Latency = latency
			unit.Error = handleErr
			app.hookUnitCh <- unit
		}

		cancel()
	} else {
		sendError(request, errors.New(555, "***WARNING***", "method is invalid"))
		return
	}

}

var (
	jsonUTF8ContentType = []byte("application/json;charset=utf-8")

	emptyBody = []byte("{}")

	healthCheckPath = "/health"

	descriptionPathItem = "description"

	pathSplitter = []byte("/")

	authorizationHeader = []byte("Authorization")

	requestIdHeader = []byte("X-Fns-Request-Id")

	requestMetaHeader = []byte("X-Fns-Meta")

	responseLatencyHeader = []byte("X-Fns-Latency")
)

func sendError(request *fasthttp.RequestCtx, err errors.CodeError) {
	body, _ := json.Marshal(err)
	request.SetStatusCode(err.Code())
	request.SetContentTypeBytes(jsonUTF8ContentType)
	request.SetBody(body)
}

func (app *application) mountHooks() (err error) {
	if app.hooks == nil {
		return
	}
	for _, hook := range app.hooks {
		if hook == nil {
			continue
		}
		buildErr := hook.Build(app.config)
		if buildErr != nil {
			err = fmt.Errorf("fns build hook failed, %v", buildErr)
			return
		}
	}
	app.hookUnitPool.New = func() interface{} {
		return &HookUnit{}
	}
	app.hookUnitCh = make(chan *HookUnit, 256*1024)
	app.hookStopCh = make(chan chan struct{}, 1)
	app.hasHook = true
	go func(units *sync.Pool, ch chan *HookUnit, stop chan chan struct{}, hooks []Hook) {
		for {
			var stopCallbackCh chan struct{}
			stopped := false
			select {
			case stopCallbackCh = <-stop:
				stopped = true
				break
			case unit, ok := <-ch:
				if !ok {
					stopped = true
					break
				}
				for _, hook := range hooks {
					hook.Handle(*unit)
				}
				units.Put(unit)
			}
			if stopped {
				for _, hook := range hooks {
					hook.Close()
				}
				stopCallbackCh <- struct{}{}
				break
			}
		}
	}(&app.hookUnitPool, app.hookUnitCh, app.hookStopCh, app.hooks)
	return
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
	<-ch

	app.stop(timeout)
	return
}

func (app *application) stop(timeout time.Duration) {
	if atomic.LoadInt64(&app.running) == int64(0) {
		return
	}
	atomic.StoreInt64(&app.running, 0)
	if timeout < 10*time.Second {
		timeout = 10 * time.Second
	}
	cancelCTX, cancel := sc.WithTimeout(sc.TODO(), timeout)
	closeCh := make(chan struct{}, 1)
	go func(ctx sc.Context, app *application, closeCh chan struct{}) {
		// wait remain requests
		app.requestCounter.Wait()
		if app.Log().DebugEnabled() {
			app.Log().Debug().Message("fns Close: wait for the remaining requests to be processed successfully")
		}
		// unmount services
		app.svc.Close()
		if app.Log().DebugEnabled() {
			app.Log().Debug().Message("fns Close: services close successfully")
		}

		// http close
		_ = app.ln.Close()
		if app.Log().DebugEnabled() {
			app.Log().Debug().Message("fns Close: http server close successfully")
		}

		// hooks
		if app.hasHook {
			hookStopCallBackCh := make(chan struct{}, 1)
			app.hookStopCh <- hookStopCallBackCh
			<-hookStopCallBackCh
		}
		if app.Log().DebugEnabled() {
			app.Log().Debug().Message("fns Close: hooks close successfully")
		}

		closeCh <- struct{}{}
		close(closeCh)
	}(cancelCTX, app, closeCh)
	select {
	case <-closeCh:
		break
	case <-cancelCTX.Done():
		break
	}
	cancel()
}
