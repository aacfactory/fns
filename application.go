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
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configuares"
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

	configRetriever, configRetrieverErr := NewConfigRetriever(opt.ConfigRetrieverOption)
	if configRetrieverErr != nil {
		err = configRetrieverErr
		return
	}

	config, configErr := configRetriever.Get()
	if configErr != nil {
		err = configErr
		return
	}

	appConfig := &ApplicationConfig{}
	mappingErr := config.As(appConfig)
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
	var log Logs
	if opt.Log == nil {
		log = newLogs(name, appConfig.Log)
	} else {
		log = opt.Log
	}

	app0 := &app{
		name:                   name,
		running:                0,
		config:                 config,
		log:                    log,
		serviceRegistrations:   make(map[string]string),
		serviceMap:             make(map[string]Service),
		requestConfigBuilder:   opt.HttpRequestConfigBuilder,
		requestHandler:         opt.RequestHandler,
		authorizationValidator: opt.AuthorizationValidator,
		permissionValidator:    opt.PermissionValidator,
	}

	// build
	buildErr := app0.build()
	if buildErr != nil {
		err = buildErr
		return
	}

	// succeed
	app = app0

	return
}

type application struct {
	name                   string
	running                int64
	config                 configuares.Config
	log                    logs.Logger
	serviceCenter          ServiceCenter
	ln                     net.Listener
	server                 *fasthttp.Server
	requestConfigBuilder   HttpRequestConfigBuilder
	requestHandler         ServiceRequestHandler
	authorizationValidator AuthorizationValidator
	permissionValidator    PermissionValidator
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
		serviceErr := service.Build(ctx, app.config, app.log)
		if serviceErr != nil {
			err = fmt.Errorf("fns build %s service failed, %v", service.Namespace(), serviceErr)
			return
		}
	}
	// serve http
	serveErr := app.serve()
	if serveErr != nil {
		err = serveErr
		return
	}
	atomic.StoreInt64(&app.running, int64(1))
	// discovery publish service
	if app.discovery != nil {
		for namespace := range app.serviceMap {
			registrationId, pubErr := app.discovery.Publish(namespace)
			if pubErr != nil {
				err = fmt.Errorf("fns publish %s service failed, %v", namespace, pubErr)
				app.stop(10 * time.Second)
				return
			}
			app.serviceRegistrations[namespace] = registrationId
		}
	}

	return
}

func (app *application) build() (err error) {
	err = app.buildDiscovery()
	if err != nil {
		return
	}
	err = app.buildListener()
	if err != nil {
		return
	}
	err = app.buildHttpServer()
	if err != nil {
		return
	}
	return
}

func (app *application) buildDiscovery() (err error) {
	// config
	discoveryConfig := DiscoveryConfig{}
	hasDiscovery, discoveryConfigErr := app.config.Get("discovery", &discoveryConfig)
	if !hasDiscovery {
		return
	}
	if discoveryConfigErr != nil {
		err = fmt.Errorf("fns get discovery config failed, %v", discoveryConfigErr)
		return
	}

	if !discoveryConfig.Enable {
		return
	}

	retriever, hasRetriever := discoveryRetrieverMap[strings.TrimSpace(discoveryConfig.Kind)]
	if !hasRetriever || retriever == nil {
		err = fmt.Errorf("fns build discovery failed for %s kind retriever was not found", discoveryConfig.Kind)
		return
	}

	httpConfig := HttpConfig{}
	hasHttp, httpConfigGetErr := app.config.Get("http", &httpConfig)
	if !hasHttp {
		err = fmt.Errorf("fns get http config failed, http was not found")
		return
	}
	if httpConfigGetErr != nil {
		err = fmt.Errorf("fns get http config failed, %v", httpConfigGetErr)
		return
	}

	serverPublicHost := strings.TrimSpace(httpConfig.PublicHost)
	if serverPublicHost == "" {
		serverHost := strings.TrimSpace(httpConfig.Host)
		if serverHost == "" {
			hostnameIp, getIpErr := IpFromHostname(false)
			if getIpErr != nil {
				err = fmt.Errorf("fns get http config failed for can not get ipv4 from hostname, %v", getIpErr)
				return
			}
			serverPublicHost = hostnameIp
		} else {
			serverPublicHost = serverHost
		}
	}
	serverPublicPort := httpConfig.PublicPort
	if serverPublicPort <= 0 {
		serverPort := httpConfig.Port
		if serverPort <= 0 {
			serverPort = 80
		}
		serverPublicPort = serverPort
	}
	if serverPublicPort < 1 || serverPublicPort > 65535 {
		err = fmt.Errorf("fns get http config failed for bad public port, %v", serverPublicPort)
		return
	}
	serverPublicAddr := fmt.Sprintf("%s:%d", serverPublicHost, serverPublicPort)

	if httpConfig.SSL.Enable && httpConfig.SSL.Client.Enable {
		_, clientTLSErr := httpConfig.SSL.Client.Config()
		if clientTLSErr != nil {
			err = fmt.Errorf("fns get http config failed for client ssl, %v", clientTLSErr)
			return
		}
	}

	discovery, buildErr := retriever(DiscoveryOption{
		Address:   serverPublicAddr,
		ClientTLS: httpConfig.SSL.Client,
		Config:    discoveryConfig.Config,
	})

	if buildErr != nil {
		err = fmt.Errorf("fns build discovery failed, %v", buildErr)
		return
	}

	app.discovery = discovery

	return
}

func (app *application) buildListener() (err error) {
	// config
	httpConfig := HttpConfig{}
	hasHttp, httpConfigGetErr := app.config.Get("http", &httpConfig)
	if !hasHttp {
		err = fmt.Errorf("fns get http config failed, http was not found")
		return
	}
	if httpConfigGetErr != nil {
		err = fmt.Errorf("fns get http config failed, %v", httpConfigGetErr)
		return
	}
	var serverTLS *tls.Config
	if httpConfig.SSL.Enable {
		serverTLS, err = httpConfig.SSL.Config()
		if err != nil {
			err = fmt.Errorf("fns get http config failed for server ssl, %v", err)
			return
		}
		if httpConfig.SSL.Client.Enable {
			_, clientTLSErr := httpConfig.SSL.Client.Config()
			if clientTLSErr != nil {
				err = fmt.Errorf("fns get http config failed for client ssl, %v", clientTLSErr)
				return
			}
		}
	}
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

	var ln net.Listener
	if serverTLS != nil {
		ln, err = tls.Listen("tcp", serverAddr, serverTLS)
	} else {
		ln, err = net.Listen("tcp", serverAddr)
	}
	if err != nil {
		err = fmt.Errorf("fns build http server failed, %v", err)
		return
	}

	app.ln = ln
	return
}

func (app *application) buildHttpServer() (err error) {
	// config
	httpConfig := HttpConfig{}
	hasHttp, httpConfigGetErr := app.config.Get("http", &httpConfig)
	if !hasHttp {
		err = fmt.Errorf("fns get http config failed, http was not found")
		return
	}
	if httpConfigGetErr != nil {
		err = fmt.Errorf("fns get http config failed, %v", httpConfigGetErr)
		return
	}

	workConfig := WorkConfig{}
	hasWork, workConfigGetErr := app.config.Get("work", &workConfig)
	if !hasWork {
		err = fmt.Errorf("fns get work config failed, work was not found")
		return
	}
	if workConfigGetErr != nil {
		err = fmt.Errorf("fns get work config failed, %v", workConfigGetErr)
		return
	}
	// server
	fasthttp.TimeoutHandler()
	app.server = &fasthttp.Server{
		Handler:        fasthttp.CompressHandler(app.handleHttpRequest),
		ReadBufferSize: 64 * KB,
		ErrorHandler:   nil,
		HeaderReceived: func(header *fasthttp.RequestHeader) (requestConfig fasthttp.RequestConfig) {
			contentType := string(header.ContentType())
			namespace := string(header.Peek(httpHeaderNamespace))
			name := string(header.Peek(httpHeaderFnName))
			config := app.requestConfigBuilder.Build(contentType, namespace, name)
			requestConfig.MaxRequestBodySize = config.RequestBodyMaxSize
			requestConfig.ReadTimeout = config.ReadTimeout
			requestConfig.WriteTimeout = config.WriteTimeout
			return
		},
		ContinueHandler:                    nil,
		Name:                               "FNS",
		Concurrency:                        workConfig.Concurrency,
		IdleTimeout:                        time.Duration(workConfig.MaxIdleTimeSecond) * time.Second,
		MaxConnsPerIP:                      httpConfig.MaxConnectionsPerIP,
		MaxRequestsPerConn:                 httpConfig.MaxRequestsPerConnection,
		TCPKeepalive:                       httpConfig.KeepAlive,
		TCPKeepalivePeriod:                 time.Duration(httpConfig.KeepalivePeriodSecond) * time.Second,
		ReduceMemoryUsage:                  workConfig.ReduceMemoryUsage,
		DisablePreParseMultipartForm:       true,
		SleepWhenConcurrencyLimitsExceeded: 0,
		NoDefaultDate:                      true,
		NoDefaultContentType:               true,
	}
	return
}

func (app *application) serve() (err error) {

	errCh := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)

	go func(a *app, errCh chan error) {
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

func (app *application) handleHttpRequest(request *fasthttp.RequestCtx) {

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
	go func(ctx sc.Context, a *application, closeCh chan struct{}) {
		atomic.StoreInt64(&a.running, int64(0))
		// un publish
		if a.discovery != nil {
			for _, registrationId := range a.serviceRegistrations {
				_ = a.discovery.UnPublish(registrationId)
			}
			a.discovery.Close()
		}
		// http close
		_ = a.server.Shutdown()
		// server close
		for _, service := range a.serviceMap {
			_ = service.Close(ctx)
		}
		// log sync
		_ = a.log.Sync()
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
