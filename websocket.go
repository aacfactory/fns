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
	"github.com/fasthttp/websocket"
	"io"
)

type WebsocketConnection interface {
	Read() (p []byte, closed bool, err error)
	Write(p []byte) (err error)
	Close() (err error)
}

type WebsocketContext interface {
	Context
	ConnectionId() (id string)
	Write(p []byte) (err error)
	WriteTo(socketId string, p []byte) (err error)
	Close() (err error)
}

func WithWebsocket(ctx Context, connection WebsocketConnection) (wsCtx WebsocketContext) {

	return
}

func MapToWebsocketContext(ctx Context) (wsCtx WebsocketContext, ok bool) {

	return
}

type WebsocketConnectionAgent interface {
	ConnectionId() string
	Register() (err error)
	Deregister() (err error)
	Send(ctx Context, p []byte) (err error)
}

func newLocalWebsocketConnectionAgent() {

}

type localWebsocketConnectionProxy struct {
}

func handleWebsocketConnection(conn *websocket.Conn, svc *services, proxy WebsocketConnectionAgent) {

	// 全在这里处理，fn中使用WithSocketDestinations([]id)来发送定向结果，当有定向时且正确时，当前链接返回空。

	// 使用GetWebsocketConnectionId(ctx)来获取链接id。

	socketId := UID()
	// proxy

	for {
		messageType, reader, nextReaderErr := conn.NextReader()
		if nextReaderErr != nil {
			if nextReaderErr != io.EOF {
				if svc.log.ErrorEnabled() {
					svc.log.Error().Cause(nextReaderErr).With("websocket", "next_reader").Message("fns Http: websocket get reader failed")
				}
			}
			break
		}

		if messageType == websocket.TextMessage {
			r = &validator{r: r}
		}
	}
	// wait proxy close

	_ = conn.Close()
}

type WebsocketHandler struct {
	socketId string
	svc      *services
}
