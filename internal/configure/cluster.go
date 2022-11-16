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

package configure

import "github.com/aacfactory/json"

type Cluster struct {
	DevMode           bool            `json:"devMode"`
	NodesProxyAddress string          `json:"nodesProxyAddress"`
	Kind              string          `json:"kind"`
	Client            ClusterClient   `json:"client"`
	Visitor           bool            `json:"visitor"`
	Options           json.RawMessage `json:"options"`
}

type ClusterClient struct {
	MaxIdleConnSeconds    int `json:"maxIdleConnSeconds"`
	MaxConnsPerHost       int `json:"maxConnsPerHost"`
	MaxIdleConnsPerHost   int `json:"maxIdleConnsPerHost"`
	RequestTimeoutSeconds int `json:"requestTimeoutSeconds"`
}
