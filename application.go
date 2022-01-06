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
	"github.com/aacfactory/fns/secret"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
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
		id:             UID(),
		name:           name,
		version:        opt.Version,
		address:        "",
		publicAddress:  "",
		minPROCS:       opt.MinPROCS,
		maxPROCS:       opt.MaxPROCS,
		running:        0,
		config:         config,
		log:            log,
		validate:       validate,
		serviceMap:     make(map[string]Service),
		svc:            nil,
		requestCounter: sync.WaitGroup{},
		ln:             nil,
		server:         nil,
		hasHook:        false,
		hookUnitCh:     nil,
		hookStopCh:     nil,
		hooks:          opt.Hooks,
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
	id             string
	name           string
	version        string
	address        string
	publicAddress  string
	minPROCS       int
	maxPROCS       int
	undoMAXPROCS   func()
	running        int64
	config         configuares.Config
	log            logs.Logger
	validate       *validator.Validate
	serviceMap     map[string]Service
	svc            *services
	docSync        singleflight.Group
	requestCounter sync.WaitGroup
	ln             net.Listener
	server         *fasthttp.Server
	hasHook        bool
	hookUnitCh     chan *HookUnit
	hookStopCh     chan chan struct{}
	hooks          []Hook
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
	// GOMAXPROCS
	app.setGOMAXPROCS()
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

	svc := newServices(app.id, app.version, app.publicAddress, _config.Concurrency, app.log, app.validate)

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

func (app *application) getDocuments() (p []byte) {
	b, err, _ := app.docSync.Do("doc", func() (v interface{}, err error) {
		documents := make(map[string]*ServiceDocument)
		registrations := app.svc.discovery.Registrations()
		for _, registration := range registrations {
			_, loaded := documents[registration.Namespace]
			if loaded {
				continue
			}
			if app.svc.discovery.IsLocal(registration.Namespace) {
				doc, hasDoc := app.svc.Documents()[registration.Namespace]
				if !hasDoc {
					continue
				}
				documents[registration.Namespace] = doc
			} else {
				result := app.svc.RemoteRequest(false, "", nil, nil, registration.Namespace, "_document", EmptyArgument())
				doc := &ServiceDocument{}
				handleErr := result.Get(sc.TODO(), &doc)
				if handleErr != nil {
					continue
				}
				documents[registration.Namespace] = doc
			}
		}
		return
	})
	if err != nil {
		app.docSync.Forget("doc")
		doc := json.NewObject()
		_ = doc.Put("swagger", "2.0")
		info := json.NewObject()
		_ = info.Put("version", "x.y.z")
		_ = info.Put("title", "fns")
		_ = info.Put("description", fmt.Sprintf("Failed: %s", err.Error()))
		_ = doc.Put("info", info)
		paths := json.NewObject()
		healthPath := json.NewObject()
		healthPathGet := json.NewObject()
		_ = healthPathGet.Put("summary", "服务状态健康检查")
		_ = healthPathGet.Put("description", "服务状态健康检查")
		_ = healthPathGet.Put("operationId", "_health")
		_ = healthPathGet.Put("produces", []string{"application/json"})
		healthPathGetResponse := json.NewObject()
		healthPathGetResponse200 := json.NewObject()
		_ = healthPathGetResponse200.Put("description", "服务信息")
		healthPathGetResponse200Schema := json.NewObject()
		_ = healthPathGetResponse200Schema.Put("type", "object")
		healthPathGetResponse200SchemaProperties := json.NewObject()
		_ = healthPathGetResponse200SchemaProperties.Put("name", map[string]string{"type": "string"})
		_ = healthPathGetResponse200SchemaProperties.Put("version", map[string]string{"type": "string"})
		_ = healthPathGetResponse200Schema.Put("properties", healthPathGetResponse200SchemaProperties)
		_ = healthPathGetResponse200.Put("schema", healthPathGetResponse200Schema)
		_ = healthPathGetResponse.Put("200", healthPathGetResponse200)
		_ = healthPathGet.Put("responses", healthPathGetResponse)
		_ = healthPath.Put("get", healthPathGet)
		_ = paths.Put("/health", healthPath)
		_ = doc.Put("paths", paths)
		p, _ = doc.MarshalJSON()
		return
	}
	app.docSync.Forget("doc")
	p = b.([]byte)
	return
}

func (app *application) handleHttpRequest(request *fasthttp.RequestCtx) {
	if atomic.LoadInt64(&app.running) != int64(1) {
		sendError(request, errors.New(555, "***WARNING***", "fns Http: fns is not ready or closing"))
		return
	}
	if request.IsGet() {
		uri := request.URI()
		if uri == nil || uri.Path() == nil || len(uri.Path()) == 0 {
			sendError(request, errors.New(555, "***WARNING***", "fns Http: uri is invalid"))
			return
		}
		p := uri.Path()
		switch string(p) {
		case healthCheckPath:
			// health check
			request.SetStatusCode(200)
			request.SetContentTypeBytes(jsonUTF8ContentType)
			request.SetBody([]byte(fmt.Sprintf("{\"name\": \"%s\", \"version\": \"%s\"}", app.name, app.version)))
		case documentsPath:
			// documents
			request.SetStatusCode(200)
			request.SetContentTypeBytes(jsonUTF8ContentType)
			request.SetBody(app.getDocuments())
		default:
			sendError(request, errors.New(555, "***WARNING***", "fns Http: uri is invalid"))
			return
		}
	} else if request.IsPost() {
		app.requestCounter.Add(1)
		defer app.requestCounter.Done()

		uri := request.URI()
		if uri == nil || uri.Path() == nil || len(uri.Path()) == 0 {
			sendError(request, errors.New(555, "***WARNING***", "fns Http: uri is invalid"))
			return
		}
		p := uri.Path()
		items := bytes.Split(p[1:], pathSplitter)
		if len(items) != 2 {
			sendError(request, errors.New(555, "***WARNING***", "fns Http: uri is invalid"))
			return
		}
		namespace := string(items[0])
		if !app.svc.Exist(namespace) {
			sendError(request, errors.NotFound(fmt.Sprintf("%s was not found", namespace)))
			return
		}
		// fn
		fn := string(items[1])

		// authorization
		authorization := request.Request.Header.PeekBytes(authorizationHeader)
		if authorization == nil {
			authorization = make([]byte, 0, 1)
		}
		// requestId
		requestId := ""

		isInnerRequest := false
		contentType := request.Request.Header.ContentType()
		if contentType != nil && len(contentType) > 0 {
			isInnerRequest = bytes.Equal(fnsProxyContentType, contentType)
		}
		var arg Argument
		var argErr error
		var meta []byte
		if isInnerRequest {
			signHeader := request.Request.Header.PeekBytes(requestSignHeader)
			if signHeader == nil || len(signHeader) == 0 {
				sendError(request, errors.Warning(fmt.Sprintf("fns Http: invalid request of %s/%s failed", namespace, fn)))
				return
			}
			requestId = string(request.Request.Header.PeekBytes(requestIdHeader))
			body := request.PostBody()
			buf := bytebufferpool.Get()
			_, _ = buf.WriteString(requestId)
			_, _ = buf.Write(body)
			signedTarget := buf.Bytes()
			bytebufferpool.Put(buf)
			if !secret.Verify(signedTarget, signHeader, secretKey) {
				sendError(request, errors.Warning(fmt.Sprintf("fns Http: invalid request of %s/%s failed", namespace, fn)))
				return
			}
			metaValue, argValue := proxyMessageDecode(body)
			arg, argErr = NewArgument(argValue)
			if argErr != nil {
				sendError(request, errors.BadRequest("fns Http: request body must be json content"))
				return
			}
			meta = metaValue
		} else {
			requestId = UID()
			arg, argErr = NewArgument(request.PostBody())
			if argErr != nil {
				sendError(request, errors.BadRequest("fns Http: request body must be json content"))
				return
			}
			meta = emptyMeta
		}

		// request
		handleBeg := time.Now()
		result := app.svc.Request(isInnerRequest, requestId, meta, authorization, namespace, fn, arg)
		latency := time.Now().Sub(handleBeg)
		data := json.RawMessage{}
		handleErr := result.Get(sc.TODO(), &data)
		if handleErr != nil {
			request.Response.Header.SetBytesK(requestIdHeader, requestId)
			request.Response.Header.SetBytesK(responseLatencyHeader, latency.String())
			sendError(request, handleErr)
			return
		}
		request.SetStatusCode(200)
		request.SetContentTypeBytes(jsonUTF8ContentType)
		request.Response.Header.SetBytesK(requestIdHeader, requestId)
		request.Response.Header.SetBytesK(responseLatencyHeader, latency.String())
		if len(data) > 0 {
			request.SetBody(data)
		}

		// hook
		if app.hasHook {
			unit := &HookUnit{}
			unit.Namespace = namespace
			unit.FnName = fn
			unit.RequestId = requestId
			unit.Authorization = authorization
			unit.RequestSize = int64(len(request.PostBody()))
			unit.ResponseSize = int64(len(data))
			unit.Latency = latency
			unit.HandleError = handleErr
			app.hookUnitCh <- unit
		}
	} else {
		sendError(request, errors.New(555, "***WARNING***", "fns Http: method is invalid"))
		return
	}

}

var (
	jsonUTF8ContentType = []byte("application/json;charset=utf-8")

	emptyBody = []byte("{}")
	emptyMeta = []byte("{}")

	healthCheckPath = "/health"

	documentsPath = "/_documents"

	pathSplitter = []byte("/")

	authorizationHeader = []byte("Authorization")

	requestIdHeader = []byte("X-Fns-Request-Id")

	requestSignHeader = []byte("X-Fns-Signature")

	responseLatencyHeader = []byte("X-Fns-Latency")

	nullJson = []byte("null")
)

func sendError(request *fasthttp.RequestCtx, err errors.CodeError) {
	body, _ := json.Marshal(err)
	if err.Contains(errors.NilError()) {
		request.SetStatusCode(444)
	} else {
		request.SetStatusCode(err.Code())
	}
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
