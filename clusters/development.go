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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/clusters/proxy"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/mmhash"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"strings"
	"time"
)

const developmentName = "dev"

type DevelopmentConfig struct {
	ProxyAddr string `json:"proxyAddr"`
}

func NewDevelopment(dialer transports.Dialer, signature signatures.Signature) Cluster {
	return &Development{
		log:       nil,
		signature: signature,
		address:   nil,
		dialer:    dialer,
		client:    nil,
		events:    make(chan NodeEvent, 64),
		closeFn:   nil,
		nodes:     make(Nodes, 0, 1),
	}
}

type Development struct {
	log       logs.Logger
	signature signatures.Signature
	address   []byte
	dialer    transports.Dialer
	client    transports.Client
	events    chan NodeEvent
	closeFn   context.CancelFunc
	nodes     Nodes
}

func (cluster *Development) Construct(options ClusterOptions) (err error) {
	cluster.log = options.Log
	config := DevelopmentConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: dev cluster construct failed").WithCause(configErr)
		return
	}
	address := strings.TrimSpace(config.ProxyAddr)
	if address == "" {
		err = errors.Warning("fns: dev cluster construct failed").WithCause(fmt.Errorf("address is required"))
		return
	}
	cluster.address = []byte(address)
	return
}

func (cluster *Development) AddService(_ Service) {
	return
}

func (cluster *Development) Join(ctx context.Context) (err error) {
	cluster.client, err = cluster.dialer.Dial(cluster.address)
	if err != nil {
		err = errors.Warning("fns: dev cluster join failed").WithCause(err)
		return
	}
	ctx, cluster.closeFn = context.WithCancel(ctx)
	go cluster.watching(ctx)
	return
}

func (cluster *Development) Leave(_ context.Context) (err error) {
	cluster.closeFn()
	close(cluster.events)
	return
}

func (cluster *Development) NodeEvents() (events <-chan NodeEvent) {
	events = cluster.events
	return
}

func (cluster *Development) Shared() (shared shareds.Shared) {
	shared = proxy.NewShared(cluster.Client, cluster.signature)
	return
}

func (cluster *Development) Client() transports.Client {
	return cluster.client
}

func (cluster *Development) Barrier() (barrier barriers.Barrier) {
	barrier = barriers.New()
	return
}

func (cluster *Development) watching(ctx context.Context) {
	cluster.fetchAndUpdate(ctx)
	stop := false
	timer := time.NewTimer(10 * time.Second)
	for {
		select {
		case <-ctx.Done():
			stop = true
			break
		case <-timer.C:
			cluster.fetchAndUpdate(ctx)
			break
		}
		if stop {
			timer.Stop()
			break
		}
	}
}

func (cluster *Development) fetchAndUpdate(ctx context.Context) {
	nodes := cluster.fetchNodes(ctx)
	if len(nodes) == 0 {
		if len(cluster.nodes) == 0 {
			return
		}
		cluster.events <- NodeEvent{
			Kind: Remove,
			Node: cluster.nodes[0],
		}
	} else {
		op, _ := json.Marshal(cluster.nodes)
		np, _ := json.Marshal(nodes)
		if mmhash.Sum64(op) == mmhash.Sum64(np) {
			return
		}
		cluster.events <- NodeEvent{
			Kind: Remove,
			Node: nodes[0],
		}
		cluster.events <- NodeEvent{
			Kind: Add,
			Node: nodes[0],
		}
	}
	cluster.nodes = nodes
	return
}

func (cluster *Development) fetchNodes(ctx context.Context) (nodes Nodes) {
	infos, infosErr := proxy.FetchEndpointInfos(ctx, cluster.client, cluster.signature)
	if infosErr != nil {
		if cluster.log.WarnEnabled() {
			cluster.log.Warn().Cause(infosErr).With("cluster", developmentName).Message("fns: fetch endpoint infos failed")
		}
		return
	}
	for i, info := range infos {
		info.Address = bytex.ToString(cluster.address)
		infos[i] = info
	}
	nodes = MapEndpointInfosToNodes(infos)
	return
}
