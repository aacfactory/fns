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
	"encoding/json"
	"github.com/aacfactory/cluster"
)

// +-------------------------------------------------------------------------------------------------------------------+

type ClusterConfig struct {
	Enable bool            `json:"enable,omitempty"`
	Kind   string          `json:"kind,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

var clusterRetriever ClusterRetriever = nil

type ClusterRetriever func(name string, tags []string, config []byte) (c cluster.Cluster, err error)

//RegisterClusterRetriever 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterClusterRetriever(fn ClusterRetriever) {
	clusterRetriever = fn
}

// +-------------------------------------------------------------------------------------------------------------------+
