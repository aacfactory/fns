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

package arrays

import "sort"

type Array[E any] []E

func Map[S any, R any](src Array[S], fn func(S) (R, error)) (dst Array[R], err error) {
	dst = make([]R, 0, 1)
	if src == nil || len(src) == 0 {
		return
	}
	for _, s := range src {
		d, mapErr := fn(s)
		if mapErr != nil {
			err = mapErr
			return
		}
		dst = append(dst, d)
	}
	return
}

func Foreach[T any](tt Array[T], fn func(int, T) error) (err error) {
	if tt == nil || len(tt) == 0 {
		return
	}
	for i, t := range tt {
		err = fn(i, t)
		if err != nil {
			return
		}
	}
	return
}

func Find[T any](tt Array[T], fn func(T) (bool, error)) (r T, found bool, err error) {
	if tt == nil || len(tt) == 0 {
		return
	}
	for _, t := range tt {
		found, err = fn(t)
		if err != nil {
			return
		}
		if found {
			r = t
			return
		}
	}
	return
}

func Filter[T any](tt Array[T], fn func(T) (bool, error)) (rr Array[T], err error) {
	rr = make([]T, 0, 1)
	if tt == nil || len(tt) == 0 {
		return
	}
	for _, t := range tt {
		found, findErr := fn(t)
		if findErr != nil {
			err = findErr
			return
		}
		if found {
			rr = append(rr, t)
		}
	}
	return
}

func Sort[T any](tt Array[T], fn func(T, T) (int, error)) (rr Array[T], err error) {
	if tt == nil || len(tt) == 0 {
		rr = tt
		return
	}
	sort.Slice(tt, func(i, j int) bool {
		n, cErr := fn(tt[i], tt[j])
		if cErr != nil {
			err = cErr
			return false
		}
		return n < 0
	})
	rr = tt
	return
}
