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

package services_test

import (
	"context"
	"fmt"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/json"
	"testing"
)

func TestFutureResult_MarshalJSON(t *testing.T) {
	p, f := services.NewFuture()
	v := func() (v []string) {
		return
	}()
	p.Succeed(v)
	r, getErr := f.Get(context.TODO())
	if getErr != nil {
		t.Errorf("%+v", getErr)
		return
	}
	b, encodeErr := json.Marshal(r)
	if encodeErr != nil {
		t.Errorf("%+v", encodeErr)
		return
	}
	fmt.Println(string(b))
}
