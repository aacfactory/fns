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
	"context"
	"crypto/tls"
	"fmt"
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
	Run(ctx context.Context) (err error)
	Sync()
	SyncWithTimeout(timeout time.Duration)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Option func(*Options) error

var (
	defaultOptions = &Options{
		Config:         defaultConfigRetrieverOption,
		RequestHandler: newMappedServiceRequestHandler(),
	}
)

type Options struct {
	Config                   ConfigRetrieverOption
	Log                      Logs
	HttpRequestConfigBuilder HttpRequestConfigBuilder
	RequestHandler           ServiceRequestHandler
	AuthorizationValidator   AuthorizationValidator
	PermissionValidator      PermissionValidator
}

func CustomizeLog(logs Logs) Option {
	return func(o *Options) error {
		if logs == nil {
			return fmt.Errorf("fns create failed, customize log is nil")
		}
		o.Log = logs
		return nil
	}
}

func FileConfig(path string, format string, active string) Option {
	return func(o *Options) error {
		path = strings.TrimSpace(path)
		if path == "" {
			return fmt.Errorf("fns create file config failed, path is empty")
		}
		active = strings.TrimSpace(active)
		format = strings.ToUpper(strings.TrimSpace(format))
		store := NewConfigFileStore(path)
		o.Config = ConfigRetrieverOption{
			Active: active,
			Format: format,
			Store:  store,
		}
		return nil
	}
}

func CustomizeHttpRequestConfigBuilder(builder HttpRequestConfigBuilder) Option {
	return func(o *Options) error {
		if builder == nil {
			return fmt.Errorf("fns create failed, customize http request config builder is nil")
		}
		o.HttpRequestConfigBuilder = builder
		return nil
	}
}

func CustomizeServiceRequestHandler(requestHandler ServiceRequestHandler) Option {
	return func(o *Options) error {
		if requestHandler == nil {
			return fmt.Errorf("fns create failed, customize service request handler is nil")
		}
		o.RequestHandler = requestHandler
		return nil
	}
}

func CustomizeAuthorizationValidator(validator AuthorizationValidator) Option {
	return func(o *Options) error {
		if validator == nil {
			return fmt.Errorf("fns create failed, customize authorization validator is nil")
		}
		o.AuthorizationValidator = validator
		return nil
	}
}

func CustomizePermissionValidator(validator PermissionValidator) Option {
	return func(o *Options) error {
		if validator == nil {
			return fmt.Errorf("fns create failed, customize permission validator is nil")
		}
		o.PermissionValidator = validator
		return nil
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func New(options ...Option) (a Application, err error) {
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

	configRetriever, configRetrieverErr := NewConfigRetriever(opt.Config)
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
	a = app0

	return
}

type app struct {
	name                   string
	running                int64
	config                 Config
	log                    Logs
	discovery              Discovery
	serviceRegistrations   map[string]string
	serviceMap             map[string]Service
	ln                     net.Listener
	server                 *fasthttp.Server
	requestConfigBuilder   HttpRequestConfigBuilder
	requestHandler         ServiceRequestHandler
	authorizationValidator AuthorizationValidator
	permissionValidator    PermissionValidator
}

func (a *app) Deploy(services ...Service) {
	if services == nil || len(services) == 0 {
		return
	}
	for _, service := range services {
		if service == nil {
			continue
		}
		_, has := a.serviceMap[service.Namespace()]
		if has {
			panic(fmt.Sprintf("fns deploy service failed for service %s is duplicated", service.Namespace()))
			return
		}
		a.serviceMap[service.Namespace()] = service
	}
	return
}

func (a *app) Run(ctx context.Context) (err error) {

	// build services
	for _, service := range a.serviceMap {
		serviceErr := service.Build(ctx, a.config, a.log)
		if serviceErr != nil {
			err = fmt.Errorf("fns build %s service failed, %v", service.Namespace(), serviceErr)
			return
		}
	}
	// serve http
	serveErr := a.serve()
	if serveErr != nil {
		err = serveErr
		return
	}
	atomic.StoreInt64(&a.running, int64(1))
	// discovery publish service
	if a.discovery != nil {
		for namespace := range a.serviceMap {
			registrationId, pubErr := a.discovery.Publish(namespace)
			if pubErr != nil {
				err = fmt.Errorf("fns publish %s service failed, %v", namespace, pubErr)
				a.stop(10 * time.Second)
				return
			}
			a.serviceRegistrations[namespace] = registrationId
		}
	}

	return
}

func (a *app) build() (err error) {
	err = a.buildDiscovery()
	if err != nil {
		return
	}
	err = a.buildListener()
	if err != nil {
		return
	}
	err = a.buildHttpServer()
	if err != nil {
		return
	}
	return
}

func (a *app) buildDiscovery() (err error) {
	// config
	discoveryConfig := DiscoveryConfig{}
	hasDiscovery, discoveryConfigErr := a.config.Get("discovery", &discoveryConfig)
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
	hasHttp, httpConfigGetErr := a.config.Get("http", &httpConfig)
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

	a.discovery = discovery

	return
}

func (a *app) buildListener() (err error) {
	// config
	httpConfig := HttpConfig{}
	hasHttp, httpConfigGetErr := a.config.Get("http", &httpConfig)
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

	a.ln = ln
	return
}

func (a *app) buildHttpServer() (err error) {
	// config
	httpConfig := HttpConfig{}
	hasHttp, httpConfigGetErr := a.config.Get("http", &httpConfig)
	if !hasHttp {
		err = fmt.Errorf("fns get http config failed, http was not found")
		return
	}
	if httpConfigGetErr != nil {
		err = fmt.Errorf("fns get http config failed, %v", httpConfigGetErr)
		return
	}

	workConfig := WorkConfig{}
	hasWork, workConfigGetErr := a.config.Get("work", &workConfig)
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
	a.server = &fasthttp.Server{
		Handler:      fasthttp.CompressHandler(a.handleHttpRequest),
		ReadBufferSize: 64 * KB,
		ErrorHandler: nil,
		HeaderReceived: func(header *fasthttp.RequestHeader) (requestConfig fasthttp.RequestConfig) {
			contentType := string(header.ContentType())
			namespace := string(header.Peek(httpHeaderNamespace))
			name := string(header.Peek(httpHeaderFnName))
			config := a.requestConfigBuilder.Build(contentType, namespace, name)
			requestConfig.MaxRequestBodySize = config.RequestBodyMaxSize
			requestConfig.ReadTimeout = config.ReadTimeout
			requestConfig.WriteTimeout = config.WriteTimeout
			return
		},
		ContinueHandler: nil,
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

func (a *app) serve() (err error) {

	errCh := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)

	go func(a *app, errCh chan error) {
		serveErr := a.server.Serve(a.ln)
		if serveErr != nil {
			errCh <- fmt.Errorf("fns http serve failed, %v", serveErr)
			close(errCh)
			a.stop(10 * time.Second)
		}
	}(a, errCh)
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

func (a *app) handleHttpRequest(request *fasthttp.RequestCtx) {



}

func (a *app) Sync() {
	a.SyncWithTimeout(10 * time.Second)
	return
}

func (a *app) SyncWithTimeout(timeout time.Duration) {

	ch := make(chan os.Signal, 1)
	signal.Notify(ch,
		os.Interrupt,
		syscall.SIGINT,
		os.Kill,
		syscall.SIGKILL,
		syscall.SIGTERM,
	)
	a.stop(timeout)
	return
}

func (a *app) stop(timeout time.Duration) {
	if timeout < 10*time.Second {
		timeout = 10 * time.Second
	}
	cancelCTX, cancel := context.WithTimeout(context.TODO(), timeout)
	closeCh := make(chan struct{}, 1)
	go func(ctx context.Context, a *app, closeCh chan struct{}) {
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
	}(cancelCTX, a, closeCh)
	select {
	case <-closeCh:
		cancel()
		break
	case <-cancelCTX.Done():
		cancel()
		break
	}
}
