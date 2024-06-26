/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package clusters

import (
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/clusters/proxy"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/workers"
	"strings"
)

type ClusterOptions struct {
	Log     logs.Logger
	Config  configures.Config
	Id      string
	Version versions.Version
	Address string
}

type Cluster interface {
	Construct(options ClusterOptions) (err error)
	AddService(service Service)
	Join(ctx context.Context) (err error)
	Leave(ctx context.Context) (err error)
	NodeEvents() (events <-chan NodeEvent)
	Shared() (shared shareds.Shared)
	Barrier() (barrier barriers.Barrier)
}

type ClusterBuilderOptions struct {
	Config configures.Config
	Log    logs.Logger
}

var (
	clusterMap = make(map[string]Cluster)
)

func RegisterCluster(name string, cluster Cluster) {
	clusterMap[name] = cluster
}

func loadCluster(name string) (cluster Cluster, has bool) {
	cluster, has = clusterMap[name]
	return
}

type Options struct {
	Id      string
	Version versions.Version
	Port    int
	Log     logs.Logger
	Worker  workers.Workers
	Local   services.EndpointsManager
	Dialer  transports.Dialer
	Config  Config
}

func New(options Options) (manager services.EndpointsManager, shared shareds.Shared, barrier barriers.Barrier, handlers []transports.MuxHandler, err error) {
	// signature
	signature := NewSignature(options.Config.Secret)
	// host
	hostRetrieverName := strings.TrimSpace(options.Config.HostRetriever)
	if hostRetrieverName == "" {
		hostRetrieverName = "default"
	}
	hostRetriever, hasHostRetriever := getHostRetriever(hostRetrieverName)
	if !hasHostRetriever {
		err = errors.Warning("fns: new cluster failed").WithCause(fmt.Errorf("host retriever was not found")).WithMeta("name", hostRetrieverName)
		return
	}
	host, hostErr := hostRetriever()
	if hostErr != nil {
		err = errors.Warning("fns: new cluster failed").WithCause(hostErr)
		return
	}
	address := fmt.Sprintf("%s:%d", host, options.Port)
	// cluster
	var cluster Cluster
	if options.Config.Name == developmentName {
		cluster = NewDevelopment(options.Dialer, signature)
	} else {
		has := false
		cluster, has = loadCluster(options.Config.Name)
		if !has {
			err = errors.Warning("fns: new cluster failed").WithCause(fmt.Errorf("cluster was not found")).WithMeta("name", options.Config.Name)
			return
		}
	}
	if options.Config.Option == nil && len(options.Config.Option) < 2 {
		options.Config.Option = []byte{'{', '}'}
	}
	clusterConfig, clusterConfigErr := configures.NewJsonConfig(options.Config.Option)
	if clusterConfigErr != nil {
		err = errors.Warning("fns: new cluster failed").WithCause(clusterConfigErr).WithMeta("name", options.Config.Name)
		return
	}
	clusterErr := cluster.Construct(ClusterOptions{
		Log:     options.Log.With("cluster", options.Config.Name),
		Config:  clusterConfig,
		Id:      options.Id,
		Version: options.Version,
		Address: address,
	})
	if clusterErr != nil {
		err = errors.Warning("fns: new cluster failed").WithCause(clusterErr).WithMeta("name", options.Config.Name)
		return
	}
	// shared
	shared = cluster.Shared()
	// barrier
	barrier = cluster.Barrier()
	// manager
	manager = NewManager(options.Id, options.Version, address, cluster, options.Local, options.Worker, options.Log, options.Dialer, signature)
	// handlers
	handlers = make([]transports.MuxHandler, 0, 1)
	handlers = append(handlers, NewInternalHandler(options.Local, signature))
	if options.Config.Proxy {
		// append proxy handler
		handlers = append(handlers, proxy.NewHandler(signature, manager, cluster.Shared()))
	}
	return
}
