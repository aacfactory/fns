package services

import (
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/logs"
	"strings"
	"sync"
)

type Endpoints struct {
	log       logs.Logger
	rt        *Runtime
	deployed  map[string]*endpoint
	discovery Discovery
	closeCh   chan struct{}
}

func (e *Endpoints) Runtime() (rt *Runtime) {
	rt = e.rt
	return
}

func (e *Endpoints) Deploy(service Service) (err error) {
	name := strings.TrimSpace(service.Name())
	serviceConfig, hasConfig := e.config.Node(name)
	if !hasConfig {
		serviceConfig, _ = configures.NewJsonConfig([]byte("{}"))
	}
	buildErr := svc.Build(Options{
		AppId:      e.rt.appId,
		AppName:    e.rt.appName,
		AppVersion: e.rt.appVersion,
		Log:        e.log.With("fns", "service").With("service", name),
		Config:     serviceConfig,
	})
	if buildErr != nil {
		err = errors.Warning(fmt.Sprintf("fns: endpoints deploy %s service failed", name)).WithMeta("service", name).WithCause(buildErr)
		return
	}
	ep := &endpoint{
		rt:            e.rt,
		handleTimeout: e.handleTimeout,
		svc:           svc,
		pool:          sync.Pool{},
	}
	ep.pool.New = func() any {
		return newFnTask(svc, e.rt.barrier, e.handleTimeout, func(task *fnTask) {
			ep.release(task)
		})
	}
	e.deployed[svc.Name()] = ep
	e.rt.appServices = append(e.rt.appServices, svc)
	return
}
