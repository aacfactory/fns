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
	"crypto/md5"
	"encoding/hex"
	stdjson "encoding/json"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/fns/internal/cors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/fasthttp/websocket"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

type WebsocketHandlerOptions struct {
	ReadBufferSize    string
	WriteBufferSize   string
	EnableCompression bool
	MaxConns          uint64
	Cors              *cors.Cors
	Log               logs.Logger
	Endpoints         service.Endpoints
}

func NewWebsocketHandler(options WebsocketHandlerOptions) (h Handler) {
	readBufferSize, _ := commons.ToBytes(options.ReadBufferSize)
	writeBufferSize, _ := commons.ToBytes(options.WriteBufferSize)
	maxConns := options.MaxConns
	if maxConns == 0 {
		maxConns = 65535
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  int(readBufferSize),
		WriteBufferSize: int(writeBufferSize),
		WriteBufferPool: nil,
		Subprotocols:    nil,
		Error:           nil,
		CheckOrigin: func(r *http.Request) bool {
			return options.Cors.OriginAllowed(r)
		},
		EnableCompression: options.EnableCompression,
	}
	h = &websocketHandler{
		log:       options.Log.With("fns", "handler").With("handle", "websocket"),
		handling:  0,
		maxConns:  maxConns,
		counter:   sync.WaitGroup{},
		endpoints: options.Endpoints,
		upgrader:  &upgrader,
		lock:      sync.RWMutex{},
		closed:    false,
	}
	return
}

type websocketHandler struct {
	log       logs.Logger
	handling  uint64
	maxConns  uint64
	counter   sync.WaitGroup
	endpoints service.Endpoints
	upgrader  *websocket.Upgrader
	lock      sync.RWMutex
	closed    bool
}

func (h *websocketHandler) Handle(writer http.ResponseWriter, request *http.Request) (ok bool) {
	if !(request.Method == http.MethodGet && request.URL.Path == "/websockets") {
		return
	}
	ok = true
	if atomic.AddUint64(&h.handling, 1) > h.maxConns {
		atomic.AddUint64(&h.handling, -1)
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
		atomic.AddUint64(&h.handling, -1)
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

func NewWebsocketRequest(msg *WebsocketMessage, remoteIp string) (r *websocketRequest, err errors.CodeError) {
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
	hash := md5.New()
	hash.Write([]byte(svc + fn))
	authorization := msg.Header.Get("authorization")
	if authorization != "" {
		hash.Write([]byte(authorization))
	}
	hash.Write(msg.Body)
	hashCode := hex.EncodeToString(hash.Sum(nil))
	r = &websocketRequest{
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

type websocketRequest struct {
	id            string
	authorization string
	remoteIp      string
	user          service.RequestUser
	local         service.RequestLocal
	header        service.RequestHeader
	service       string
	fn            string
	argument      service.Argument
	hashCode      string
}

func (r *websocketRequest) Id() (id string) {
	id = r.id
	return
}

func (r *websocketRequest) Internal() (ok bool) {
	return
}

func (r *websocketRequest) Authorization() (v string) {
	v = r.authorization
	return
}

func (r *websocketRequest) RemoteIp() (v string) {
	v = r.remoteIp
	return
}

func (r *websocketRequest) User() (user service.RequestUser) {
	user = r.user
	return
}

func (r *websocketRequest) SetUser(id string, attr *json.Object) {
	r.user = service.NewRequestUser(id, attr)
}

func (r *websocketRequest) Local() (local service.RequestLocal) {
	local = r.local
	return
}

func (r *websocketRequest) Header() (header service.RequestHeader) {
	header = r.header
	return
}

func (r *websocketRequest) Fn() (service string, fn string) {
	service, fn = r.service, r.fn
	return
}

func (r *websocketRequest) Argument() (argument service.Argument) {
	argument = r.argument
	return
}

func (r *websocketRequest) Hash() (code string) {
	code = r.hashCode
	return
}
