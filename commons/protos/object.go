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
	"google.golang.org/protobuf/proto"
)

type Object[T proto.Message] struct {
	Entry T
}

func (obj Object[T]) Valid() (ok bool) {
	ok = obj.Entry != nil
	return
}

func (obj Object[T]) Unmarshal(dst any) (err error) {
	dst = obj.Entry
	return
}

func (obj Object[T]) Marshal() (p []byte, err error) {
	p, err = proto.Marshal(obj.Entry)
	return
}

func (obj Object[T]) Value() (v any) {
	v = obj.Entry
	return
}
