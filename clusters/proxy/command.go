/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package proxy

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
)

type Command struct {
	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload"`
}

func ParseCommand(r transports.Request) (cmd Command, err error) {
	body, bodyErr := r.Body()
	if bodyErr != nil {
		err = errors.Warning("fns: parse proxy command failed").WithCause(bodyErr)
		return
	}
	err = json.Unmarshal(body, &cmd)
	if err != nil {
		err = errors.Warning("fns: parse proxy command failed").WithCause(err)
		return
	}
	return
}

func encodeCommand(cmd Command, signature signatures.Signature) (body []byte, sign []byte, err error) {
	body, err = json.Marshal(cmd)
	if err != nil {
		err = errors.Warning("fns: encode proxy command failed").WithCause(err)
		return
	}
	sign = signature.Sign(body)
	return
}
