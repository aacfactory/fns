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

package operators_test

import (
	"fmt"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/context/operators"
	"testing"
)

func TestMap(t *testing.T) {
	nn := []int{1, 2, 3, 4, 5}
	ss, ssErr := operators.MapSlice(context.TODO(), nn, func(ctx context.Context, n int) (string, error) {
		return fmt.Sprintf("%d", n), nil
	})
	if ssErr != nil {
		t.Errorf("%+v", ssErr)
		return
	}
	t.Log(ss)
}
