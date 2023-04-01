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

package service

import (
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/systems/cpu"
	"github.com/aacfactory/systems/memory"
	"golang.org/x/sync/singleflight"
	"net/http"
	"sort"
	"strings"
	"time"
)

type TransportHandlerOptions struct {
	AppId      string
	AppName    string
	AppVersion versions.Version
	Log        logs.Logger
	Config     configures.Config
	Discovery  EndpointDiscovery
	Shared     Shared
}

type TransportHandler interface {
	Name() (name string)
	Build(options TransportHandlerOptions) (err error)
	Accept(r *http.Request) (ok bool)
	http.Handler
	Close()
}

type TransportHandlersOptions struct {
	AppId      string
	AppName    string
	AppVersion versions.Version
	AppStatus  *Status
	Log        logs.Logger
	Config     configures.Config
	Discovery  EndpointDiscovery
	Shared     Shared
}

func newTransportHandlers(options TransportHandlersOptions) *TransportHandlers {
	handlers := make([]TransportHandler, 0, 1)
	handlers = append(handlers, newTransportApplicationHandler(options.AppStatus))
	return &TransportHandlers{
		appId:      options.AppId,
		appName:    options.AppName,
		appVersion: options.AppVersion,
		log:        options.Log.With("transports", "handlers"),
		config:     options.Config,
		discovery:  options.Discovery,
		shared:     options.Shared,
		handlers:   handlers,
	}
}

type TransportHandlers struct {
	appId      string
	appName    string
	appVersion versions.Version
	log        logs.Logger
	config     configures.Config
	discovery  EndpointDiscovery
	shared     Shared
	handlers   []TransportHandler
}

func (handlers *TransportHandlers) Append(handler TransportHandler) (err error) {
	if handler == nil {
		return
	}
	name := strings.TrimSpace(handler.Name())
	if name == "" {
		err = errors.Warning("append handler failed").WithCause(errors.Warning("one of handler has no name"))
		return
	}
	pos := sort.Search(len(handlers.handlers), func(i int) bool {
		return handlers.handlers[i].Name() == name
	})
	if pos < len(handlers.handlers) {
		err = errors.Warning("append handler failed").WithCause(errors.Warning("this handle was appended")).WithMeta("handler", name)
		return
	}
	config, exist := handlers.config.Node(name)
	if !exist {
		config, _ = configures.NewJsonConfig([]byte{'{', '}'})
	}
	buildErr := handler.Build(TransportHandlerOptions{
		AppId:      handlers.appId,
		AppName:    handlers.appName,
		AppVersion: handlers.appVersion,
		Log:        handlers.log.With("handler", name),
		Config:     config,
		Discovery:  handlers.discovery,
		Shared:     handlers.shared,
	})
	if buildErr != nil {
		err = errors.Warning("append handler failed").WithCause(buildErr).WithMeta("handler", name)
		return
	}
	handlers.handlers = append(handlers.handlers, handler)
	return
}

func (handlers *TransportHandlers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handled := false
	for _, handler := range handlers.handlers {
		if accepted := handler.Accept(r); accepted {
			handler.ServeHTTP(w, r)
			handled = true
			break
		}
	}
	if !handled {
		w.Header().Set(httpContentType, httpContentTypeJson)
		w.WriteHeader(http.StatusNotFound)
		p, _ := json.Marshal(ErrNotFound)
		_, _ = w.Write(p)
	}
}

func (handlers *TransportHandlers) Close() {
	for _, handler := range handlers.handlers {
		handler.Close()
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	transportApplicationHandlerName = "application"
)

type applicationStats struct {
	Id      string         `json:"id"`
	Name    string         `json:"name"`
	Running bool           `json:"running"`
	Mem     *memory.Memory `json:"mem"`
	CPU     *cpuOccupancy  `json:"cpu"`
}

type cpuOccupancy struct {
	Max   cpu.Core `json:"max"`
	Min   cpu.Core `json:"min"`
	Avg   float64  `json:"avg"`
	Cores cpu.CPU  `json:"cores"`
}

func newTransportApplicationHandler(status *Status) *transportApplicationHandler {
	return &transportApplicationHandler{
		appId:        "",
		appName:      "",
		appVersion:   versions.Version{},
		status:       status,
		launchAT:     time.Time{},
		statsEnabled: false,
		group:        singleflight.Group{},
	}
}

type transportApplicationHandler struct {
	appId        string
	appName      string
	appVersion   versions.Version
	status       *Status
	launchAT     time.Time
	statsEnabled bool
	group        singleflight.Group
}

func (handler *transportApplicationHandler) Name() (name string) {
	name = transportApplicationHandlerName
	return
}

func (handler *transportApplicationHandler) Build(options TransportHandlerOptions) (err error) {
	handler.appId = options.AppId
	handler.appName = options.AppName
	handler.appVersion = options.AppVersion
	handler.launchAT = time.Now()
	_, statsErr := options.Config.Get("statsEnable", &handler.statsEnabled)
	if statsErr != nil {
		err = errors.Warning("fns: application handler build failed").WithCause(statsErr).WithMeta("handler", handler.Name())
		return
	}
	return
}

func (handler *transportApplicationHandler) Accept(r *http.Request) (ok bool) {
	ok = r.Method == http.MethodGet && r.URL.Path == "/application/health"
	if ok {
		return
	}
	ok = r.Method == http.MethodGet && r.URL.Path == "/application/stats"
	if ok {
		return
	}
	return
}

func (handler *transportApplicationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/application/health" {
		body := fmt.Sprintf(
			"{\"name\":\"%s\", \"id\":\"%s\", \"version\":\"%s\", \"launch\":\"%s\", \"now\":\"%s\", \"deviceIp\":\"%s\"}",
			handler.appName, handler.appId, handler.appVersion.String(), handler.launchAT.Format(time.RFC3339), time.Now().Format(time.RFC3339), r.Header.Get(httpDeviceIpHeader),
		)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytex.FromString(body))
		return
	}
	if handler.statsEnabled && r.Method == http.MethodGet && r.URL.Path == "/application/stats" {
		v, _, _ := handler.group.Do(handler.Name(), func() (v interface{}, err error) {
			stat := &applicationStats{
				Id:      handler.appId,
				Name:    handler.appName,
				Running: handler.status.Starting() || handler.status.Serving(),
				Mem:     nil,
				CPU:     nil,
			}
			mem, memErr := memory.Stats()
			if memErr == nil {
				stat.Mem = mem
			}
			cpus, cpuErr := cpu.Occupancy()
			if cpuErr == nil {
				stat.CPU = &cpuOccupancy{
					Max:   cpus.Max(),
					Min:   cpus.Min(),
					Avg:   cpus.AVG(),
					Cores: cpus,
				}
			}
			v, _ = json.Marshal(stat)
			return
		})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(v.([]byte))
		return
	}
	return
}

func (handler *transportApplicationHandler) Close() {
	return
}
