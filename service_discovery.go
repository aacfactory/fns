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
		err = fmt.Errorf("ServiceDiscovery: Publish nil pointer service")
		return
	}
	namespace := strings.TrimSpace(service.Namespace())
	if namespace == "" {
		err = fmt.Errorf("ServiceDiscovery: Publish no namespace service")
		return
	}

	_, has := discovery.proxyMap[namespace]
	if has {
		err = fmt.Errorf("ServiceDiscovery: Publish duplicated namespace service")
		return
	}
	discovery.proxyMap[namespace] = NewLocaledServiceProxy(service)
	return
}

func (discovery *standaloneServiceDiscovery) IsLocal(namespace string) (ok bool) {
	_, ok = discovery.proxyMap[namespace]
	return
}

func (discovery *standaloneServiceDiscovery) Proxy(ctx Context, namespace string) (proxy ServiceProxy, err errors.CodeError) {
	proxy0, has := discovery.proxyMap[namespace]
	if !has || proxy0 == nil {
		err = errors.NotFound(fmt.Sprintf("%s service was not found", namespace))
		return
	}
	proxy = proxy0
	return
}

func (discovery *standaloneServiceDiscovery) ProxyByExact(ctx Context, proxyId string) (proxy ServiceProxy, err errors.CodeError) {
	for _, serviceProxy := range discovery.proxyMap {
		if serviceProxy.Id() == proxyId {
			proxy = serviceProxy
			return
		}
	}
	return
}

func (discovery *standaloneServiceDiscovery) Close() {}
