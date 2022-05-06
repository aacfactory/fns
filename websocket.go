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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"github.com/fasthttp/websocket"
	"io"
)

type WebsocketConnectionProxy interface {
	Write(ctx Context, response *WebsocketResponse) (err error)
}

type WebsocketConnections interface {
	// 在这里进行监听消息
	Register(ctx Context, conn WebsocketConnection) (err error)
	Deregister(ctx Context, conn WebsocketConnection) (err error)
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
	Authorization string            `json:"authorization"`
	Service       string            `json:"service"`
	Fn            string            `json:"fn"`
	Meta          map[string]string `json:"meta"`
	Argument      json.RawMessage   `json:"argument"`
}

func (r *WebsocketRequest) DecodeArgument(v interface{}) (err error) {
	err = json.Unmarshal(r.Argument, v)
	return
}

type WebsocketResponse struct {
	Succeed bool             `json:"succeed"`
	Error   errors.CodeError `json:"error"`
	Data    interface{}      `json:"data"`
}

func newWebsocketConnection(conn *websocket.Conn, svc *services) *WebsocketConnection {
	return &WebsocketConnection{
		id:   UID(),
		conn: conn,
		svc:  svc,
	}
}

type WebsocketConnection struct {
	id   string
	conn *websocket.Conn
	svc  *services
}

func (conn *WebsocketConnection) Id() string {
	return conn.id
}

func (conn *WebsocketConnection) Handle() (err error) {
	for {
		request := &WebsocketRequest{}
		readErr := conn.conn.ReadJSON(request)
		if readErr == io.ErrUnexpectedEOF {
			break
		}
		// handle service
		// write response

	}
	return
}

func (conn *WebsocketConnection) Close() (err error) {
	err = conn.conn.Close()
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
