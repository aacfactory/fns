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

package protos

import (
	"fmt"
	"github.com/aacfactory/errors"
	"google.golang.org/protobuf/proto"
)

type RawMessage []byte

func (raw RawMessage) Valid() (ok bool) {

	ok = len(raw) > 0
	return
}

func (raw RawMessage) Unmarshal(dst any) (err error) {
	msg, ok := dst.(proto.Message)
	if !ok {
		err = errors.Warning("fns: proto raw message unmarshal failed").WithCause(fmt.Errorf("dst type is not proto.Message"))
		return
	}
	err = proto.Unmarshal(raw, msg)
	return
}

func (raw RawMessage) Marshal() (p []byte, err error) {
	p = raw
	return
}

func (raw RawMessage) Value() (v any) {
	v = raw
	return
}
