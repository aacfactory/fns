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

type FilterFn[T any] func(ctx context.Context, element T) (ok bool, err error)

func Filter[T any](ctx context.Context, elements []T, fn FilterFn[T]) (targets []T, err error) {
	if elements == nil || len(elements) == 0 {
		return
	}
	for _, e := range elements {
		found, findErr := fn(ctx, e)
		if findErr != nil {
			err = findErr
			return
		}
		if found {
			if targets == nil {
				targets = make([]T, 0, 1)
			}
			targets = append(targets, e)
		}
	}
	return
}
