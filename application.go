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
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons"
	"github.com/aacfactory/fns/secret"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/fasthttp/router"
	"github.com/fasthttp/websocket"
	"github.com/go-playground/validator/v10"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/sync/singleflight"
	"net"
	"os"
	"os/signal"
	"runtime"
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

var (
	jsonUTF8ContentType   = []byte("application/json;charset=utf-8")
	emptyJson             = []byte("{}")
	healthCheckPath       = "/health"
	documentsPath         = "/_documents"
	documentsOASPath      = "/_documents.json"
	pathSplitter          = []byte("/")
	authorizationHeader   = []byte("Authorization")
	requestIdHeader       = []byte("X-Fns-Request-Id")
	requestSignHeader     = []byte("X-Fns-Signature")
	responseLatencyHeader = []byte("X-Fns-Latency")
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
	// description
	description := strings.TrimSpace(appConfig.Description)
	// terms
	terms := strings.TrimSpace(appConfig.Terms)
	// contact
	contact := appConfig.Contact
	// license
	license := appConfig.License

	// concurrency
	concurrency := appConfig.Concurrency
	if concurrency < 1024 {
		concurrency = 256 * 1024
	}
	appConfig.Concurrency = concurrency

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

	// validate
	validate := opt.Validate
	if validate == nil {
		validate = validator.New()
		commons.ValidateRegisterRegex(validate)
		commons.ValidateRegisterNotBlank(validate)
		commons.ValidateRegisterNotEmpty(validate)
		commons.ValidateRegisterDefault(validate)
	}

	app0 := &application{
		id:                UID(),
		name:              name,
		description:       description,
		terms:             terms,
		contact:           contact,
		license:           license,
		version:           opt.Version,
		address:           "",
		publicAddress:     "",
		https:             false,
		minPROCS:          opt.MinPROCS,
		maxPROCS:          opt.MaxPROCS,
		running:           0,
		config:            config,
		log:               log,
		validate:          validate,
		svc:               nil,
		requestCounter:    sync.WaitGroup{},
		ln:                nil,
		server:            nil,
		websocketUpgrader: nil,
		hasHook:           false,
		hookUnitCh:        nil,
		hookStopCh:        nil,
		hooks:             opt.Hooks,
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

type appContact struct {
	name  string
	url   string
	email string
}

type appLicense struct {
	name string
	url  string
}

type application struct {
	id                   string
	name                 string
	description          string
	terms                string
	contact              *appContact
	license              *appLicense
	version              string
	address              string
	publicAddress        string
	https                bool
	minPROCS             int
	maxPROCS             int
	undoMAXPROCS         func()
	running              int64
	config               configuares.Config
	log                  logs.Logger
	validate             *validator.Validate
	svc                  *services
	docSync              singleflight.Group
	requestCounter       sync.WaitGroup
	ln                   net.Listener
	server               *fasthttp.Server
	websocketUpgrader    *websocket.FastHTTPUpgrader
	websocketConnections WebsocketConnections
	hasHook              bool
	hookUnitCh           chan *HookUnit
	hookStopCh           chan chan struct{}
	hooks                []Hook
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
		if app.svc.Exist(service.Namespace()) {
			err = fmt.Errorf("fns Deploy: service %s is duplicated", service.Namespace())
			return
		}
		config, _ := app.config.Node(service.Namespace())
		ctx, _ := newContext(sc.TODO(), true, "-", []byte(""), nil, app.svc.app)
		buildErr := service.Build(ctx, config)
		if buildErr != nil {
			err = fmt.Errorf("fns Deploy: service %s build failed, %v", service.Namespace(), buildErr)
			return
		}
		mountErr := app.svc.Mount(service)
		if mountErr != nil {
			err = fmt.Errorf("fns Deploy: mount %s service failed, %v", service.Namespace(), mountErr)
			app.stop(10 * time.Second)
			return
		}
		if app.Log().DebugEnabled() {
			app.Log().Debug().Message(fmt.Sprintf("fns Deploy: mount %s service succeed", service.Namespace()))
		}
	}
	return
}

func (app *application) setGOMAXPROCS() {
	if app.maxPROCS == 0 {
		maxprocsLog := &printf{
			core: app.Log(),
		}
		undo, setErr := maxprocs.Set(maxprocs.Min(app.minPROCS), maxprocs.Logger(maxprocsLog.Printf))
		if setErr != nil {
			if app.Log().DebugEnabled() {
				app.log.Debug().Message("fns Run: set automaxprocs failed, use runtime.GOMAXPROCS(0) insteadof")
			}
			runtime.GOMAXPROCS(0)
			return
		}
		app.undoMAXPROCS = undo
		return
	}
	runtime.GOMAXPROCS(app.maxPROCS)
}

func (app *application) Run(ctx sc.Context) (err error) {

	// build services
	if len(app.svc.items) == 0 {
		err = fmt.Errorf("fns Run: no services")
		return
	}
	// GOMAXPROCS
	app.setGOMAXPROCS()

	// services
	runServiceErr := app.svc.Run()
	if runServiceErr != nil {
		err = fmt.Errorf("fns Run: run services failed, %v", runServiceErr)
		app.stop(10 * time.Second)
		return
	}

	// http
	serveErr := app.serve(ctx)
	if serveErr != nil {
		err = serveErr
		err = fmt.Errorf("fns Run: start http server failed, %v", serveErr)
		return
	}
	if app.Log().DebugEnabled() {
		app.Log().Debug().Message(fmt.Sprintf("fns Run: listen %s succeed", app.address))
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

	err = app.buildPublicAddr(config)
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

func (app *application) buildPublicAddr(_config ApplicationConfig) (err error) {
	config := _config.Http
	// public address
	publicHost := strings.TrimSpace(config.PublicHost)
	if publicHost == "" {
		// from env
		publicHost, _ = getPublicHostFromEnv()
		// from hostname
		if publicHost == "" {
			publicHost, _ = getPublicHostFromHostname()
		}
	}
	if publicHost != "" {
		publicPort := config.PublicPort
		if publicPort == 0 {
			publicPort, _ = getPublicPortFromEnv()
			if publicPort == 0 {
				serverPort := config.Port
				if serverPort <= 0 {
					serverPort = 80
				}
				if serverPort < 1 || serverPort > 65535 {
					err = fmt.Errorf("http port is invalid, %v", serverPort)
					return
				}
				publicPort = serverPort
			}
		}
		if publicPort < 1 || publicPort > 65535 {
			err = fmt.Errorf("http public port is invalid, %v", publicPort)
			return
		}
		app.publicAddress = fmt.Sprintf("%s:%d", publicHost, publicPort)
	}
	return
}

func (app *application) buildServices(_config ApplicationConfig) (err error) {
	config := _config.Services
	svc := newServices(app, _config.Concurrency)
	buildErr := svc.Build(config)
	if buildErr != nil {
		err = buildErr
		return
	}
	app.svc = svc
	return
}

func (app *application) buildHttpServer(_config ApplicationConfig) (err error) {
	// config
	config := _config.Http
	concurrency := _config.Concurrency
	reduceMemoryUsage := _config.Services.ReduceMemoryUsage

	// server
	requestHandler := app.httpRouter().Handler
	if config.Cors.Enable {
		config.Cors.fill()
		requestHandler = newCors(config.Cors).handler(requestHandler)
	}
	// buffer size
	readBufferSize := 64 * KB
	if config.ReadBufferSize != "" {
		bs := strings.ToUpper(strings.TrimSpace(config.ReadBufferSize))
		if bs != "" {
			bs0, bsErr := commons.ToBytes(bs)
			if bsErr != nil {
				err = fmt.Errorf("fns Build: invalid http readBufferSize in config")
				return
			}
			readBufferSize = int(bs0)
		}
	}
	writeBufferSize := 4 * MB
	if config.WriteBufferSize != "" {
		bs := strings.ToUpper(strings.TrimSpace(config.WriteBufferSize))
		if bs != "" {
			bs0, bsErr := commons.ToBytes(bs)
			if bsErr != nil {
				err = fmt.Errorf("fns Build: invalid http writeBufferSize in config")
				return
			}
			writeBufferSize = int(bs0)
		}
	}
	// websocket
	upgrader, upgraderErr := config.Websocket.upgrader()
	if upgraderErr != nil {
		err = upgraderErr
		return
	}
	app.websocketUpgrader = upgrader
	app.websocketConnections = websocketConnectionsRetriever()
	// server
	app.server = &fasthttp.Server{
		Handler:         requestHandler,
		ReadBufferSize:  readBufferSize,
		WriteBufferSize: writeBufferSize,
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

func (app *application) listen() (err error) {
	// config
	config := HttpConfig{}
	has, configErr := app.config.Get("http", &config)
	if !has {
		err = fmt.Errorf("http config was not found")
		return
	}
	if configErr != nil {
		err = fmt.Errorf("get http config failed, %v", configErr)
		return
	}

	serverHost := strings.TrimSpace(config.Host)
	if serverHost == "" {
		serverHost = "0.0.0.0"
	}
	serverPort := config.Port
	if serverPort <= 0 {
		serverPort = 80
	}
	if serverPort < 1 || serverPort > 65535 {
		err = fmt.Errorf("fns get http config failed for bad port, %v", serverPort)
		return
	}
	serverAddr := fmt.Sprintf("%s:%d", serverHost, serverPort)

	if config.TLS.Enable {
		httpsConfig, httpsConfigErr := config.TLS.mapToTLS()
		if httpsConfigErr != nil {
			err = httpsConfigErr
			return
		}
		ln, lnErr := tls.Listen("tcp", serverAddr, httpsConfig)
		if lnErr != nil {
			err = fmt.Errorf("fns build http server failed, %v", lnErr)
			return
		}
		app.ln = ln
		app.https = true
	} else {
		ln, lnErr := net.Listen("tcp", serverAddr)
		if lnErr != nil {
			err = fmt.Errorf("fns build http server failed, %v", lnErr)
			return
		}
		app.ln = ln
		app.https = false
	}

	app.address = serverAddr

	return
}

func (app *application) serve(_ sc.Context) (err error) {
	errCh := make(chan error, 1)
	go func(a *application, errCh chan error) {
		lnErr := a.listen()
		if lnErr != nil {
			errCh <- fmt.Errorf("fns http serve failed, %v", lnErr)
			close(errCh)
			a.stop(1 * time.Second)
			return
		}
		serveErr := a.server.Serve(a.ln)
		if serveErr != nil {
			errCh <- fmt.Errorf("fns http serve failed, %v", serveErr)
			close(errCh)
			a.stop(1 * time.Second)
			return
		}
	}(app, errCh)
	select {
	case serveErr := <-errCh:
		err = serveErr
	case <-time.After(1 * time.Second):
		close(errCh)
	}
	return
}

func (app *application) httpRouter() (r *router.Router) {
	r = router.New()
	r.GET("/health", fasthttp.CompressHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(200)
		ctx.SetContentTypeBytes(jsonUTF8ContentType)
		ctx.SetBody([]byte(fmt.Sprintf("{\"name\": \"%s\", \"version\": \"%s\"}", app.name, app.version)))
	}))
	r.GET("/_documents", fasthttp.CompressHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(200)
		ctx.SetContentTypeBytes(jsonUTF8ContentType)
		ctx.SetBody(json.UnsafeMarshal(app.svc.doc))
	}))
	r.GET("/_documents.json", fasthttp.CompressHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(200)
		ctx.SetContentTypeBytes(jsonUTF8ContentType)
		ctx.SetBody(app.svc.doc.mapToOpenApi())
	}))
	r.GET("/websocket", func(ctx *fasthttp.RequestCtx) {
		// todo check http schema
		upgradeErr := app.websocketUpgrader.Upgrade(ctx, func(conn *websocket.Conn) {
			connection, connErr := newWebsocketConnection(conn, app.svc, app.websocketConnections)
			if connErr != nil {
				_ = conn.WriteControl(websocket.CloseInternalServerErr, []byte(connErr.Error()), time.Time{})
				_ = conn.Close()
				return
			}
			handleErr := connection.Handle()
			if handleErr != nil {
				_ = conn.WriteControl(websocket.CloseInternalServerErr, []byte(handleErr.Error()), time.Time{})
			}
			closeErr := connection.Close()
			if closeErr != nil {
				app.log.Warn().Cause(closeErr).Message("fns Http: close websocket connection failed")
				return
			}
		})
		if upgradeErr != nil {
			sendError(ctx, errors.New(555, "***WARNING***", "fns Http: upgrade the HTTP server connection to the WebSocket protocol failed").WithCause(upgradeErr))
		}
	})
	r.POST("/{service}/{fn}", fasthttp.CompressHandler(func(ctx *fasthttp.RequestCtx) {
		app.requestCounter.Add(1)
		defer app.requestCounter.Done()
		service := ctx.UserValue("service").(string)
		fn := ctx.UserValue("fn").(string)
		// authorization
		authorization := ctx.Request.Header.PeekBytes(authorizationHeader)
		if authorization == nil {
			authorization = make([]byte, 0, 1)
		}
		// requestId
		requestId := ""

		isInnerRequest := false
		contentType := ctx.Request.Header.ContentType()
		if contentType != nil && len(contentType) > 0 {
			isInnerRequest = bytes.Equal(fnsProxyContentType, contentType)
		}
		var arg Argument
		var argErr error
		var meta []byte
		if isInnerRequest {
			signHeader := ctx.Request.Header.PeekBytes(requestSignHeader)
			if signHeader == nil || len(signHeader) == 0 {
				sendError(ctx, errors.Warning(fmt.Sprintf("fns Http: invalid request of %s/%s failed", service, fn)))
				return
			}
			requestId = string(ctx.Request.Header.PeekBytes(requestIdHeader))
			body := ctx.PostBody()
			buf := bytebufferpool.Get()
			_, _ = buf.WriteString(requestId)
			_, _ = buf.Write(body)
			signedTarget := buf.Bytes()
			bytebufferpool.Put(buf)
			if !secret.Verify(signedTarget, signHeader, secretKey) {
				sendError(ctx, errors.Warning(fmt.Sprintf("fns Http: invalid request of %s/%s failed", service, fn)))
				return
			}
			metaValue, argValue := proxyMessageDecode(body)
			arg, argErr = NewArgument(argValue)
			if argErr != nil {
				sendError(ctx, errors.BadRequest("fns Http: request body must be json content"))
				return
			}
			meta = metaValue
		} else {
			requestId = UID()
			arg, argErr = NewArgument(ctx.PostBody())
			if argErr != nil {
				sendError(ctx, errors.BadRequest("fns Http: request body must be json content"))
				return
			}
			meta = emptyJson
		}

		// request
		handleBeg := time.Now()
		timeoutCtx, cancel := sc.WithTimeout(ctx, app.svc.fnHandleTimeout)
		requestCtx, ctxErr := newContext(timeoutCtx, isInnerRequest, requestId, authorization, meta, app.svc.app)
		if ctxErr != nil {
			latency := time.Now().Sub(handleBeg)
			ctx.Response.Header.SetBytesK(requestIdHeader, requestId)
			ctx.Response.Header.SetBytesK(responseLatencyHeader, latency.String())
			sendError(ctx, errors.Warning("fns Http: create context failed").WithCause(ctxErr))
			cancel()
			return
		}
		result := app.svc.Request(requestCtx, service, fn, arg)

		data := json.RawMessage{}
		handleErr := result.Get(requestCtx, &data)
		latency := time.Now().Sub(handleBeg)
		if handleErr != nil {
			ctx.Response.Header.SetBytesK(requestIdHeader, requestId)
			ctx.Response.Header.SetBytesK(responseLatencyHeader, latency.String())
			sendError(ctx, handleErr)
			cancel()
			return
		}
		cancel()
		ctx.SetStatusCode(200)
		ctx.SetContentTypeBytes(jsonUTF8ContentType)
		ctx.Response.Header.SetBytesK(requestIdHeader, requestId)
		ctx.Response.Header.SetBytesK(responseLatencyHeader, latency.String())
		if len(data) > 0 {
			ctx.SetBody(data)
		}
		// hook
		if app.hasHook {
			unit := &HookUnit{}
			unit.Service = service
			unit.FnName = fn
			unit.RequestId = requestId
			unit.Authorization = authorization
			unit.RequestSize = int64(len(ctx.PostBody()))
			unit.ResponseSize = int64(len(data))
			unit.Latency = latency
			unit.HandleError = handleErr
			app.hookUnitCh <- unit
		}

	}))
	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		sendError(ctx, errors.New(555, "***WARNING***", "fns Http: uri is invalid"))
	}
	return
}

func sendError(request *fasthttp.RequestCtx, err errors.CodeError) {
	body, encodeErr := json.Marshal(err)
	if encodeErr != nil {
		err = errors.Warning("encoding failed").WithCause(encodeErr)
		sendError(request, err)
		return
	}
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

	app.hookUnitCh = make(chan *HookUnit, 256*1024)
	app.hookStopCh = make(chan chan struct{}, 1)
	app.hasHook = true
	go func(ch chan *HookUnit, stop chan chan struct{}, hooks []Hook) {
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
			}
			if stopped {
				for _, hook := range hooks {
					hook.Close()
				}
				stopCallbackCh <- struct{}{}
				break
			}
		}
	}(app.hookUnitCh, app.hookStopCh, app.hooks)
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
	if app.undoMAXPROCS != nil {
		app.undoMAXPROCS()
	}
	cancel()
}
