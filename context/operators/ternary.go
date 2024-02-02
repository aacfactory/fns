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

type TernaryFn func(ctx context.Context) (returning bool, err error)

func Ternary(ctx context.Context, condition func(ctx context.Context) (ok bool), truthy TernaryFn, falsy TernaryFn) (returning bool, err error) {
	if condition(ctx) {
		returning, err = truthy(ctx)
	} else {
		returning, err = falsy(ctx)
	}
	return
}

func TernaryTruthy(ctx context.Context, condition func(ctx context.Context) (ok bool), truthy TernaryFn) (returning bool, err error) {
	if condition(ctx) {
		returning, err = truthy(ctx)
	}
	return
}

func TernaryFalsy(ctx context.Context, condition func(ctx context.Context) (ok bool), falsy TernaryFn) (returning bool, err error) {
	if condition(ctx) {
		returning, err = falsy(ctx)
	}
	return
}
