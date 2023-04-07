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
	"context"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/logs"
)

type ClusterBuilderOptions struct {
	Config     configures.Config
	Log        logs.Logger
	AppId      string
	AppName    string
	AppVersion versions.Version
}

type ClusterBuilder func(options ClusterBuilderOptions) (cluster Cluster, err error)

// Cluster 只给address，然后service通过register的ch获得address（新增和删除），调用/services/stats获取列表，
// 判断是否有websocket的方式为用address，调用get /applications/handlers 获取，判断有没有，一般是services，websockets
type Cluster interface {
	Join(ctx context.Context) (err error)
	Leave(ctx context.Context) (err error)
	// Nodes 记得塞deviceId（appId） 和 签名
	Nodes(ctx context.Context) (nodes Nodes, err error)
	Shared() (shared Shared)
}

type Node struct {
	Id       string           `json:"id"`
	Name     string           `json:"name"`
	Version  versions.Version `json:"version"`
	Address  string           `json:"address"`
	Services []string         `json:"services"`
}

type Nodes []*Node

func (nodes Nodes) Len() int {
	return len(nodes)
}

func (nodes Nodes) Less(i, j int) bool {
	return nodes[i].Id < nodes[j].Id
}

func (nodes Nodes) Swap(i, j int) {
	nodes[i], nodes[j] = nodes[j], nodes[i]
	return
}

var (
	builders = make(map[string]ClusterBuilder)
)

func RegisterClusterBuilder(name string, builder ClusterBuilder) {
	builders[name] = builder
}

func getClusterBuilder(name string) (builder ClusterBuilder, has bool) {
	if name == devClusterBuilderName {
		return devClusterBuilder, true
	}
	builder, has = builders[name]
	return
}

const (
	devClusterBuilderName = "dev"
)

func devClusterBuilder(options ClusterBuilderOptions) (cluster Cluster, err error) {
	// todo add dialer builder
	return
}

func newDevProxyCluster(appId string, cluster Cluster, proxyAddress string, dialer transports.Dialer, secretKey []byte) Cluster {
	return &devCluster{
		appId:        appId,
		proxyAddress: proxyAddress,
		dialer:       dialer,
		proxy:        cluster,
		signer:       secret.NewSigner(secretKey),
	}
}

type devCluster struct {
	appId        string
	proxyAddress string
	dialer       transports.Dialer
	proxy        Cluster
	signer       *secret.Signer
}

func (cluster *devCluster) Dialer() transports.Dialer {
	return cluster.dialer
}

func (cluster *devCluster) Join(ctx context.Context) (err error) {
	return
}

func (cluster *devCluster) Leave(ctx context.Context) (err error) {
	return
}

func (cluster *devCluster) Nodes(ctx context.Context) (nodes Nodes, err error) {

	return
}

func (cluster *devCluster) Shared() (shared Shared) {
	// todo 也proxy
	shared = cluster.proxy.Shared()
	return
}
