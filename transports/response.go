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

package transports

import (
	"bufio"
	sc "context"
	"github.com/aacfactory/fns/context"
	"io"
	"net"
)

type ResponseWriter interface {
	context.Context
	Status() int
	SetStatus(status int)
	SetCookie(cookie *Cookie)
	Header() Header
	Succeed(v interface{})
	Failed(cause error)
	Write(body []byte) (int, error)
	Body() []byte
	Hijack(func(conn net.Conn, rw *bufio.ReadWriter) (err error)) (async bool, err error)
	Hijacked() bool
}

type WriteBuffer interface {
	io.Writer
	Bytes() []byte
}

func LoadResponseWriter(ctx sc.Context) ResponseWriter {
	w, ok := ctx.(ResponseWriter)
	if ok {
		return w
	}
	return w
}

func TryLoadResponseWriter(ctx sc.Context) (ResponseWriter, bool) {
	w, ok := ctx.(ResponseWriter)
	return w, ok
}

func TryLoadResponseHeader(ctx sc.Context) (Header, bool) {
	w, ok := ctx.(ResponseWriter)
	if !ok {
		return nil, false
	}
	return w.Header(), ok
}
