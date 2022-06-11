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

package cluster

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
)

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
