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

package fns_test

import (
	"context"
	"fmt"
	"github.com/aacfactory/fns"
	"github.com/aacfactory/json"
	"testing"
)

type sampleResult struct {
	Id int
}

func TestAsyncResult(t *testing.T) {
	p, _ := json.Marshal(&sampleResult{
		Id: 1,
	})
	r := fns.AsyncResult()
	r.Succeed(p)
	v := json.RawMessage(make([]byte, 0))
	err := r.Get(context.TODO(), &v)
	fmt.Println(string(v), err)
}
