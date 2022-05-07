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

func GetWebsocketConnectionProxy(ctx Context, id string) (proxy WebsocketConnectionProxy, has bool, err error) {
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
	proxy, has, err = connections.Proxy(ctx, id)
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
		id:    UID(),
		conn:  conn,
		svc:   svc,
		conns: conns,
	}
	ctx, _ := newContext(sc.TODO(), true, "-", []byte(""), nil, svc.app)
	err = conns.Register(ctx, v)
	return
}

type WebsocketConnection struct {
	id    string
	conn  *websocket.Conn
	svc   *services
	conns WebsocketConnections
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
	closeErr := conn.conn.Close()
	if closeErr != nil {
		err = closeErr
	}
	ctx, _ := newContext(sc.TODO(), true, "-", []byte(""), nil, conn.svc.app)
	deregisterErr := conn.conns.Deregister(ctx, conn)
	if deregisterErr != nil {
		if err != nil {
			err = fmt.Errorf("close websocket conn failed, %v, %v", err, deregisterErr)
		} else {
			err = deregisterErr
		}
	}
	return
}

func (conn *WebsocketConnection) Write(response *WebsocketResponse) (err error) {
	if response == nil {
		return
	}
	p, encodeErr := json.Marshal(response)
	if encodeErr != nil {
		err = encodeErr
		return
	}
	err = conn.conn.WriteMessage(websocket.TextMessage, p)
	return
}

func newLocalWebsocketConnections() (v WebsocketConnections) {

	return
}

type localWebsocketConnections struct {
}

func (wcs *localWebsocketConnections) Register(ctx Context, conn WebsocketConnection) (err error) {
	//TODO implement me
	panic("implement me")
}

func (wcs *localWebsocketConnections) Deregister(ctx Context, conn WebsocketConnection) (err error) {
	//TODO implement me
	panic("implement me")
}

func (wcs *localWebsocketConnections) Proxy(ctx Context, id string) (proxy WebsocketConnectionProxy, has bool, err error) {
	//TODO implement me
	panic("implement me")
}
