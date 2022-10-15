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

package server

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/fns/internal/configure"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/cespare/xxhash/v2"
	"github.com/fasthttp/websocket"
	"github.com/valyala/bytebufferpool"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

func NewWebsocketHandler() (h Handler) {
	h = &websocketHandler{
		log:       nil,
		handling:  0,
		maxConns:  0,
		counter:   sync.WaitGroup{},
		endpoints: nil,
		upgrader:  nil,
		lock:      sync.RWMutex{},
		closed:    false,
	}
	return
}

type websocketHandler struct {
	log       logs.Logger
	handling  int64
	maxConns  int64
	counter   sync.WaitGroup
	endpoints service.Endpoints
	upgrader  *websocket.Upgrader
	lock      sync.RWMutex
	closed    bool
}

func (h *websocketHandler) Name() (name string) {
	name = "websocket"
	return
}

func (h *websocketHandler) Build(options *HandlerOptions) (err error) {
	config := &configure.Websocket{}
	has, getErr := options.Config.Get("server.websocket", config)
	if getErr != nil {
		err = fmt.Errorf("build websocket handler failed, %v", getErr)
		return
	}
	if !has {
		err = fmt.Errorf("build websocket handler failed, there is no server.websocket in config")
		return
	}
	readBufferSize, _ := commons.ToBytes(config.ReadBufferSize)
	writeBufferSize, _ := commons.ToBytes(config.WriteBufferSize)
	maxConns := config.MaxConns
	if maxConns == 0 {
		maxConns = 65535
	}
	h.upgrader = &websocket.Upgrader{
		ReadBufferSize:  int(readBufferSize),
		WriteBufferSize: int(writeBufferSize),
		WriteBufferPool: nil,
		Subprotocols:    nil,
		Error:           nil,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		EnableCompression: config.EnableCompression,
	}
	h.maxConns = config.MaxConns
	h.endpoints = options.Endpoints
	h.log = options.Log.With("fns", "handler").With("handle", "websocket")
	return
}

func (h *websocketHandler) Handle(writer http.ResponseWriter, request *http.Request) (ok bool) {
	if !(request.Method == http.MethodGet && request.URL.Path == "/websockets") {
		return
	}
	ok = true
	if atomic.AddInt64(&h.handling, 1) > h.maxConns {
		atomic.AddInt64(&h.handling, -1)
		err := errors.NotAcceptable("fns: upgrades the HTTP server connection to the WebSocket protocol failed").WithCause(fmt.Errorf("connections is full"))
		writer.Header().Set(httpServerHeader, httpServerHeaderValue)
		writer.Header().Set(httpContentType, httpContentTypeJson)
		writer.WriteHeader(err.Code())
		p, _ := json.Marshal(err)
		_, _ = writer.Write(p)
		return
	}
	conn, upgradeErr := h.upgrader.Upgrade(writer, request, nil)
	if upgradeErr != nil {
		err := errors.ServiceError("fns: upgrades the HTTP server connection to the WebSocket protocol failed").WithCause(upgradeErr)
		writer.Header().Set(httpServerHeader, httpServerHeaderValue)
		writer.Header().Set(httpContentType, httpContentTypeJson)
		writer.WriteHeader(err.Code())
		p, _ := json.Marshal(err)
		_, _ = writer.Write(p)
		return
	}
	socket := newWebsocketEndpoint(conn)
	// handle
	go func(h *websocketHandler, socket *websocketEndpoint) {
		for {
			closed := false
			select {
			case <-socket.closeCh:
				closed = true
				break
			default:
				h.lock.RLock()
				if h.closed {
					_ = socket.conn.Close()
					closed = true
					h.lock.RUnlock()
					break
				}
				h.lock.RUnlock()
				msg := WebsocketMessage{}
				readErr := socket.conn.ReadJSON(&msg)
				if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
					_ = socket.conn.Close()
					closed = true
					break
				}
				ctx := context.TODO()
				r, requestErr := NewWebsocketRequest(&msg, socket.remoteIp)
				if requestErr != nil {
					_ = socket.Write(ctx, requestErr)
					break
				}
				h.counter.Add(1)
				ctx = setWebsocketEndpoint(ctx, socket)
				ctx = service.SetRequest(ctx, r)
				ctx = service.SetTracer(ctx)
				result, handleErr := h.endpoints.Handle(ctx, r)
				if handleErr == nil {
					if result == nil {
						_ = socket.Write(ctx, nil)
					} else {
						switch result.(type) {
						case []byte:
							_ = socket.Write(ctx, result.([]byte))
							break
						case json.RawMessage:
							_ = socket.Write(ctx, result.(json.RawMessage))
							break
						case stdjson.RawMessage:
							_ = socket.Write(ctx, result.(stdjson.RawMessage))
							break
						default:
							_ = socket.Write(ctx, result)
							break
						}
					}
				} else {
					_ = socket.Write(ctx, handleErr)
				}
				h.counter.Done()
			}
			if closed {
				break
			}
		}
		atomic.AddInt64(&h.handling, -1)
	}(h, socket)
	return
}

func (h *websocketHandler) Close() {
	h.lock.Lock()
	h.closed = true
	h.lock.Unlock()
	h.counter.Wait()
}

type WebsocketEndpoint interface {
	Id() (id string)
	Write(ctx context.Context, message interface{}) (err errors.CodeError)
	Close()
}

func newWebsocketEndpoint(conn *websocket.Conn) (v *websocketEndpoint) {
	remoteIp := conn.RemoteAddr().String()
	if remoteIp != "" {
		if strings.Index(remoteIp, ".") > 0 && strings.Index(remoteIp, ":") > 0 {
			// ip:port
			remoteIp = remoteIp[0:strings.Index(remoteIp, ":")]
		}
	}
	v = &websocketEndpoint{
		id:       uid.UID(),
		conn:     conn,
		remoteIp: remoteIp,
		lock:     sync.Mutex{},
		closed:   false,
		closeCh:  make(chan struct{}, 1),
	}
	return
}

type websocketEndpoint struct {
	id       string
	conn     *websocket.Conn
	remoteIp string
	lock     sync.Mutex
	closed   bool
	closeCh  chan struct{}
}

func (endpoint *websocketEndpoint) Id() (id string) {
	id = endpoint.id
	return
}

func (endpoint *websocketEndpoint) Write(ctx context.Context, message interface{}) (err errors.CodeError) {
	endpoint.lock.Lock()
	defer endpoint.lock.Unlock()
	if endpoint.closed {
		err = errors.Warning("fns: websocket endpoint write message failed").WithCause(fmt.Errorf("conn was closed"))
		return
	}
	if message == nil {
		message = service.Empty{}
	}
	status := 0
	switch message.(type) {
	case errors.CodeError:
		msg := message.(errors.CodeError)
		status = msg.Code()
	case error:
		msg := message.(error)
		status = http.StatusInternalServerError
		message = errors.ServiceError(msg.Error())
	default:
		status = http.StatusOK
	}
	body, bodyErr := json.Marshal(message)
	if bodyErr != nil {
		err = errors.Warning("fns: websocket endpoint write message failed").WithCause(bodyErr)
		return
	}
	msg := WebsocketMessage{
		Header: http.Header{},
		Body:   body,
	}
	msg.Header.Set("status", fmt.Sprintf("%d", status))
	request, hasRequest := service.GetRequest(ctx)
	if hasRequest {
		msg.Header.Set("requestId", request.Id())
		svc, fn := request.Fn()
		msg.Header.Set("service", svc)
		msg.Header.Set("fn", fn)
	}
	p, encodeErr := json.Marshal(msg)
	if encodeErr != nil {
		err = errors.Warning("fns: websocket endpoint write message failed").WithCause(encodeErr)
		return
	}
	writeErr := endpoint.conn.WriteMessage(websocket.TextMessage, p)
	if writeErr != nil {
		err = errors.Warning("fns: websocket endpoint write message failed").WithCause(writeErr)
		return
	}
	return
}

func (endpoint *websocketEndpoint) Close() {
	endpoint.lock.Lock()
	if endpoint.closed {
		endpoint.lock.Unlock()
		return
	}
	endpoint.closed = true
	_ = endpoint.conn.Close()
	endpoint.closeCh <- struct{}{}
	close(endpoint.closeCh)
	endpoint.lock.Unlock()
	return
}

const (
	contextWebsocketKey = "_websocket_"
)

func setWebsocketEndpoint(ctx context.Context, v WebsocketEndpoint) context.Context {
	ctx = context.WithValue(ctx, contextWebsocketKey, v)
	return ctx
}

type WebsocketMessage struct {
	Header http.Header     `json:"header"`
	Body   json.RawMessage `json:"body"`
}

func NewWebsocketRequest(msg *WebsocketMessage, remoteIp string) (r *WebsocketRequest, err errors.CodeError) {
	svc := msg.Header.Get("service")
	if svc == "" {
		err = errors.BadRequest("fns: invalid websocket request message, there is no service in message header")
		return
	}
	fn := msg.Header.Get("fn")
	if fn == "" {
		err = errors.BadRequest("fns: invalid websocket request message, there is no fn in message header")
		return
	}
	buf := bytebufferpool.Get()
	_, _ = buf.Write([]byte(svc + fn))
	authorization := msg.Header.Get("authorization")
	if authorization != "" {
		_, _ = buf.Write([]byte(authorization))
	}
	_, _ = buf.Write(msg.Body)
	hashCode := xxhash.Sum64(buf.Bytes())
	bytebufferpool.Put(buf)
	r = &WebsocketRequest{
		id:            uid.UID(),
		authorization: authorization,
		remoteIp:      remoteIp,
		user:          service.NewRequestUser("", json.NewObject()),
		local:         service.NewRequestLocal(),
		header:        service.NewRequestHeader(msg.Header),
		service:       svc,
		fn:            fn,
		argument:      service.NewArgument(msg.Body),
		hashCode:      hashCode,
	}
	return
}

type WebsocketRequest struct {
	id            string
	authorization string
	remoteIp      string
	user          service.RequestUser
	local         service.RequestLocal
	header        service.RequestHeader
	service       string
	fn            string
	argument      service.Argument
	hashCode      uint64
}

func (r *WebsocketRequest) Id() (id string) {
	id = r.id
	return
}

func (r *WebsocketRequest) Internal() (ok bool) {
	return
}

func (r *WebsocketRequest) Authorization() (v string) {
	v = r.authorization
	return
}

func (r *WebsocketRequest) RemoteIp() (v string) {
	v = r.remoteIp
	return
}

func (r *WebsocketRequest) User() (user service.RequestUser) {
	user = r.user
	return
}

func (r *WebsocketRequest) SetUser(id string, attr *json.Object) {
	r.user = service.NewRequestUser(id, attr)
}

func (r *WebsocketRequest) Local() (local service.RequestLocal) {
	local = r.local
	return
}

func (r *WebsocketRequest) Header() (header service.RequestHeader) {
	header = r.header
	return
}

func (r *WebsocketRequest) Fn() (service string, fn string) {
	service, fn = r.service, r.fn
	return
}

func (r *WebsocketRequest) Argument() (argument service.Argument) {
	argument = r.argument
	return
}

func (r *WebsocketRequest) Hash() (code uint64) {
	code = r.hashCode
	return
}
