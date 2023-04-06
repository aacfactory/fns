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
	"github.com/aacfactory/errors"
	"net"
)

type ResponseWriter interface {
	Status() int
	SetStatus(status int)
	Header() Header
	Succeed(v interface{})
	Failed(cause errors.CodeError)
	Hijack(func(conn net.Conn, brw *bufio.ReadWriter, err error))
	Hijacked() bool
}

type Response struct {
	Status int
	Header Header
	Body   []byte
}
