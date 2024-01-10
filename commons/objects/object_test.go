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

package objects_test

import (
	"fmt"
	"github.com/aacfactory/avro"
	"github.com/aacfactory/fns/commons/avros"
	"github.com/aacfactory/fns/commons/objects"
	"testing"
	"time"
)

func TestValue(t *testing.T) {
	now := time.Now()
	s := objects.New(now)
	fmt.Println(objects.Value[time.Time](s))
	p, _ := avro.Marshal(now)
	s = avros.RawMessage(p)
	fmt.Println(objects.Value[time.Time](s))
}

func TestBytes(t *testing.T) {
	pp, _ := avro.Marshal([]byte("0123456789"))
	r := raw(pp)
	s := objects.New(r)
	p, err := objects.Value[[]byte](s)
	fmt.Println(string(p), err)
}

func raw(pp []byte) any {
	return avros.RawMessage(pp)
}
