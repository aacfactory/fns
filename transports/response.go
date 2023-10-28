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
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"io"
	"net"
)

type ResponseWriter interface {
	Status() int
	SetStatus(status int)
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

const (
	contextResponseKey = "fns:transports:response"
)

func WithResponse(ctx context.Context, response ResponseWriter) context.Context {
	return context.WithValue(ctx, contextResponseKey, response)
}

func LoadResponse(ctx context.Context) ResponseWriter {
	v := ctx.Value(contextResponseKey)
	if v == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: there is no transport response in context")))
		return nil
	}
	r, ok := v.(ResponseWriter)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: runtime in context is not github.com/aacfactory/fns/transports.ResponseWriter")))
		return nil
	}
	return r
}
