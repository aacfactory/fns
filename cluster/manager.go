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

package cluster

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/configure"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"sort"
	"strings"
	"time"
)

type ManagerOptions struct {
	Log               logs.Logger
	Port              int
	Config            *configure.Cluster
	ClientHttps       bool
	ClientTLS         *tls.Config
	ClientBuilder     ClientBuilder
	DevMode           bool
	NodesProxyAddress string
}

func NewManager(options ManagerOptions) (manager *Manager, err error) {
	kind := strings.TrimSpace(options.Config.Kind)
	if kind == "" {
		kind = "members"
		return
	}
	bootstrap, hasBootstrap := getRegisteredBootstrap(kind)
	if !hasBootstrap {
		err = errors.Warning(fmt.Sprintf("fns: %s kind bootstrap is not registerd", kind))
		return
	}
	optionsConfig, optionsConfigErr := configures.NewJsonConfig(options.Config.Options)
	if optionsConfigErr != nil {
		err = errors.Warning(fmt.Sprintf("fns: options is invalid")).WithCause(optionsConfigErr)
		return
	}
	bootstrapConfig, hasBootstrapConfig := optionsConfig.Node(kind)
	if !hasBootstrapConfig {
		bootstrapConfig, _ = configures.NewJsonConfig([]byte{'{', '}'})
	}
	bootstrapBuildErr := bootstrap.Build(BootstrapOptions{
		Config: bootstrapConfig,
		Log:    options.Log.With("cluster", "bootstrap"),
	})
	if bootstrapBuildErr != nil {
		err = errors.Warning(fmt.Sprintf("fns: build bootstrap failed")).WithCause(bootstrapBuildErr)
		return
	}
	id := bootstrap.Id()
	if id == "" {
		err = fmt.Errorf("fns: can not get my id from bootstrap")
		return
	}
	ip := bootstrap.Ip()
	if ip == "" {
		err = fmt.Errorf("fns: can not get my ip from bootstrap")
		return
	}
	clientConfig := options.Config.Client
	maxIdleConnSeconds := clientConfig.MaxIdleConnSeconds
	if maxIdleConnSeconds < 1 {
		maxIdleConnSeconds = 10
	}
	maxConnsPerHost := clientConfig.MaxConnsPerHost
	if maxConnsPerHost < 1 {
		maxConnsPerHost = 0
	}
	maxIdleConnsPerHost := clientConfig.MaxIdleConnsPerHost
	if maxIdleConnsPerHost < 1 {
		maxIdleConnsPerHost = 0
	}
	requestTimeoutSeconds := clientConfig.RequestTimeoutSeconds
	if requestTimeoutSeconds < 1 {
		requestTimeoutSeconds = 2
	}
	clientOptions := ClientOptions{
		Log:                 options.Log.With("cluster", "client"),
		Https:               options.ClientHttps,
		TLS:                 options.ClientTLS,
		MaxIdleConnDuration: time.Duration(maxIdleConnSeconds) * time.Second,
		MaxConnsPerHost:     maxConnsPerHost,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		RequestTimeout:      time.Duration(requestTimeoutSeconds) * time.Second,
	}
	client, clientErr := options.ClientBuilder(clientOptions)
	if clientErr != nil {
		err = errors.Warning("fns: create cluster client failed").WithCause(clientErr)
		return
	}
	manager = &Manager{
		log:               options.Log.With("cluster", "manager"),
		devMode:           options.DevMode,
		nodesProxyAddress: strings.TrimSpace(options.NodesProxyAddress),
		bootstrap:         bootstrap,
		interval:          60 * time.Second,
		node: &node{
			Id_:      id,
			Address:  fmt.Sprintf("%s:%d", ip, options.Port),
			Services: make([]*nodeService, 0, 1),
			client:   client,
		},
		client:        client,
		registrations: newRegistrationsManager(options.Log, client),
		stopCh:        make(chan struct{}, 1),
	}
	return
}

type Manager struct {
	log               logs.Logger
	devMode           bool
	nodesProxyAddress string
	bootstrap         Bootstrap
	interval          time.Duration
	node              *node
	client            Client
	registrations     *RegistrationsManager
	stopCh            chan struct{}
}

func (manager *Manager) Join() {
	manager.linkMembers()
	go manager.keepAlive()
}

func (manager *Manager) linkMembers() {
	memberAddresses := manager.bootstrap.FindMembers(context.TODO())
	if memberAddresses == nil {
		memberAddresses = make([]string, 0, 1)
	}
	existMembers := manager.registrations.members()
	for _, member := range existMembers {
		if sort.SearchStrings(memberAddresses, member.Address) < len(memberAddresses) {
			continue
		}
		memberAddresses = append(memberAddresses, member.Address)
	}
	if len(memberAddresses) == 0 {
		if manager.log.DebugEnabled() {
			manager.log.Debug().With("members", fmt.Sprintf("[ ]")).Message(fmt.Sprintf("fns: cluster size is %d", 0))
		}
		return
	}
	p, pErr := json.Marshal(manager.node)
	if pErr != nil {
		if manager.log.DebugEnabled() {
			manager.log.Debug().Message(fmt.Sprintf("%+v", errors.Warning("fns: encode node failed").WithCause(pErr)))
		}
		return
	}
	body := encodeRequestBody(p)
	header := http.Header{}
	header.Set(httpContentType, httpContentTypeCluster)
	memberAddressesLen := len(memberAddresses)
	for _, address := range memberAddresses {
		manager.linkMember(address, memberAddresses, memberAddressesLen, header, body)
	}
	manager.registrations.removeUnavailableNodes()
}

func (manager *Manager) linkMember(target string, members []string, membersLen int, header http.Header, body []byte) {
	target = strings.TrimSpace(target)
	if target == "" {
		return
	}
	if manager.devMode {
		header.Set("X-Fns-DevMode", "true")
	}
	status, _, respBody, callErr := manager.client.Do(context.TODO(), http.MethodPost, target, joinPath, header, body)
	if callErr != nil {
		if manager.log.DebugEnabled() {
			manager.log.Debug().With("member", target).With("status", status).With("step", "call").Cause(callErr).Message("fns: link member failed")
		}
		return
	}
	if status != http.StatusOK {
		if manager.log.DebugEnabled() {
			manager.log.Debug().With("member", target).With("status", status).With("step", "call").Message("fns: link member failed")
		}
		return
	}
	nodes := make([]*node, 0, 1)
	decodeErr := json.Unmarshal(respBody, &nodes)
	if decodeErr != nil {
		if manager.log.DebugEnabled() {
			manager.log.Debug().With("member", target).With("status", status).With("step", "decode response").Cause(decodeErr).Message("fns: link member failed")
		}
		return
	}
	for i, n := range nodes {
		if manager.nodesProxyAddress != "" {
			n.ProxyAddress = manager.nodesProxyAddress
		}
		if i == 0 {
			manager.registrations.register(n)
			continue
		}
		if manager.registrations.containsNode(n) {
			continue
		}
		if sort.SearchStrings(members, n.Address) < membersLen {
			continue
		}
		manager.linkMember(n.Address, members, membersLen, header, body)
	}
}

func (manager *Manager) keepAlive() {
	for {
		stopped := false
		select {
		case <-manager.stopCh:
			stopped = true
			break
		case <-time.After(manager.interval):
			manager.linkMembers()
		}
		if stopped {
			break
		}
	}
}

func (manager *Manager) Leave() {
	close(manager.stopCh)
	manager.registrations.Close()
	body := encodeRequestBody([]byte(fmt.Sprintf("{\"id\":\"%s\"}", manager.node.Id_)))
	header := http.Header{}
	header.Set(httpContentType, httpContentTypeCluster)
	members := manager.registrations.members()
	for _, member := range members {
		status, _, _, _ := manager.client.Do(context.TODO(), http.MethodPost, member.Address, leavePath, header, body)
		if manager.log.DebugEnabled() {
			manager.log.Debug().With("member", member.Id_).With("status", status).Message("fns: leaved")
		}
	}
}

func (manager *Manager) Node() (node Node) {
	node = manager.node
	return
}

func (manager *Manager) Registrations() (registrations *RegistrationsManager) {
	registrations = manager.registrations
	return
}
