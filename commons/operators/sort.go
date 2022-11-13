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

package operators

import (
	"context"
	"sort"
)

func Sort[T any](ctx context.Context, array []T, less func(context.Context, T, T) (bool, error)) (rr []T, err error) {
	if array == nil || len(array) == 0 {
		rr = array
		return
	}
	sort.Slice(array, func(i, j int) bool {
		n, cErr := less(ctx, array[i], array[j])
		if cErr != nil {
			err = cErr
			return false
		}
		return n
	})
	rr = array
	return
}
