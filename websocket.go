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

type WebsocketConnection interface {
	Read() (p []byte, closed bool, err error)
	Write(p []byte) (err error)
	Close() (err error)
}

type WebsocketContext interface {
	Context
	SocketId() (id string)
	WriteTo(socketId string, p []byte) (err error)
	Close() (err error)
}

func WithWebsocket(ctx Context, connection WebsocketConnection) (wsCtx WebsocketContext) {

	return
}

func MapToWebsocketContext(ctx Context) (wsCtx WebsocketContext, ok bool) {

	return
}

type WebsocketHandler interface {
	Handle(ctx Context, connection WebsocketConnection) (err error)
}
