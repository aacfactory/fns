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
	"github.com/aacfactory/json"
	"net/http"
)

type Node interface {
	Id() string
	AppendService(service string, internal bool)
}

type nodeService struct {
	Name     string `json:"name"`
	Internal bool   `json:"internal"`
}

type node struct {
	Id_      string         `json:"id"`
	Address  string         `json:"address"`
	Services []*nodeService `json:"services"`
	client   Client
}

func (node *node) Id() string {
	return node.Id_
}

func (node *node) AppendService(service string, internal bool) {
	node.Services = append(node.Services, &nodeService{
		Name:     service,
		Internal: internal,
	})
}

func (node *node) registrations() (registrations []*Registration) {
	registrations = make([]*Registration, 0, 1)
	if node.Services != nil {
		for _, service := range node.Services {
			registrations = append(registrations, &Registration{
				Id:               node.Id_,
				Name:             service.Name,
				Internal:         service.Internal,
				Address:          node.Address,
				client:           node.client,
				unavailableTimes: 0,
			})
		}
	}
	return
}

func (node *node) available() (ok bool) {
	status, _, body, callErr := node.client.Do(context.TODO(), http.MethodGet, node.Address, "/health", nil, nil)
	if callErr != nil {
		return
	}
	if status != http.StatusOK {
		return
	}
	if body == nil || !json.Validate(body) {
		return
	}
	obj := json.NewObjectFromBytes(body)
	_ = obj.Get("running", &ok)
	return
}

type nodeEvent struct {
	kind  string
	value *node
}
