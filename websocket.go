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
	sc "context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cluster"
	"github.com/aacfactory/fns/documents"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/fasthttp/websocket"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// +-------------------------------------------------------------------------------------------------------------------+

func newWebsocketUpgrader(config WebsocketConfig) (v *websocket.Upgrader) {
	if config.HandshakeTimeoutSeconds <= 0 {
		config.HandshakeTimeoutSeconds = 10
	}
	readBufferSize := 4 * KB
	if config.ReadBufferSize != "" {
		bs := strings.ToUpper(strings.TrimSpace(config.ReadBufferSize))
		if bs != "" {
			bs0, bsErr := commons.ToBytes(bs)
			if bsErr == nil {
				readBufferSize = int(bs0)
			}
		}
	}
	writeBufferSize := 4 * KB
	if config.WriteBufferSize != "" {
		bs := strings.ToUpper(strings.TrimSpace(config.WriteBufferSize))
		if bs != "" {
			bs0, bsErr := commons.ToBytes(bs)
			if bsErr == nil {
				writeBufferSize = int(bs0)
			}
		}
	}
	v = &websocket.Upgrader{
		HandshakeTimeout: time.Duration(config.HandshakeTimeoutSeconds) * time.Second,
		ReadBufferSize:   readBufferSize,
		WriteBufferSize:  writeBufferSize,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type websocketRequest struct {
	Authorization string          `json:"authorization"`
	Service       string          `json:"service"`
	Fn            string          `json:"fn"`
	Argument      json.RawMessage `json:"argument"`
}

func (r *websocketRequest) Hash() (v string) {
	if r.Authorization == "" && (r.Argument == nil || len(r.Argument) == 0) {
		v = "empty"
		return
	}
	hash := md5.New()
	if r.Authorization != "" {
		hash.Write([]byte(r.Authorization))
	}
	hash.Write(r.Argument)
	v = hex.EncodeToString(hash.Sum(nil))
	return
}

func (r *websocketRequest) DecodeArgument(v interface{}) (err error) {
	err = json.Unmarshal(r.Argument, v)
	return
}

type websocketResponse struct {
	Succeed bool             `json:"succeed"`
	Error   errors.CodeError `json:"error"`
	Data    json.RawMessage  `json:"data"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type websocketClient interface {
	Write(response *websocketResponse) (err error)
}

type websocketConnection struct {
	env                  Environments
	id                   string
	mutex                sync.Mutex
	disconnected         bool
	conn                 *websocket.Conn
	cancel               func()
	discovery            *websocketDiscovery
	counter              sync.WaitGroup
	runtime              Runtime
	barrier              Barrier
	requestHandleTimeout time.Duration
	tracerReporter       TracerReporter
	hooks                *hooks
}

func (socket *websocketConnection) Id() (id string) {
	id = socket.id
	return
}

func (socket *websocketConnection) Write(response *websocketResponse) (err error) {
	if response == nil {
		err = errors.Warning("fns: websocket write response failed for response is nil")
		return
	}
	socket.mutex.Lock()
	defer socket.mutex.Unlock()
	if socket.disconnected {
		err = errors.Warning("fns: websocket write response failed for it is disconnected")
		return
	}
	p, encodeErr := json.Marshal(response)
	if encodeErr != nil {
		err = errors.Warning("fns: websocket write response failed for encode response").WithCause(encodeErr)
		return
	}
	socket.counter.Add(1)
	writer, writerErr := socket.conn.NextWriter(websocket.TextMessage)
	socket.counter.Done()
	if writerErr != nil {
		err = errors.Warning("fns: websocket write response failed for can not get writer").WithCause(writerErr)
		return
	}
	_, writeErr := writer.Write(p)
	if writeErr != nil {
		_ = writer.Close()
		err = errors.Warning("fns: websocket write response failed").WithCause(writeErr)
		return
	}
	_ = writer.Close()
	return
}

func (socket *websocketConnection) Listen() {
	ctx, cancel := sc.WithCancel(sc.TODO())
	socket.cancel = cancel
	go func(ctx sc.Context, socket *websocketConnection) {
		closed := false
		for {
			if closed {
				break
			}
			select {
			case <-ctx.Done():
				closed = true
				break
			default:
				if !socket.env.Running() {
					_ = socket.conn.WriteControl(
						websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "service is closing"),
						time.Time{},
					)
					socket.Close()
					closed = true
					break
				}
				socket.handle()
			}
		}
	}(ctx, socket)
	return
}

func (socket *websocketConnection) handle() {
	defer func(socket *websocketConnection) {
		err := recover()
		if err != nil {
			socket.Close()
		}
	}(socket)
	mt, p, readErr := socket.conn.ReadMessage()
	if readErr != nil {
		if readErr != io.EOF {
			_ = socket.conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "read failed"),
				time.Time{},
			)
		}
		socket.Close()
		return
	}
	switch mt {
	case websocket.TextMessage, websocket.BinaryMessage:
		if !utf8.Valid(p) {
			_ = socket.conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseInvalidFramePayloadData, ""),
				time.Time{})
			socket.Close()
			return
		}
		break
	case websocket.PingMessage:
		_ = socket.conn.WriteControl(websocket.PongMessage, []byte("pong"), time.Time{})
		return
	default:
		_ = socket.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "message type is unsupported"),
			time.Time{},
		)
		socket.Close()
		return
	}
	// handle
	wr := &websocketRequest{}
	decodeErr := json.Unmarshal(p, wr)
	if decodeErr != nil {
		_ = socket.Write(&websocketResponse{
			Succeed: false,
			Error:   errors.NotAcceptable("fns: decode request message failed").WithCause(decodeErr),
			Data:    nil,
		})
		return
	}
	service := strings.TrimSpace(wr.Service)
	if service == "" {
		_ = socket.Write(&websocketResponse{
			Succeed: false,
			Error:   errors.BadRequest("fns: there is no service in message"),
			Data:    nil,
		})
		return
	}
	fn := strings.TrimSpace(wr.Fn)
	if fn == "" {
		_ = socket.Write(&websocketResponse{
			Succeed: false,
			Error:   errors.BadRequest("fns: there is no fn in message"),
			Data:    nil,
		})
		return
	}
	authorization := strings.TrimSpace(wr.Authorization)
	arg := NewArgument(wr.Argument)
	timeoutCtx, cancel := sc.WithTimeout(sc.TODO(), socket.requestHandleTimeout)
	ctx := newContext(timeoutCtx, newWebsocketRequest(authorization), newContextData(json.NewObject()), socket.runtime)
	// handle
	socket.counter.Add(1)
	// endpoint
	endpoint, getEndpointErr := socket.runtime.Endpoints().Get(ctx, service)
	if getEndpointErr != nil {
		_ = socket.Write(&websocketResponse{
			Succeed: false,
			Error:   getEndpointErr,
			Data:    nil,
		})
		socket.counter.Done()
		cancel()
		return
	}
	barrierKey := fmt.Sprintf("%s:%s:%s", service, fn, wr.Hash())
	handleResult, handleErr, _ := socket.barrier.Do(ctx, barrierKey, func() (v interface{}, err error) {
		result := endpoint.Request(ctx, fn, arg)
		resultBytes := json.RawMessage{}
		has, getErr := result.Get(ctx, &resultBytes)
		if getErr != nil {
			err = getErr
			return
		}
		if has {
			v = resultBytes
		}
		return
	})
	socket.barrier.Forget(ctx, barrierKey)
	cancel()
	var responseBody json.RawMessage = nil
	if handleResult != nil {
		responseBody = handleResult.(json.RawMessage)
	}
	var codeErr errors.CodeError = nil
	if handleErr != nil {
		codeErr0, ok := handleErr.(errors.CodeError)
		if ok {
			codeErr = codeErr0
		} else {
			codeErr = errors.Warning("fns: handle request failed").WithCause(handleErr)
		}
	}
	_ = socket.Write(&websocketResponse{
		Succeed: codeErr == nil,
		Error:   codeErr,
		Data:    responseBody,
	})
	// done
	socket.counter.Done()
	// report tracer
	socket.tracerReporter.Report(ctx.Fork(sc.TODO()), ctx.Tracer())
	// hook
	socket.hooks.send(newHookUnit(ctx, service, fn, responseBody, codeErr, ctx.tracer.RootSpan().Latency()))
}

func (socket *websocketConnection) Close() {
	socket.mutex.Lock()
	if socket.disconnected {
		socket.mutex.Unlock()
		return
	}
	socket.disconnected = true
	socket.mutex.Unlock()
	socket.discovery.Deregister(socket)
	socket.cancel()
	socket.counter.Wait()
	_ = socket.conn.Close()
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type websocketProxy struct {
	id           string
	ctx          Context
	registration *cluster.Registration
}

func (socket *websocketProxy) Write(response *websocketResponse) (err error) {
	p, encodeErr := json.Marshal(response)
	if encodeErr != nil {
		err = errors.Warning("fns: encode websocket response failed").WithCause(encodeErr)
		return
	}
	obj := json.NewObjectFromBytes(p)
	_ = obj.Put("destinationSocketIds", []string{socket.id})
	span := socket.ctx.Tracer().StartSpan(socket.registration.Name, "send")
	span.AddTag("remote", socket.registration.Address)
	_, proxyErr := proxy(socket.ctx, span, socket.registration, "send", NewArgument(obj.Raw()))
	span.Finish()
	if proxyErr != nil {
		err = errors.Warning("fns: send message to websocket failed").WithCause(proxyErr)
		return
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type websocketOptions struct {
	env                  Environments
	config               WebsocketConfig
	runtime              Runtime
	barrier              Barrier
	requestHandleTimeout time.Duration
	tracerReporter       TracerReporter
	hooks                *hooks
	clusterManager       *cluster.Manager
}

func newWebsocketManager(options websocketOptions) (manager *websocketManager) {

	manager = &websocketManager{
		env:      options.env,
		log:      options.env.Log().With("fns", "websocket").With("websocket", "manager"),
		upgrader: newWebsocketUpgrader(options.config),
		discovery: &websocketDiscovery{
			log:            options.env.Log().With("fns", "websocket").With("websocket", "discovery"),
			connections:    sync.Map{},
			clusterManager: options.clusterManager,
		},
		counter:              sync.WaitGroup{},
		runtime:              options.runtime,
		barrier:              options.barrier,
		requestHandleTimeout: options.requestHandleTimeout,
		tracerReporter:       options.tracerReporter,
		hooks:                options.hooks,
	}
	return
}

type websocketManager struct {
	env                  Environments
	log                  logs.Logger
	upgrader             *websocket.Upgrader
	discovery            *websocketDiscovery
	counter              sync.WaitGroup
	runtime              Runtime
	barrier              Barrier
	requestHandleTimeout time.Duration
	tracerReporter       TracerReporter
	hooks                *hooks
}

func (manager *websocketManager) Upgrade(response http.ResponseWriter, request *http.Request) (err errors.CodeError) {
	if !manager.env.Running() {
		err = errors.Unavailable("fns: service is unavailable").WithMeta("fns", "websocket")
		return
	}
	conn, connErr := manager.upgrader.Upgrade(response, request, nil)
	if connErr != nil {
		err = errors.NotAcceptable("fns: upgrade to websocket failed").WithCause(connErr)
		return
	}
	socket := &websocketConnection{
		env:                  manager.env,
		id:                   UID(),
		mutex:                sync.Mutex{},
		disconnected:         false,
		conn:                 conn,
		cancel:               nil,
		discovery:            manager.discovery,
		counter:              sync.WaitGroup{},
		runtime:              manager.runtime,
		barrier:              manager.barrier,
		requestHandleTimeout: manager.requestHandleTimeout,
		tracerReporter:       manager.tracerReporter,
		hooks:                manager.hooks,
	}
	socket.Listen()
	manager.discovery.Register(socket)
	return
}

func (manager *websocketManager) Service() (service Service) {
	service = &websocketService{
		discovery: manager.discovery,
	}
	return
}

func (manager *websocketManager) Close() {
	manager.discovery.Close()
}

// +-------------------------------------------------------------------------------------------------------------------+

type websocketDiscovery struct {
	log            logs.Logger
	connections    sync.Map
	clusterManager *cluster.Manager
}

func (discovery *websocketDiscovery) Register(socket *websocketConnection) {
	discovery.connections.Store(socket.id, socket)
	if discovery.clusterManager != nil {
		discovery.clusterManager.SaveResource(fmt.Sprintf("ws:%s", socket.id), []byte(discovery.clusterManager.Node().Id))
	}
}

func (discovery *websocketDiscovery) Deregister(socket *websocketConnection) {
	discovery.connections.Delete(socket.id)
	if discovery.clusterManager != nil {
		discovery.clusterManager.RemoveResource(fmt.Sprintf("ws:%s", socket.id))
	}
}

func (discovery *websocketDiscovery) Get(ctx Context, socketId string) (socket websocketClient, has bool) {
	value, exist := discovery.connections.Load(socketId)
	if exist {
		socket, has = value.(*websocketConnection)
		return
	}
	nodeId, hasProxy := discovery.clusterManager.Registrations().GetNodeResource(fmt.Sprintf("ws:%s", socketId))
	if !hasProxy {
		return
	}
	registration, hasRegistration := discovery.clusterManager.Registrations().GetRegistration("websockets", string(nodeId))
	if !hasRegistration {
		return
	}
	socket = &websocketProxy{
		id:           socketId,
		ctx:          ctx,
		registration: registration,
	}
	return
}

func (discovery *websocketDiscovery) Close() {
	sockets := make([]*websocketConnection, 0, 1)
	discovery.connections.Range(func(key, value interface{}) bool {
		sockets = append(sockets, value.(*websocketConnection))
		return true
	})
	for _, socket := range sockets {
		discovery.Deregister(socket)
		socket.Close()
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

// +-------------------------------------------------------------------------------------------------------------------+

type websocketService struct {
	discovery *websocketDiscovery
}

func (service *websocketService) Name() (name string) {
	name = "websockets"
	return
}

func (service *websocketService) Internal() (internal bool) {
	internal = true
	return
}

func (service *websocketService) Build(_ Environments) (err error) {
	return
}

func (service *websocketService) Components() (components map[string]ServiceComponent) {
	return
}

func (service *websocketService) Document() (doc *documents.Service) {
	return
}

func (service *websocketService) Handle(ctx Context, fn string, argument Argument, result ResultWriter) {
	switch fn {
	case "send":
		results, err := service.handleSend(ctx, argument)
		if err == nil {
			result.Succeed(results)
		} else {
			result.Failed(err)
		}
		break
	default:
		result.Failed(errors.NotFound(fmt.Sprintf("fns: there is no named %s fn in websockets service", fn)))
		break
	}
}

func (service *websocketService) Shutdown(_ sc.Context) (err error) {
	return
}

func (service *websocketService) handleSend(ctx Context, argument Argument) (results []*websocketSendResult, err errors.CodeError) {
	if argument.IsNil() {
		err = errors.BadRequest("fns: can not send nothing to websocket connection")
		return
	}
	message := &websocketSendMessage{}
	decodeErr := argument.As(message)
	if decodeErr != nil {
		err = errors.BadRequest("fns: decode message failed").WithCause(decodeErr)
		return
	}
	ids := message.DestinationSocketIds
	if ids == nil || len(ids) == 0 {
		err = errors.BadRequest("fns: no websocket connections to send")
		return
	}
	for _, id := range ids {
		socket, has := service.discovery.Get(ctx, id)
		if !has {
			continue
		}
		writeErr := socket.Write(&websocketResponse{
			Succeed: message.Succeed,
			Error:   message.Error,
			Data:    message.Data,
		})
		results = append(results, &websocketSendResult{
			Id:      id,
			Succeed: writeErr == nil,
		})
	}
	return
}

type websocketSendMessage struct {
	DestinationSocketIds []string         `json:"destinationSocketIds"`
	Succeed              bool             `json:"succeed"`
	Data                 json.RawMessage  `json:"data"`
	Error                errors.CodeError `json:"error"`
}

type websocketSendResult struct {
	Id      string `json:"id"`
	Succeed bool   `json:"succeed"`
}
