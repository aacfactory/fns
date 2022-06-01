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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cluster"
	"github.com/aacfactory/fns/commons"
	"github.com/aacfactory/fns/documents"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/fasthttp/websocket"
	"io/ioutil"
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

type WebsocketRequest struct {
	Authorization string          `json:"authorization"`
	Service       string          `json:"service"`
	Fn            string          `json:"fn"`
	Argument      json.RawMessage `json:"argument"`
}

func (r *WebsocketRequest) DecodeArgument(v interface{}) (err error) {
	err = json.Unmarshal(r.Argument, v)
	return
}

type WebsocketResponse struct {
	Succeed bool             `json:"succeed"`
	Error   errors.CodeError `json:"error"`
	Data    json.RawMessage  `json:"data"`
}

type Websocket interface {
	Id() (id string)
	Write(response *WebsocketResponse) (err errors.CodeError)
}

type WebsocketDiscovery interface {
	Register(ctx Context, socket Websocket) (err errors.CodeError)
	Deregister(ctx Context, socket Websocket) (err errors.CodeError)
	Close() (err error)
}

func newMemoryWebsocketDiscovery(log logs.Logger) *memoryWebsocketDiscovery {
	return &memoryWebsocketDiscovery{
		log: log,
	}
}

type memoryWebsocketDiscovery struct {
	log logs.Logger
}

func (discovery *memoryWebsocketDiscovery) Register(ctx Context, socket Websocket) (err errors.CodeError) {
	//TODO implement me
	panic("implement me")
}

func (discovery *memoryWebsocketDiscovery) Deregister(ctx Context, socket Websocket) (err errors.CodeError) {
	//TODO implement me
	panic("implement me")
}

func (discovery *memoryWebsocketDiscovery) Close() (err error) {
	//TODO implement me
	panic("implement me")
}

func newClusterWebsocketDiscovery(log logs.Logger, manager *cluster.Manager) *clusterWebsocketDiscovery {
	return &clusterWebsocketDiscovery{
		log:     log,
		manager: manager,
	}
}

type clusterWebsocketDiscovery struct {
	log     logs.Logger
	manager *cluster.Manager
}

func (discovery *clusterWebsocketDiscovery) Register(ctx Context, socket Websocket) (err errors.CodeError) {
	//TODO implement me
	panic("implement me")
}

func (discovery *clusterWebsocketDiscovery) Deregister(ctx Context, socket Websocket) (err errors.CodeError) {
	//TODO implement me
	panic("implement me")
}

func (discovery *clusterWebsocketDiscovery) Close() (err error) {
	//TODO implement me
	panic("implement me")
}

// +-------------------------------------------------------------------------------------------------------------------+

// +-------------------------------------------------------------------------------------------------------------------+

type websocketService struct {
	discovery WebsocketDiscovery
}

func (service *websocketService) Name() (name string) {
	name = "websocket"
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

func (service *websocketService) Handle(context Context, fn string, argument Argument, result ResultWriter) {
	//TODO implement me
	panic("implement me")
}

func (service *websocketService) Shutdown(_ sc.Context) (err error) {
	//TODO implement me
	panic("implement me")
}

// todo 做成 service

type WebsocketConnectionProxy interface {
	Write(ctx Context, response *WebsocketResponse) (err error)
}

type WebsocketConnections interface {
	// 在这里进行监听消息
	Register(ctx Context, conn *WebsocketConnection) (err error)
	Deregister(ctx Context, conn *WebsocketConnection) (err error)
	GetLocal(id string) (conn *WebsocketConnection, has bool)
	Proxy(ctx Context, id string) (proxy WebsocketConnectionProxy, has bool, err error)
	Close() (err error)
}

var websocketConnectionsRetriever = localWebsocketConnectionsRetriever

type WebsocketConnectionsRetriever func() (v WebsocketConnections)

func localWebsocketConnectionsRetriever() (v WebsocketConnections) {
	v = newLocalWebsocketConnections()
	return
}

func newWebsocketResponseChan() *WebsocketResponseChan {
	return &WebsocketResponseChan{
		ch:     make(chan *WebsocketResponse, 8),
		closed: false,
		mutex:  sync.Mutex{},
	}
}

type WebsocketResponseChan struct {
	ch     chan *WebsocketResponse
	closed bool
	mutex  sync.Mutex
}

func (wrc *WebsocketResponseChan) close() {
	wrc.mutex.Lock()
	defer wrc.mutex.Unlock()
	if wrc.closed {
		return
	}
	close(wrc.ch)
	wrc.closed = true
}

func (wrc *WebsocketResponseChan) send(response *WebsocketResponse) (ok bool) {
	wrc.mutex.Lock()
	defer wrc.mutex.Unlock()
	if wrc.closed {
		return
	}
	wrc.ch <- response
	ok = true
	return
}

type WebsocketConnection struct {
	online     bool
	address    string
	mutex      *sync.Mutex
	id         string
	conn       *websocket.Conn
	responseCh *WebsocketResponseChan
}

func (conn *WebsocketConnection) listen() {
	go func(conn *WebsocketConnection) {
		for {
			response, ok := <-conn.responseCh.ch
			if !ok {
				break
			}
			_ = conn.Write(response)
		}
	}(conn)
}

func (conn *WebsocketConnection) Id() string {
	return conn.id
}

func (conn *WebsocketConnection) Handle() (err error) {
	for {
		_, reader, nextReaderErr := conn.conn.NextReader()
		if nextReaderErr != nil {
			err = nextReaderErr
			return
		}
		message, readErr := ioutil.ReadAll(reader)
		if readErr != nil {
			err = readErr
			return
		}
		if !utf8.Valid(message) {
			err = errors.Warning("invalid utf8")
			return
		}
		request := &WebsocketRequest{}
		decodeErr := json.Unmarshal(message, request)
		if decodeErr != nil {
			err = readErr
			return
		}

		// write response
		writeErr := conn.conn.WriteMessage(websocket.TextMessage, nil)
		if writeErr != nil {
			return
		}
	}
}

func (conn *WebsocketConnection) Close() (err error) {

	return
}

func (conn *WebsocketConnection) Write(response *WebsocketResponse) (err error) {
	conn.mutex.Lock()
	if !conn.online {
		err = fmt.Errorf("fns Http: websocket connection is closed")
		conn.mutex.Unlock()
		return
	}
	if response == nil {
		conn.mutex.Unlock()
		return
	}
	p, encodeErr := json.Marshal(response)
	if encodeErr != nil {
		err = encodeErr
		conn.mutex.Unlock()
		return
	}
	err = conn.conn.WriteMessage(websocket.TextMessage, p)
	conn.mutex.Unlock()
	return
}

type localWebsocketConnectionProxy struct {
	ch *WebsocketResponseChan
}

func (proxy *localWebsocketConnectionProxy) Write(_ Context, response *WebsocketResponse) (err error) {
	ok := proxy.ch.send(response)
	if !ok {
		err = fmt.Errorf("fns Http: websocket connection proxy is closed")
	}
	return
}

type remoteWebsocketConnectionProxy struct {
	id    string
	proxy *serviceProxy
}

func (proxy *remoteWebsocketConnectionProxy) Write(ctx Context, response *WebsocketResponse) (err error) {

	return
}

func newLocalWebsocketConnections() (v WebsocketConnections) {
	v = &localWebsocketConnections{
		items: &sync.Map{},
	}
	return
}

type localWebsocketConnections struct {
	items *sync.Map
}

func (wcs *localWebsocketConnections) Register(_ Context, conn *WebsocketConnection) (err error) {
	wcs.items.Store(conn.Id(), &localWebsocketConnectionProxy{
		ch: conn.responseCh,
	})
	return
}

func (wcs *localWebsocketConnections) Deregister(_ Context, conn *WebsocketConnection) (err error) {
	wcs.items.Delete(conn.Id())
	return
}

func (wcs *localWebsocketConnections) GetLocal(id string) (conn *WebsocketConnection, has bool) {
	item, exist := wcs.items.Load(id)
	if !exist {
		return
	}
	conn, has = item.(*WebsocketConnection)
	return
}

func (wcs *localWebsocketConnections) Proxy(_ Context, id string) (proxy WebsocketConnectionProxy, has bool, err error) {
	v, exist := wcs.items.Load(id)
	if !exist {
		return
	}
	proxy, has = v.(*localWebsocketConnectionProxy)
	return
}

func (wcs *localWebsocketConnections) Close() (err error) {
	conns := make([]*WebsocketConnection, 0, 1)
	wcs.items.Range(func(_, value interface{}) bool {
		conns = append(conns, value.(*WebsocketConnection))
		return true
	})
	for _, conn := range conns {
		_ = conn.conn.WriteControl(websocket.CloseNormalClosure, []byte("close"), time.Time{})
		_ = conn.Close()
	}
	return
}
