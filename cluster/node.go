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

type Node struct {
	Id               string   `json:"id"`
	Address          string   `json:"address"`
	Services         []string `json:"services"`
	InternalServices []string `json:"internalServices"`
	client           Client
}

func (node *Node) AppendService(services ...string) {
	node.Services = append(node.Services, services...)
}

func (node *Node) AppendInternalService(services ...string) {
	node.InternalServices = append(node.InternalServices, services...)
}

func (node *Node) Registrations() (registrations []*Registration) {
	registrations = make([]*Registration, 0, 1)
	if node.Services != nil {
		for _, service := range node.Services {
			registrations = append(registrations, &Registration{
				Id:               node.Id,
				Name:             service,
				Internal:         false,
				Address:          node.Address,
				client:           node.client,
				unavailableTimes: 0,
			})
		}
	}
	if node.InternalServices != nil {
		for _, service := range node.InternalServices {
			registrations = append(registrations, &Registration{
				Id:               node.Id,
				Name:             service,
				Internal:         true,
				Address:          node.Address,
				client:           node.client,
				unavailableTimes: 0,
			})
		}
	}
	return
}

type nodeEvent struct {
	kind  string
	value *Node
}
