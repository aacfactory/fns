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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/internal/lru"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"golang.org/x/sync/singleflight"
	"net/http"
	"time"
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
	client, clientErr := cluster.dialer.Dial(cluster.proxyAddress)
	if clientErr != nil {
		err = errors.Warning("proxy: dialer proxy failed").WithCause(clientErr).WithMeta("proxy", cluster.proxyAddress)
		return
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	header := http.Header{}
	header.Add(httpDeviceIdHeader, cluster.appId)
	header.Add(httpRequestSignatureHeader, bytex.ToString(cluster.signer.Sign(bytex.FromString(cluster.appId))))
	status, _, respBody, postErr := client.Post(ctx, "/cluster/nodes", header, []byte{'{', '}'})
	if postErr != nil {
		err = errors.Warning("proxy: get nodes from proxy failed").WithCause(postErr).WithMeta("proxy", cluster.proxyAddress)
		return
	}
	if status == http.StatusOK {
		nodes = make([]Node, 0, 1)
		decodeErr := json.Unmarshal(respBody, nodes)
		if decodeErr != nil {
			err = errors.Warning("proxy: get nodes from proxy failed").WithCause(decodeErr).WithMeta("proxy", cluster.proxyAddress)
			return
		}
	} else {
		err = errors.Warning("proxy: get nodes from proxy failed").WithCause(errors.Decode(respBody)).WithMeta("proxy", cluster.proxyAddress)
	}
	return
}

func (cluster *devCluster) Shared() (shared Shared) {
	// todo 也proxy
	shared = cluster.proxy.Shared()
	return
}

const (
	clusterProxyName = "cluster_proxy"
)

func newClusterProxyHandler(cluster Cluster, secretKey []byte) *clusterProxyHandler {
	return &clusterProxyHandler{
		log:     nil,
		cluster: cluster,
		signer:  secret.NewSigner(secretKey),
		cache:   lru.New[string, json.RawMessage](8),
		group:   &singleflight.Group{},
	}
}

type clusterProxyHandler struct {
	appId      string
	appName    string
	appVersion versions.Version
	log        logs.Logger
	cluster    Cluster
	signer     *secret.Signer
	cache      *lru.LRU[string, json.RawMessage] // todo change lru to caches
	group      *singleflight.Group
}

func (handler *clusterProxyHandler) Name() (name string) {
	name = clusterProxyName
	return
}

func (handler *clusterProxyHandler) Build(options *HttpHandlerOptions) (err error) {
	handler.log = options.Log
	handler.appId = options.AppId
	handler.appName = options.AppName
	handler.appVersion = options.AppVersion
	return
}

func (handler *clusterProxyHandler) Accept(r *http.Request) (ok bool) {
	ok = r.Method == http.MethodPost && r.URL.Path == "/cluster/nodes"
	if ok {
		return
	}
	return
}

func (handler *clusterProxyHandler) Close() {
	return
}

func (handler *clusterProxyHandler) ServeHTTP(writer http.ResponseWriter, r *http.Request) {
	const (
		nodesKey        = "nodes"
		refreshGroupKey = "nodes_refresh"
	)
	devId := r.Header.Get(httpDeviceIdHeader)
	if devId == "" {
		handler.failed(writer, 555, errors.Warning("proxy: X-Fns-Device-Id is required"))
		return
	}
	if !handler.signer.Verify(bytex.FromString(devId), bytex.FromString(r.Header.Get(httpRequestSignatureHeader))) {
		handler.failed(writer, 555, errors.Warning("proxy: X-Fns-Request-Signature is invalid"))
		return
	}
	groupKey := nodesKey
	if r.URL.Query().Get("refresh") == "true" {
		groupKey = refreshGroupKey
	}
	v, err, _ := handler.group.Do(groupKey, func() (v interface{}, err error) {
		if r.URL.Query().Get("refresh") != "true" {
			cached, has := handler.cache.Get(nodesKey)
			if has {
				v = cached
				return
			}
		}
		nodes, getNodesErr := listMembers(r.Context(), handler.cluster, handler.appId, handler.appName, handler.appVersion)
		if getNodesErr != nil {
			err = errors.Warning("proxy: handle cluster nodes request failed").WithCause(getNodesErr)
			return
		}
		p, encodeErr := json.Marshal(nodes)
		if encodeErr != nil {
			err = errors.Warning("proxy: handle cluster nodes request failed").WithCause(encodeErr)
			return
		}
		handler.cache.Add(nodesKey, p, 60*time.Second)
		v = json.RawMessage(p)
		return
	})
	if err != nil {
		handler.failed(writer, 555, errors.Map(err))
		return
	}
	body := v.(json.RawMessage)
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(http.StatusOK)
	n := 0
	bodyLen := len(body)
	for n < bodyLen {
		nn, writeErr := writer.Write(body[n:])
		if writeErr != nil {
			return
		}
		n += nn
	}
	return
}

func (handler *clusterProxyHandler) failed(writer http.ResponseWriter, status int, cause interface{}) {
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(status)
	body, _ := json.Marshal(cause)
	n := 0
	bodyLen := len(body)
	for n < bodyLen {
		nn, writeErr := writer.Write(body[n:])
		if writeErr != nil {
			return
		}
		n += nn
	}
	return
}
