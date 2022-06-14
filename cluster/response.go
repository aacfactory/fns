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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
)

type response struct {
	SpanData json.RawMessage `json:"span"`
	Data     json.RawMessage `json:"data"`
}

func (resp *response) HasSpan() (has bool) {
	has = resp.SpanData != nil && len(resp.SpanData) > 0
	return
}

func (resp *response) Span() (span service.Span, err errors.CodeError) {
	span, err = service.DecodeSpan(resp.SpanData)
	return
}

func (resp *response) AsError() (err errors.CodeError) {
	err = errors.Empty()
	decodeErr := json.Unmarshal(resp.Data, err)
	if decodeErr != nil {
		err = errors.Warning("fns: type of remote endpoint response error is not errors.CodeError").WithCause(fmt.Errorf(string(resp.Data)))
	}
	return
}
