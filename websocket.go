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
	"github.com/aacfactory/json"
	"github.com/fasthttp/websocket"
	"io/ioutil"
	"sync"
	"time"
	"unicode/utf8"
)

type WebsocketConnectionProxy interface {
	Write(ctx Context, response *WebsocketResponse) (err error)
}

type WebsocketConnections interface {
	// 在这里进行监听消息
	Register(ctx Context, conn *WebsocketConnection) (err error)
	Deregister(ctx Context, conn *WebsocketConnection) (err error)
	Proxy(ctx Context, id string) (proxy WebsocketConnectionProxy, has bool, err error)
	Close() (err error)
}

var websocketConnectionsRetriever = localWebsocketConnectionsRetriever

type WebsocketConnectionsRetriever func() (v WebsocketConnections)

func localWebsocketConnectionsRetriever() (v WebsocketConnections) {
	v = newLocalWebsocketConnections()
	return
}

const (
	contextWebsocketConnectionsKey = "_websocket_"
)

func WithWebsocket(ctx Context, connections WebsocketConnections) Context {
	ctx.App().ServiceMeta().Set(contextWebsocketConnectionsKey, connections)
	return ctx
}

func GetWebsocketConnectionProxy(ctx Context, requestId string) (proxy WebsocketConnectionProxy, has bool, err error) {
	connections0, hasConnections := ctx.App().ServiceMeta().Get(contextWebsocketConnectionsKey)
	if !hasConnections {
		err = errors.Warning("there is no websocket in context")
		return
	}
	connections, ok := connections0.(WebsocketConnections)
	if !ok {
		err = errors.Warning("type of websocket in context is not fns.WebsocketConnections")
		return
	}
	proxy, has, err = connections.Proxy(ctx, requestId)
	return
}

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

func newWebsocketConnection(conn *websocket.Conn, svc *services, conns WebsocketConnections) (v *WebsocketConnection, err error) {
	v = &WebsocketConnection{
		online: true,
		mutex:  new(sync.Mutex),
		id:     UID(),
		conn:   conn,
		svc:    svc,
		conns:  conns,
	}
	ctx, _ := newContext(sc.TODO(), true, "-", []byte(""), nil, svc.app)
	err = conns.Register(ctx, v)
	return
}

type WebsocketConnection struct {
	online bool
	mutex  *sync.Mutex
	id     string
	conn   *websocket.Conn
	svc    *services
	conns  WebsocketConnections
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
		timeoutCtx, cancel := sc.WithTimeout(sc.TODO(), conn.svc.fnHandleTimeout)
		ctx, ctxErr := newContext(timeoutCtx, false, conn.id, []byte(request.Authorization), nil, conn.svc.app)
		if ctxErr != nil {
			err = ctxErr
			cancel()
			return
		}
		arg, argErr := NewArgument(request.Argument)
		if argErr != nil {
			err = argErr
			cancel()
			return
		}
		// handle service
		result := conn.svc.Request(WithWebsocket(ctx, conn.conns), request.Service, request.Fn, arg)
		response := &WebsocketResponse{}
		data := json.RawMessage{}
		handleErr := result.Get(ctx, &data)
		cancel()
		if handleErr == nil {
			response.Succeed = true
			response.Data = data
		} else {
			response.Error = handleErr
		}
		p, encodeErr := json.Marshal(response)
		if encodeErr != nil {
			err = encodeErr
			return
		}
		// write response
		writeErr := conn.conn.WriteMessage(websocket.TextMessage, p)
		if writeErr != nil {
			return
		}
	}
}

func (conn *WebsocketConnection) Close() (err error) {
	conn.mutex.Lock()
	conn.online = false
	ctx, _ := newContext(sc.TODO(), true, "-", []byte(""), nil, conn.svc.app)
	deregisterErr := conn.conns.Deregister(ctx, conn)
	if deregisterErr != nil {
		err = deregisterErr
	}
	closeErr := conn.conn.Close()
	if closeErr != nil {
		if err != nil {
			err = fmt.Errorf("close websocket conn failed, %v, %v", err, closeErr)
		} else {
			err = closeErr
		}
	}
	conn.mutex.Unlock()
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
	conn *WebsocketConnection
}

func (proxy *localWebsocketConnectionProxy) Write(_ Context, response *WebsocketResponse) (err error) {
	err = proxy.conn.Write(response)
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
		conn: conn,
	})
	return
}

func (wcs *localWebsocketConnections) Deregister(_ Context, conn *WebsocketConnection) (err error) {
	wcs.items.Delete(conn.Id())
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
