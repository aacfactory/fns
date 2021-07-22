package fns

import (
	"context"
	"fmt"
	"github.com/aacfactory/cluster"
	"github.com/aacfactory/eventbus"
	"github.com/aacfactory/logs"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
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
	Sync(ctx context.Context) (err error)
	SyncWithTimeout(ctx context.Context, timeout time.Duration) (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Option func(*Options) error

var (
	defaultOptions = &Options{
		Config: defaultConfigRetrieverOption,
	}
)

type Options struct {
	Config   ConfigRetrieverOption
	Log      Logs
	Eventbus eventbus.Eventbus
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

func CustomizeEventbus(eventbus eventbus.Eventbus) Option {
	return func(o *Options) error {
		if eventbus == nil {
			return fmt.Errorf("fns create failed, customize eventbus is nil")
		}
		o.Eventbus = eventbus
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

	app0 := &app{
		config:       config,
		servicesLock: sync.Mutex{},
		services:     make([]Service, 0, 1),
	}

	// name
	name := strings.TrimSpace(appConfig.Name)
	if name == "" {
		err = fmt.Errorf("fns create failed, no name in config")
		return
	}
	app0.name = name
	// tags
	tags := appConfig.Tags
	if tags == nil {
		tags = make([]string, 0, 1)
	}
	app0.tags = tags

	// logs
	if opt.Log == nil {
		log := newLogs(name, appConfig.Log)
		app0.log = log
	} else {
		app0.log = opt.Log
	}

	// cluster
	clusterEnabled := appConfig.Cluster.Enable
	app0.clusterMode = clusterEnabled
	if clusterEnabled {
		if clusterRetriever == nil {
			err = fmt.Errorf("fns create failed, cluster mode is enabled, but cluster retriever was not registered")
			return
		}
		clusterConfig := appConfig.Cluster.Config
		if clusterConfig == nil || len(clusterConfig) <= 2 {
			err = fmt.Errorf("fns create failed, cluster mode is enabled, no config of cluster in config")
			return
		}
		c, clusterErr := clusterRetriever(name, tags, clusterConfig)
		if clusterErr != nil {
			err = fmt.Errorf("fns create failed, create cluster from retriever failed, %v", clusterErr)
			return
		}
		app0.cluster = c
	}
	// eventbus
	if opt.Eventbus == nil {
		bus, busErr := newEventBus(app0.cluster, appConfig.EventBus)
		if busErr != nil {
			err = fmt.Errorf("fns create failed, create eventbus failed, %v", busErr)
			return
		}
		app0.eventbus = bus
	} else {
		app0.eventbus = opt.Eventbus
	}

	// succeed
	a = app0

	return
}

type app struct {
	name         string
	tags         []string
	config       Config
	clusterMode  bool
	cluster      cluster.Cluster
	servicesLock sync.Mutex
	services     []Service
	log          Logs
	eventbus     eventbus.Eventbus
}

func (a *app) Deploy(services ...Service) {
	if services == nil || len(services) == 0 {
		return
	}
	a.servicesLock.Lock()
	defer a.servicesLock.Unlock()

	for _, service := range services {
		if service == nil {
			continue
		}
		if a.services == nil {
			a.services = make([]Service, 0, 1)
		}
		a.services = append(a.services, service)
	}
	return
}

func (a *app) Run(ctx context.Context) (err error) {
	servicesErr := a.checkService()
	if servicesErr != nil {
		err = servicesErr
		return
	}
	// join into cluster
	if a.clusterMode {
		joinErr := a.cluster.Join()
		if joinErr != nil {
			err = fmt.Errorf("fns run failed, join into cluster failed, %v", joinErr)
			return
		}
	}
	// start services
	services := a.services
	failed := false
	var cause error
	env := newFnsEnvironment(a.config, a.cluster)
	for _, service := range services {
		serviceName := service.Name()
		serviceLog := logs.With(a.log, logs.F("service", serviceName))
		fnsCTX := newFnsContext(ctx, serviceLog, a.eventbus, a.cluster)
		serviceErr := service.Start(fnsCTX, env)
		if serviceErr != nil {
			cause = serviceErr
			failed = true
			break
		}
	}
	if failed {
		_ = a.Close(ctx)
		err = fmt.Errorf("fns run failed, %v", cause)
		return
	}
	// ...

	return
}

func (a *app) Sync(ctx context.Context) (err error) {
	err = a.SyncWithTimeout(ctx, 10*time.Second)
	return
}

func (a *app) SyncWithTimeout(ctx context.Context, timeout time.Duration) (err error) {

	ch := make(chan os.Signal, 1)
	signal.Notify(ch,
		os.Interrupt,
		syscall.SIGINT,
		os.Kill,
		syscall.SIGKILL,
		syscall.SIGTERM,
	)

	select {
	case <-ch:
		cancelCTX, cancel := context.WithTimeout(ctx, timeout)
		closeCh := make(chan struct{}, 1)
		go func(ctx context.Context, a *app, closeCh chan struct{}) {
			_ = a.Close(ctx)
			closeCh <- struct{}{}
			close(closeCh)
		}(cancelCTX, a, closeCh)
		select {
		case <-closeCh:
			cancel()
			return
		case <-cancelCTX.Done():
			err = fmt.Errorf("fns sync timeout")
			cancel()
			return
		}
	case <-ctx.Done():
		break
	}

	return
}

func (a *app) checkService() (err error) {
	services := a.services
	if services == nil || len(services) == 0 {
		err = fmt.Errorf("fns create failed, services is empty")
		return
	}
	for _, s1 := range services {
		s1Name := strings.TrimSpace(s1.Name())
		if s1Name == "" {
			err = fmt.Errorf("fns create failed, has no named service")
			return
		}
		duplicated := false
		for _, s2 := range services {
			if s1.Name() == s2.Name() {
				duplicated = true
				break
			}
		}
		if duplicated {
			err = fmt.Errorf("fns run failed, service %s is duplicated", s1.Name())
		}
	}
	// services sort
	sort.Slice(services, func(i, j int) bool {
		return services[i].Index() < services[j].Index()
	})
	a.services = services
	return
}

func (a *app) Close(ctx context.Context) (err error) {
	// leave into cluster
	if a.clusterMode {
		leaveErr := a.cluster.Leave()
		if leaveErr != nil {
			a.log.Warnf("fns leave from cluster failed, %v", leaveErr)
		}
	}
	services := a.services
	for _, service := range services {
		serviceName := service.Name()
		serviceLog := logs.With(a.log, logs.F("service", serviceName))
		fnsCTX := newFnsContext(ctx, serviceLog, a.eventbus, a.cluster)
		serviceErr := service.Stop(fnsCTX)
		if serviceErr != nil {
			a.log.Warnf("fns stop service %s failed, %v", serviceName, serviceErr)
		}
	}
	return
}
