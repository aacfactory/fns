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

type MapFn[S any, D any] func(ctx context.Context, src S) (dst D, err error)

func Map[S any, D any](ctx context.Context, src S, fn MapFn[S, D]) (dst D, err error) {
	dst, err = fn(ctx, src)
	return
}

func MapSlice[S any, D any](ctx context.Context, src []S, fn MapFn[S, D]) (dst []D, err error) {
	dst = make([]D, 0, 1)
	if src == nil || len(src) == 0 {
		return
	}
	for _, s := range src {
		d, mapErr := fn(ctx, s)
		if mapErr != nil {
			err = mapErr
			return
		}
		dst = append(dst, d)
	}
	return
}
