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
	"encoding/binary"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cluster"
	"github.com/aacfactory/fns/secret"
	"github.com/aacfactory/json"
	"github.com/valyala/bytebufferpool"
)

type serviceProxyRequest struct {
	ContextData ContextData     `json:"cdata"`
	Argument    json.RawMessage `json:"argument"`
}

func (req *serviceProxyRequest) Encode() (p []byte, err error) {
	content, encodeErr := json.Marshal(req)
	if encodeErr != nil {
		err = errors.Warning("fns: encode service proxy request to json failed").WithCause(encodeErr)
		return
	}
	signature := secret.Sign(content, secretKey)
	head := make([]byte, 8)
	binary.BigEndian.PutUint64(head, uint64(len(signature)))
	buf := bytebufferpool.Get()
	_, _ = buf.Write(head)
	_, _ = buf.Write(signature)
	_, _ = buf.Write(content)
	p = buf.Bytes()
	buf.Reset()
	bytebufferpool.Put(buf)
	return
}

func (req *serviceProxyRequest) Decode(p []byte) (err error) {
	head := p[0:8]
	signatureLen := binary.BigEndian.Uint64(head)
	signature := p[8 : 8+signatureLen]
	body := p[16+signatureLen:]
	if !secret.Verify(body, signature, secretKey) {
		err = fmt.Errorf("fns: verify internal request body failed")
		return
	}
	decodeErr := json.Unmarshal(body, req)
	if decodeErr != nil {
		err = errors.Warning("fns: decode service proxy request from json failed").WithCause(decodeErr)
		return
	}
	return
}

type serviceProxyResponse struct {
	Failed      bool             `json:"failed"`
	ContextData ContextData      `json:"cdata"`
	Span        Span             `json:"span"`
	Result      json.RawMessage  `json:"result"`
	Error       errors.CodeError `json:"error"`
}

// 在这里进行tracer合并, append child
func proxyRequest(ctx Context, registration *cluster.Registration, fn string, argument Argument) (result []byte, err errors.CodeError) {
	// todo

	return
}
