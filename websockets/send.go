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

package websockets

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns"
	"github.com/aacfactory/json"
)

type Message struct {
	DestinationSocketIds []string         `json:"destinationSocketIds"`
	Succeed              bool             `json:"succeed"`
	Data                 json.RawMessage  `json:"data"`
	Error                errors.CodeError `json:"error"`
}

type Affected struct {
	Id      string `json:"id"`
	Succeed bool   `json:"succeed"`
}

func Send(ctx fns.Context, msg Message) (affects []*Affected, err errors.CodeError) {
	endpoint, getErr := ctx.Runtime().Endpoints().Get(ctx, "websockets")
	if getErr != nil {
		err = errors.Warning("fns: can not find websockets endpoint").WithCause(getErr)
		return
	}
	result := endpoint.Request(ctx, "send", fns.NewArgument(&msg))
	_, resultErr := result.Get(ctx, &affects)
	if resultErr != nil {
		err = resultErr
		return
	}
	return
}
