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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/fns/service/shareds"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	devProxyHandlerName = "dev"
)

func newDevProxyHandler(registrations *Registrations, signer *secret.Signer) *devProxyHandler {
	return &devProxyHandler{
		registrations: registrations,
		signer:        signer,
		log:           nil,
		lockers:       sync.Map{},
	}
}

type devProxyHandler struct {
	registrations *Registrations
	signer        *secret.Signer
	log           logs.Logger
	lockers       sync.Map
}

func (handler *devProxyHandler) Name() (name string) {
	name = devProxyHandlerName
	return
}

func (handler *devProxyHandler) Build(options TransportHandlerOptions) (err error) {
	handler.log = options.Log
	return
}

func (handler *devProxyHandler) Accept(r *transports.Request) (ok bool) {
	if r.Header().Get(httpDevModeHeader) == "" {
		return
	}
	ok = r.IsGet() && bytex.ToString(r.Path()) == "/cluster/nodes"
	if ok {
		return
	}
	ok = r.IsPost() && bytex.ToString(r.Path()) == "/cluster/shared"
	if ok {
		return
	}
	ok = r.IsPost() && r.Header().Get(httpContentType) == httpContentTypeJson &&
		r.Header().Get(httpRequestSignatureHeader) != "" && r.Header().Get(httpDevModeHeader) != "" &&
		len(strings.Split(bytex.ToString(r.Path()), "/")) == 3
	if ok {
		return
	}
	return
}

func (handler *devProxyHandler) Handle(w transports.ResponseWriter, r *transports.Request) {
	if r.IsGet() && bytex.ToString(r.Path()) == "/cluster/nodes" {
		handler.handleClusterNodes(w, r)
		return
	}
	if r.IsPost() && bytex.ToString(r.Path()) == "/cluster/shared" {
		handler.handleShared(w, r)
		return
	}
	if r.IsPost() && r.Header().Get(httpContentType) == httpContentTypeJson &&
		r.Header().Get(httpRequestSignatureHeader) != "" && r.Header().Get(httpDevModeHeader) != "" &&
		len(strings.Split(bytex.ToString(r.Path()), "/")) == 3 {
		handler.handleServiceFn(w, r)
		return
	}
	return
}

func (handler *devProxyHandler) handleClusterNodes(w transports.ResponseWriter, r *transports.Request) {
	nodes := make(Nodes, 0, 1)
	for _, node := range handler.registrations.nodes {
		nodes = append(nodes, node)
	}
	if len(nodes) > 0 {
		sort.Sort(nodes)
	}
	w.Succeed(nodes)
	return
}

type devShardRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (handler *devProxyHandler) handleShared(w transports.ResponseWriter, r *transports.Request) {
	body := r.Body()
	if body == nil || len(body) == 0 {
		w.Failed(errors.Warning("dev: handle shared failed").WithCause(errors.Warning("body is nil")))
		return
	}
	req := devShardRequest{}
	decodeErr := json.Unmarshal(body, &req)
	if decodeErr != nil {
		w.Failed(errors.Warning("dev: handle shared failed").WithCause(decodeErr))
		return
	}
	switch req.Type {
	case "locker:lock":
		handler.handleSharedLock(w, r, req.Payload)
		break
	case "locker:unlock":
		handler.handleSharedUnLock(w, r, req.Payload)
		break
	case "store:get":
		handler.handleSharedStoreGet(w, r, req.Payload)
		break
	case "store:set":
		handler.handleSharedStoreSet(w, r, req.Payload)
		break
	case "store:incr":
		handler.handleSharedStoreIncr(w, r, req.Payload)
		break
	case "store:expireKey":
		handler.handleSharedStoreExpireKey(w, r, req.Payload)
		break
	case "store:remove":
		handler.handleSharedStoreRemove(w, r, req.Payload)
		break
	default:
		w.Failed(errors.Warning("dev: handle shared failed").WithCause(errors.Warning("type is not match")))
		return
	}
	return
}

type devLockParam struct {
	Key string        `json:"key"`
	TTL time.Duration `json:"ttl"`
}

func (handler *devProxyHandler) handleSharedLock(w transports.ResponseWriter, r *transports.Request, payload json.RawMessage) {
	param := devLockParam{}
	decodeErr := json.Unmarshal(payload, &param)
	if decodeErr != nil {
		w.Failed(errors.Warning("dev: locker lock failed").WithCause(decodeErr))
		return
	}
	lockers := handler.registrations.cluster.Shared().Lockers()
	locker, lockerErr := lockers.Acquire(r.Context(), bytex.FromString(param.Key), param.TTL)
	if lockerErr != nil {
		w.Failed(errors.Warning("dev: locker lock failed").WithCause(lockerErr))
		return
	}
	lockErr := locker.Lock(r.Context())
	if lockErr != nil {
		w.Failed(errors.Warning("dev: locker lock failed").WithCause(lockErr))
		return
	}
	handler.lockers.Store(param.Key, locker)
	w.Succeed(map[string]string{"appId": handler.registrations.id})
	return
}

type devUnLockParam struct {
	Key string `json:"key"`
}

func (handler *devProxyHandler) handleSharedUnLock(w transports.ResponseWriter, r *transports.Request, payload json.RawMessage) {
	param := devUnLockParam{}
	decodeErr := json.Unmarshal(payload, &param)
	if decodeErr != nil {
		w.Failed(errors.Warning("dev: locker unlock failed").WithCause(decodeErr))
		return
	}
	x, has := handler.lockers.Load(param.Key)
	if !has {
		w.Failed(errors.Warning("dev: locker unlock failed").WithCause(errors.Warning("locker may be released")))
		return
	}
	locker := x.(shareds.Locker)
	unlockErr := locker.Unlock(r.Context())
	handler.lockers.Delete(param.Key)
	if unlockErr != nil {
		w.Failed(errors.Warning("dev: locker unlock failed").WithCause(unlockErr))
		return
	}
	w.Succeed(&Empty{})
	return
}

type devStoreGetParam struct {
	Key string `json:"key"`
}

type devStoreGetResult struct {
	Has   bool   `json:"has"`
	Value []byte `json:"value"`
}

func (handler *devProxyHandler) handleSharedStoreGet(w transports.ResponseWriter, r *transports.Request, payload json.RawMessage) {
	param := devStoreGetParam{}
	decodeErr := json.Unmarshal(payload, &param)
	if decodeErr != nil {
		w.Failed(errors.Warning("dev: store get failed").WithCause(decodeErr))
		return
	}
	store := handler.registrations.cluster.Shared().Store()
	value, has, getErr := store.Get(r.Context(), bytex.FromString(param.Key))
	if getErr != nil {
		w.Failed(errors.Warning("dev: store get failed").WithCause(getErr))
		return
	}
	w.Succeed(&devStoreGetResult{
		Has:   has,
		Value: value,
	})
	return
}

type devStoreSetParam struct {
	Key   string        `json:"key"`
	Value []byte        `json:"value"`
	TTL   time.Duration `json:"ttl"`
}

func (handler *devProxyHandler) handleSharedStoreSet(w transports.ResponseWriter, r *transports.Request, payload json.RawMessage) {
	param := devStoreSetParam{}
	decodeErr := json.Unmarshal(payload, &param)
	if decodeErr != nil {
		w.Failed(errors.Warning("dev: store set failed").WithCause(decodeErr))
		return
	}
	store := handler.registrations.cluster.Shared().Store()
	var setErr error
	if param.TTL > 0 {
		setErr = store.SetWithTTL(r.Context(), bytex.FromString(param.Key), param.Value, param.TTL)
	} else {
		setErr = store.Set(r.Context(), bytex.FromString(param.Key), param.Value)
	}
	if setErr != nil {
		w.Failed(errors.Warning("dev: store set failed").WithCause(setErr))
		return
	}
	w.Succeed(&Empty{})
	return
}

type devStoreIncrParam struct {
	Key   string `json:"key"`
	Delta int64  `json:"delta"`
}

type devStoreIncrResult struct {
	N int64 `json:"n"`
}

func (handler *devProxyHandler) handleSharedStoreIncr(w transports.ResponseWriter, r *transports.Request, payload json.RawMessage) {
	param := devStoreIncrParam{}
	decodeErr := json.Unmarshal(payload, &param)
	if decodeErr != nil {
		w.Failed(errors.Warning("dev: store incr failed").WithCause(decodeErr))
		return
	}
	store := handler.registrations.cluster.Shared().Store()
	n, incrErr := store.Incr(r.Context(), bytex.FromString(param.Key), param.Delta)
	if incrErr != nil {
		w.Failed(errors.Warning("dev: store incr failed").WithCause(incrErr))
		return
	}
	w.Succeed(&devStoreIncrResult{
		N: n,
	})
	return
}

type devStoreExprParam struct {
	Key string        `json:"key"`
	TTL time.Duration `json:"ttl"`
}

func (handler *devProxyHandler) handleSharedStoreExpireKey(w transports.ResponseWriter, r *transports.Request, payload json.RawMessage) {
	param := devStoreExprParam{}
	decodeErr := json.Unmarshal(payload, &param)
	if decodeErr != nil {
		w.Failed(errors.Warning("dev: store expire key failed").WithCause(decodeErr))
		return
	}
	store := handler.registrations.cluster.Shared().Store()
	err := store.ExpireKey(r.Context(), bytex.FromString(param.Key), param.TTL)
	if err != nil {
		w.Failed(errors.Warning("dev: store expire key failed").WithCause(err))
		return
	}
	w.Succeed(&Empty{})
	return
}

type devStoreRemoveParam struct {
	Key string `json:"key"`
}

func (handler *devProxyHandler) handleSharedStoreRemove(w transports.ResponseWriter, r *transports.Request, payload json.RawMessage) {
	param := devStoreRemoveParam{}
	decodeErr := json.Unmarshal(payload, &param)
	if decodeErr != nil {
		w.Failed(errors.Warning("dev: store remove failed").WithCause(decodeErr))
		return
	}
	store := handler.registrations.cluster.Shared().Store()
	err := store.Remove(r.Context(), bytex.FromString(param.Key))
	if err != nil {
		w.Failed(errors.Warning("dev: store remove failed").WithCause(err))
		return
	}
	w.Succeed(&Empty{})
	return
}

func (handler *devProxyHandler) handleServiceFn(w transports.ResponseWriter, r *transports.Request) {
	appId := r.Header().Get(httpDevModeHeader)
	requestId, hasRequestId := handler.getRequestId(r)
	if !hasRequestId {
		w.Failed(errors.Warning("dev: X-Fns-Request-Id was required in header"))
		return
	}
	// read path
	pathItems := strings.Split(bytex.ToString(r.Path()), "/")
	serviceName := pathItems[1]
	fnName := pathItems[2]
	// read body
	body := r.Body()
	// verify signature
	if !handler.signer.Verify(body, bytex.FromString(r.Header().Get(httpRequestSignatureHeader))) {
		w.Failed(errors.Warning("dev: signature is invalid"))
		return
	}
	registration, has := handler.registrations.GetExact(serviceName, appId)
	if !has {
		w.Failed(errors.NotFound("dev: service was not found").WithMeta("service", serviceName))
		return
	}
	// internal request
	iReq := &internalRequestImpl{}
	decodeErr := json.Unmarshal(body, iReq)
	if decodeErr != nil {
		w.Failed(errors.Warning("dev: decode body failed").WithCause(decodeErr))
		return
	}
	// timeout
	ctx := r.Context()
	var cancel context.CancelFunc
	timeout := r.Header().Get(httpRequestTimeoutHeader)
	if timeout != "" {
		timeoutMillisecond, parseTimeoutErr := strconv.ParseInt(timeout, 10, 64)
		if parseTimeoutErr != nil {
			w.Failed(errors.Warning("dev: X-Fns-Request-Timeout is not number").WithMeta("timeout", timeout))
			return
		}
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMillisecond)*time.Millisecond)
	}
	// read device
	deviceId := handler.getDeviceId(r)
	deviceIp := handler.getDeviceIp(r)
	// request
	req := NewRequest(
		ctx,
		serviceName,
		fnName,
		iReq.Argument,
		WithRequestHeader(r.Header()),
		WithRequestId(requestId),
		WithDeviceId(deviceId),
		WithDeviceIp(deviceIp),
		WithInternalRequest(),
		WithRequestTrunk(iReq.Trunk),
		WithRequestUser(iReq.User.Id(), iReq.User.Attributes()),
		WithRequestVersions(AllowAllRequestVersions()),
	)
	result, requestErr := registration.RequestSync(withTracer(ctx, requestId), req)
	if cancel != nil {
		cancel()
	}
	var span *Span
	tracer_, hasTracer := GetTracer(ctx)
	if hasTracer {
		span = tracer_.RootSpan()
	}
	resp := &internalResponse{
		User:  req.User(),
		Trunk: req.Trunk(),
		Span:  span,
		Body:  nil,
	}
	if requestErr == nil {
		resp.Succeed = true
		resp.Body = result
	} else {
		resp.Succeed = false
		resp.Body = requestErr
	}
	w.Succeed(resp)
	return
}

func (handler *devProxyHandler) getRequestId(r *transports.Request) (requestId string, has bool) {
	requestId = strings.TrimSpace(r.Header().Get(httpRequestIdHeader))
	has = requestId != ""
	return
}

func (handler *devProxyHandler) getDeviceId(r *transports.Request) (devId string) {
	devId = strings.TrimSpace(r.Header().Get(httpDeviceIdHeader))
	return
}

func (handler *devProxyHandler) getDeviceIp(r *transports.Request) (devIp string) {
	devIp = r.Header().Get(httpDeviceIpHeader)
	return
}

func (handler *devProxyHandler) Close() (err error) {
	return
}
