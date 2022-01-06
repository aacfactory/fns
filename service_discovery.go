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
	"fmt"
	"github.com/aacfactory/errors"
	"strings"
)

func standaloneServiceDiscoveryRetriever(_ ServiceDiscoveryOption) (discovery ServiceDiscovery, _ error) {
	discovery = &standaloneServiceDiscovery{
		proxyMap: make(map[string]*LocaledServiceProxy),
	}
	return
}

type standaloneServiceDiscovery struct {
	proxyMap map[string]*LocaledServiceProxy
}

func (discovery *standaloneServiceDiscovery) Publish(service Service) (err error) {
	if service == nil {
		err = fmt.Errorf("fns ServiceDiscovery Publish: nil pointer service")
		return
	}
	namespace := strings.TrimSpace(service.Namespace())
	if namespace == "" {
		err = fmt.Errorf("fns ServiceDiscovery Publish: no namespace service")
		return
	}

	_, has := discovery.proxyMap[namespace]
	if has {
		err = fmt.Errorf("fns ServiceDiscovery Publish: duplicated namespace service")
		return
	}
	discovery.proxyMap[namespace] = NewLocaledServiceProxy(service)
	return
}

func (discovery *standaloneServiceDiscovery) IsLocal(namespace string) (ok bool) {
	_, ok = discovery.proxyMap[namespace]
	return
}

func (discovery *standaloneServiceDiscovery) Proxy(_ Context, namespace string) (proxy ServiceProxy, err errors.CodeError) {
	proxy0, has := discovery.proxyMap[namespace]
	if !has || proxy0 == nil {
		err = errors.NotFound(fmt.Sprintf("fns ServiceDiscovery Proxy: %s service was not found", namespace))
		return
	}
	proxy = proxy0
	return
}

func (discovery *standaloneServiceDiscovery) Registrations() (registrations map[string]*Registration) {
	registrations = make(map[string]*Registration)
	for _, proxy := range discovery.proxyMap {
		rid := proxy.service.Namespace()
		registrations[rid] = &Registration{
			Id:        rid,
			Namespace: rid,
			Address:   "",
			Reversion: 0,
		}
	}
	return
}

func (discovery *standaloneServiceDiscovery) Close() {}

// +-------------------------------------------------------------------------------------------------------------------+

func NewAbstractServiceDiscovery(clients *HttpClients) AbstractServiceDiscovery {
	return AbstractServiceDiscovery{
		Local: &standaloneServiceDiscovery{
			proxyMap: make(map[string]*LocaledServiceProxy),
		},
		Clients: clients,
		Manager: newRegistrationsManager(clients),
	}
}

type AbstractServiceDiscovery struct {
	Local   ServiceDiscovery
	Clients *HttpClients
	Manager *RegistrationsManager
}

func (discovery *AbstractServiceDiscovery) IsLocal(namespace string) (ok bool) {
	ok = discovery.Local.IsLocal(namespace)
	return
}

func (discovery *AbstractServiceDiscovery) Proxy(ctx Context, namespace string) (proxy ServiceProxy, err errors.CodeError) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		err = errors.NotFound(fmt.Sprintf("fns ServiceDiscovery Proxy: empty namespace service"))
		return
	}
	// exact
	exactAddress, hasExactProxyAddress := ctx.Meta().GetExactProxyServiceAddress(namespace)
	if hasExactProxyAddress {
		proxy = discovery.exactProxy(namespace, exactAddress)
		return
	}

	// local
	if discovery.IsLocal(namespace) {
		proxy, err = discovery.Local.Proxy(ctx, namespace)
		return
	}

	// remote
	registration, has := discovery.Manager.Get(namespace)
	if !has {
		err = errors.NotFound(fmt.Sprintf("fns ServiceDiscovery Proxy: %s service was not found", namespace))
		return
	}

	proxy = NewRemotedServiceProxy(discovery.Clients, registration, discovery.Manager.ProblemChan())

	return
}

func (discovery *AbstractServiceDiscovery) exactProxy(namespace string, address string) (proxy ServiceProxy) {
	registration := Registration{
		Namespace: namespace,
		Address:   address,
	}
	proxy = NewRemotedServiceProxy(discovery.Clients, &registration, discovery.Manager.ProblemChan())
	return
}

func (discovery *AbstractServiceDiscovery) Close() {
	discovery.Manager.Close()
	discovery.Clients.Close()
	discovery.Local.Close()
	return
}
