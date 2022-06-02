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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cluster"
	"github.com/aacfactory/fns/internal/secret"
	"github.com/aacfactory/json"
	"github.com/valyala/bytebufferpool"
)

type proxyRequest struct {
	ContextData ContextData `json:"cdata"`
	Argument    Argument    `json:"argument"`
}

func encodeProxyRequest(data ContextData, argument Argument) (p []byte, err errors.CodeError) {
	req := &proxyRequest{
		ContextData: data,
		Argument:    argument,
	}
	reqBytes, encodeReqErr := json.Marshal(req)
	if encodeReqErr != nil {
		err = errors.Warning("fns: encode internal request failed").WithCause(encodeReqErr)
		return
	}
	signature := secret.Sign(reqBytes)
	head := make([]byte, 8)
	binary.BigEndian.PutUint64(head, uint64(len(signature)))
	buf := bytebufferpool.Get()
	_, _ = buf.Write(head)
	_, _ = buf.Write(signature)
	_, _ = buf.Write(reqBytes)
	p = buf.Bytes()
	buf.Reset()
	bytebufferpool.Put(buf)
	return
}

func decodeProxyRequest(p []byte) (data ContextData, argument Argument, err errors.CodeError) {
	head := p[0:8]
	signatureLen := binary.BigEndian.Uint64(head)
	signature := p[8 : 8+signatureLen]
	body := p[16+signatureLen:]
	if !secret.Verify(body, signature) {
		err = errors.Warning("fns: verify internal request body failed")
		return
	}
	req := &proxyRequest{}
	decodeErr := json.Unmarshal(body, req)
	if decodeErr != nil {
		err = errors.Warning("fns: decode internal request failed").WithCause(decodeErr)
		return
	}
	data = req.ContextData
	argument = req.Argument
	return
}

type proxyResponse struct {
	Failed bool             `json:"failed"`
	Span   Span             `json:"span"`
	Result json.RawMessage  `json:"result"`
	Error  errors.CodeError `json:"error"`
}

func encodeProxyResponse(failed bool, span Span, result json.RawMessage, cause errors.CodeError) (p []byte, err errors.CodeError) {
	b, encodeErr := json.Marshal(&proxyResponse{
		Failed: failed,
		Span:   span,
		Result: result,
		Error:  cause,
	})
	if encodeErr != nil {
		err = errors.Warning("fns: encode internal response").WithCause(encodeErr)
		return
	}
	p = b
	return
}

func decodeProxyResponse(p []byte) (failed bool, span Span, result json.RawMessage, cause errors.CodeError, err errors.CodeError) {
	response := &proxyResponse{}
	decodeErr := json.Unmarshal(p, response)
	if decodeErr != nil {
		err = errors.Warning("fns: decode internal response failed").WithCause(decodeErr)
		return
	}
	failed = response.Failed
	if failed {
		cause = response.Error
	} else {
		result = response.Result
	}
	span = response.Span
	return
}

// 在这里进行tracer合并, append child
func proxy(ctx Context, span Span, registration *cluster.Registration, fn string, argument Argument) (result []byte, err errors.CodeError) {
	req, reqErr := encodeProxyRequest(ctx.Data(), argument)
	if reqErr != nil {
		err = errors.Warning("fns: proxy failed").WithCause(reqErr)
		return
	}
	respBody, respErr := registration.Request(ctx, fn, ctx.Request().Header().Raw(), req)
	if respErr != nil {
		err = errors.Warning("fns: proxy failed").WithCause(respErr)
		return
	}
	failed, childSpan, resultData, cause, decodeErr := decodeProxyResponse(respBody)
	if decodeErr != nil {
		err = errors.Warning("fns: proxy failed").WithCause(decodeErr)
		return
	}
	span.AppendChild(childSpan)
	if failed {
		err = cause
		return
	}
	result = resultData
	return
}
