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
	"crypto/tls"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/shareds"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"strings"
	"time"
)

type ClusterBuilderOptions struct {
	Config     configures.Config
	Log        logs.Logger
	AppId      string
	AppName    string
	AppVersion versions.Version
	Port       int
}

type ClusterBuilder func(options ClusterBuilderOptions) (cluster Cluster, err error)

type Cluster interface {
	Join(ctx context.Context) (err error)
	Leave(ctx context.Context) (err error)
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

func initRegisterDevClusterBuilder() {
	RegisterClusterBuilder(devClusterBuilderName, devClusterBuilder)
}

func getClusterBuilder(name string) (builder ClusterBuilder, has bool) {
	builder, has = builders[name]
	return
}

type fakeTransportHandler struct{}

func (handler fakeTransportHandler) Handle(w transports.ResponseWriter, _ *transports.Request) {
	w.Succeed(&Empty{})
}

const (
	devClusterBuilderName = "dev"
)

type devClusterConfig struct {
	ProxyTransportName string     `json:"proxyTransportName"`
	ProxyAddress       string     `json:"proxyAddress"`
	TLS                *TLSConfig `json:"tls"`
}

func devClusterBuilder(options ClusterBuilderOptions) (cluster Cluster, err error) {
	config := devClusterConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: build dev cluster failed").WithCause(configErr)
		return
	}
	proxyTransportName := strings.TrimSpace(config.ProxyTransportName)
	if proxyTransportName == "" {
		err = errors.Warning("fns: build dev cluster failed").WithCause(errors.Warning("proxyTransportName of config options is required"))
		return
	}
	proxyAddress := strings.TrimSpace(config.ProxyAddress)
	if proxyAddress == "" {
		err = errors.Warning("fns: build dev cluster failed").WithCause(errors.Warning("proxyAddress of config options is required"))
		return
	}
	var srvTLS *tls.Config
	var cliTLS *tls.Config
	if config.TLS != nil {
		srvTLS, cliTLS, err = config.TLS.Config()
		if err != nil {
			err = errors.Warning("fns: build dev cluster failed").WithCause(err)
			return
		}
	}
	proxyTransport, registered := transports.Registered(proxyTransportName)
	if !registered {
		err = errors.Warning("fns: build dev cluster failed").WithCause(errors.Warning("proxy transport was not registered")).WithMeta("transport", proxyTransportName)
		return
	}
	transportsConfig, _ := configures.NewJsonConfig([]byte{'{', '}'})
	transportsOptions := transports.Options{
		Port:      13000,
		ServerTLS: srvTLS,
		ClientTLS: cliTLS,
		Handler:   &fakeTransportHandler{},
		Log:       options.Log.With("transport", "dev"),
		Config:    transportsConfig,
	}
	buildTransportErr := proxyTransport.Build(transportsOptions)
	if buildTransportErr != nil {
		err = errors.Warning("fns: build dev cluster failed").WithCause(buildTransportErr)
		return
	}
	client, dialErr := proxyTransport.Dial(proxyAddress)
	if dialErr != nil {
		err = errors.Warning("fns: build dev cluster failed").WithCause(dialErr)
		return
	}
	cluster = &devCluster{
		appId:        options.AppId,
		proxyAddress: proxyAddress,
		dialer:       proxyTransport,
		client:       client,
		shared: &devShared{
			lockers: &devSharedLockers{
				appId:  options.AppId,
				client: client,
			},
			store: &devSharedStore{
				appId:  options.AppId,
				client: client,
			},
			caches: &devSharedCaches{
				appId:  options.AppId,
				client: client,
			},
		},
	}
	return
}

type devCluster struct {
	appId        string
	proxyAddress string
	dialer       transports.Dialer
	client       transports.Client
	shared       *devShared
}

func (cluster *devCluster) Join(_ context.Context) (err error) {
	return
}

func (cluster *devCluster) Leave(_ context.Context) (err error) {
	return
}

func (cluster *devCluster) Nodes(ctx context.Context) (nodes Nodes, err error) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodGET, devClusterNodesPath)
	req.Header().Set(httpDevModeHeader, "*")
	resp, doErr := cluster.client.Do(ctx, req)
	if doErr != nil {
		err = errors.Warning("dev: cluster get nodes failed").WithCause(doErr)
		return
	}
	if resp.Status != http.StatusOK {
		err = errors.Warning("dev: cluster get nodes failed")
		return
	}
	nodes = make(Nodes, 0, 1)
	decodeErr := json.Unmarshal(resp.Body, &nodes)
	if decodeErr != nil {
		err = errors.Warning("dev: cluster get nodes failed").WithCause(decodeErr)
		return
	}
	return
}

func (cluster *devCluster) Shared() (shared Shared) {
	shared = cluster.shared
	return
}

type devShared struct {
	lockers *devSharedLockers
	store   *devSharedStore
	caches  *devSharedCaches
}

func (dev *devShared) Lockers() (lockers shareds.Lockers) {
	lockers = dev.lockers
	return
}

func (dev *devShared) Store() (store shareds.Store) {
	store = dev.store
	return
}

func (dev *devShared) Caches() (store shareds.Caches) {
	store = dev.caches
	return
}

type devSharedLockers struct {
	appId  string
	client transports.Client
}

func (dev *devSharedLockers) Acquire(ctx context.Context, key []byte, ttl time.Duration) (locker shareds.Locker, err error) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, devClusterSharedPath)
	req.Header().Set(httpDevModeHeader, "*")
	subParam := devAcquireLockerParam{
		Key: bytex.ToString(key),
		TTL: ttl,
	}
	subParamBytes, _ := json.Marshal(subParam)
	param := &devShardParam{
		Type:    "locker:acquire",
		Payload: subParamBytes,
	}
	paramBytes, _ := json.Marshal(param)
	req.SetBody(paramBytes)
	resp, doErr := dev.client.Do(ctx, req)
	if doErr != nil {
		err = errors.Warning("dev: lockers acquire failed").WithCause(doErr)
		return
	}
	if resp.Status != http.StatusOK {
		err = errors.Warning("dev: lockers acquire failed")
		return
	}
	locker = &devSharedLocker{
		client: dev.client,
		key:    key,
	}
	return
}

type devSharedLocker struct {
	client transports.Client
	key    []byte
}

func (dev *devSharedLocker) Lock(ctx context.Context) (err error) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, devClusterSharedPath)
	req.Header().Set(httpDevModeHeader, "*")
	subParam := devLockParam{
		Key: bytex.ToString(dev.key),
	}
	subParamBytes, _ := json.Marshal(subParam)
	param := &devShardParam{
		Type:    "locker:lock",
		Payload: subParamBytes,
	}
	paramBytes, _ := json.Marshal(param)
	req.SetBody(paramBytes)
	resp, doErr := dev.client.Do(ctx, req)
	if doErr != nil {
		err = errors.Warning("dev: locker lock failed").WithCause(doErr)
		return
	}
	if resp.Status != http.StatusOK {
		err = errors.Warning("dev: locker lock failed")
		return
	}
	return
}

func (dev *devSharedLocker) Unlock(ctx context.Context) (err error) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, devClusterSharedPath)
	req.Header().Set(httpDevModeHeader, "*")
	subParam := devUnLockParam{
		Key: bytex.ToString(dev.key),
	}
	subParamBytes, _ := json.Marshal(subParam)
	param := &devShardParam{
		Type:    "locker:unlock",
		Payload: subParamBytes,
	}
	paramBytes, _ := json.Marshal(param)
	req.SetBody(paramBytes)
	resp, doErr := dev.client.Do(ctx, req)
	if doErr != nil {
		err = errors.Warning("dev: locker unlock failed").WithCause(doErr)
		return
	}
	if resp.Status != http.StatusOK {
		err = errors.Warning("dev: locker unlock failed")
		return
	}
	return
}

type devSharedStore struct {
	appId  string
	client transports.Client
}

func (dev *devSharedStore) Get(ctx context.Context, key []byte) (value []byte, has bool, err errors.CodeError) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, devClusterSharedPath)
	req.Header().Set(httpDevModeHeader, "*")
	subParam := devStoreGetParam{
		Key: bytex.ToString(key),
	}
	subParamBytes, _ := json.Marshal(subParam)
	param := &devShardParam{
		Type:    "store:get",
		Payload: subParamBytes,
	}
	paramBytes, _ := json.Marshal(param)
	req.SetBody(paramBytes)
	resp, doErr := dev.client.Do(ctx, req)
	if doErr != nil {
		err = errors.Warning("dev: store get failed").WithCause(doErr)
		return
	}
	if resp.Status != http.StatusOK {
		err = errors.Warning("dev: store get failed")
		return
	}
	result := devStoreGetResult{}
	decodeErr := json.Unmarshal(resp.Body, &result)
	if decodeErr != nil {
		err = errors.Warning("dev: store get failed").WithCause(decodeErr)
		return
	}
	value = result.Value
	has = result.Has
	return
}

func (dev *devSharedStore) Set(ctx context.Context, key []byte, value []byte) (err errors.CodeError) {
	err = dev.SetWithTTL(ctx, key, value, 0)
	if err != nil {
		err = errors.Warning("dev: store set failed")
		return
	}
	return
}

func (dev *devSharedStore) SetWithTTL(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err errors.CodeError) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, devClusterSharedPath)
	req.Header().Set(httpDevModeHeader, "*")
	subParam := devStoreSetParam{
		Key:   bytex.ToString(key),
		Value: value,
		TTL:   ttl,
	}
	subParamBytes, _ := json.Marshal(subParam)
	param := &devShardParam{
		Type:    "store:set",
		Payload: subParamBytes,
	}
	paramBytes, _ := json.Marshal(param)
	req.SetBody(paramBytes)
	resp, doErr := dev.client.Do(ctx, req)
	if doErr != nil {
		err = errors.Warning("dev: store set with ttl failed").WithCause(doErr)
		return
	}
	if resp.Status != http.StatusOK {
		err = errors.Warning("dev: store set with ttl failed")
		return
	}
	return
}

func (dev *devSharedStore) Incr(ctx context.Context, key []byte, delta int64) (v int64, err errors.CodeError) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, devClusterSharedPath)
	req.Header().Set(httpDevModeHeader, "*")
	subParam := devStoreIncrParam{
		Key:   bytex.ToString(key),
		Delta: delta,
	}
	subParamBytes, _ := json.Marshal(subParam)
	param := &devShardParam{
		Type:    "store:incr",
		Payload: subParamBytes,
	}
	paramBytes, _ := json.Marshal(param)
	req.SetBody(paramBytes)
	resp, doErr := dev.client.Do(ctx, req)
	if doErr != nil {
		err = errors.Warning("dev: store incr failed").WithCause(doErr)
		return
	}
	if resp.Status != http.StatusOK {
		err = errors.Warning("dev: store incr failed")
		return
	}
	result := devStoreIncrResult{}
	decodeErr := json.Unmarshal(resp.Body, &result)
	if decodeErr != nil {
		err = errors.Warning("dev: store incr failed").WithCause(decodeErr)
		return
	}
	v = result.N
	return
}

func (dev *devSharedStore) ExpireKey(ctx context.Context, key []byte, ttl time.Duration) (err errors.CodeError) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, devClusterSharedPath)
	req.Header().Set(httpDevModeHeader, "*")
	subParam := devStoreExprParam{
		Key: bytex.ToString(key),
		TTL: ttl,
	}
	subParamBytes, _ := json.Marshal(subParam)
	param := &devShardParam{
		Type:    "store:expireKey",
		Payload: subParamBytes,
	}
	paramBytes, _ := json.Marshal(param)
	req.SetBody(paramBytes)
	resp, doErr := dev.client.Do(ctx, req)
	if doErr != nil {
		err = errors.Warning("dev: store expire key failed").WithCause(doErr)
		return
	}
	if resp.Status != http.StatusOK {
		err = errors.Warning("dev: store expire key failed")
		return
	}
	return
}

func (dev *devSharedStore) Remove(ctx context.Context, key []byte) (err errors.CodeError) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, devClusterSharedPath)
	req.Header().Set(httpDevModeHeader, "*")
	subParam := devStoreRemoveParam{
		Key: bytex.ToString(key),
	}
	subParamBytes, _ := json.Marshal(subParam)
	param := &devShardParam{
		Type:    "store:remove",
		Payload: subParamBytes,
	}
	paramBytes, _ := json.Marshal(param)
	req.SetBody(paramBytes)
	resp, doErr := dev.client.Do(ctx, req)
	if doErr != nil {
		err = errors.Warning("dev: store remove failed").WithCause(doErr)
		return
	}
	if resp.Status != http.StatusOK {
		err = errors.Warning("dev: store remove failed")
		return
	}
	return
}

type devSharedCaches struct {
	appId  string
	client transports.Client
}

func (dev *devSharedCaches) Get(ctx context.Context, key []byte) (value []byte, has bool) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, devClusterSharedPath)
	req.Header().Set(httpDevModeHeader, "*")
	subParam := devCacheGetParam{
		Key: bytex.ToString(key),
	}
	subParamBytes, _ := json.Marshal(subParam)
	param := &devShardParam{
		Type:    "cache:get",
		Payload: subParamBytes,
	}
	paramBytes, _ := json.Marshal(param)
	req.SetBody(paramBytes)
	resp, doErr := dev.client.Do(ctx, req)
	if doErr != nil {
		return
	}
	if resp.Status != http.StatusOK {
		return
	}
	result := devCacheGetResult{}
	decodeErr := json.Unmarshal(resp.Body, &result)
	if decodeErr != nil {
		return
	}
	has = result.Has
	if has {
		value = result.Value
	}
	return
}

func (dev *devSharedCaches) Exist(ctx context.Context, key []byte) (has bool) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, devClusterSharedPath)
	req.Header().Set(httpDevModeHeader, "*")
	subParam := devCacheExistParam{
		Key: bytex.ToString(key),
	}
	subParamBytes, _ := json.Marshal(subParam)
	param := &devShardParam{
		Type:    "cache:exist",
		Payload: subParamBytes,
	}
	paramBytes, _ := json.Marshal(param)
	req.SetBody(paramBytes)
	resp, doErr := dev.client.Do(ctx, req)
	if doErr != nil {
		return
	}
	if resp.Status != http.StatusOK {
		return
	}
	result := devCacheExistResult{}
	decodeErr := json.Unmarshal(resp.Body, &result)
	if decodeErr != nil {
		return
	}
	has = result.Has
	return
}

func (dev *devSharedCaches) Set(ctx context.Context, key []byte, value []byte, ttl time.Duration) (ok bool) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, devClusterSharedPath)
	req.Header().Set(httpDevModeHeader, "*")
	subParam := devCacheSetParam{
		Key:   bytex.ToString(key),
		Value: value,
		TTL:   ttl,
	}
	subParamBytes, _ := json.Marshal(subParam)
	param := &devShardParam{
		Type:    "cache:set",
		Payload: subParamBytes,
	}
	paramBytes, _ := json.Marshal(param)
	req.SetBody(paramBytes)
	resp, doErr := dev.client.Do(ctx, req)
	if doErr != nil {
		return
	}
	if resp.Status != http.StatusOK {
		return
	}
	result := devCacheSetResult{}
	decodeErr := json.Unmarshal(resp.Body, &result)
	if decodeErr != nil {
		return
	}
	ok = result.Ok
	return
}

func (dev *devSharedCaches) Remove(ctx context.Context, key []byte) {
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, devClusterSharedPath)
	req.Header().Set(httpDevModeHeader, "*")
	subParam := devCacheRemoveParam{
		Key: bytex.ToString(key),
	}
	subParamBytes, _ := json.Marshal(subParam)
	param := &devShardParam{
		Type:    "cache:remove",
		Payload: subParamBytes,
	}
	paramBytes, _ := json.Marshal(param)
	req.SetBody(paramBytes)
	_, _ = dev.client.Do(ctx, req)
	return
}

func (dev *devSharedStore) Close() {
	return
}
