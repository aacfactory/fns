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

package operators

import "github.com/aacfactory/fns/context"

type FindFn[T any] func(ctx context.Context, element T) (found bool, err error)

func Find[T any](ctx context.Context, elements []T, fn FindFn[T]) (target T, found bool, err error) {
	if elements == nil || len(elements) == 0 {
		return
	}
	for _, e := range elements {
		found, err = fn(ctx, e)
		if err != nil {
			return
		}
		if found {
			target = e
			return
		}
	}
	return
}
