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

import "context"

func Foreach[T any](ctx context.Context, array []T, fn func(context.Context, int, T) error) (err error) {
	if array == nil || len(array) == 0 {
		return
	}
	for i, e := range array {
		err = fn(ctx, i, e)
		if err != nil {
			return
		}
	}
	return
}
