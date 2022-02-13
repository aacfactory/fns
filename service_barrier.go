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

package fns

import (
	"golang.org/x/sync/singleflight"
)

var serviceBarrierRetrievers = map[string]ServiceBarrierRetriever{
	"local": localServiceBarrierRetriever,
}

func RegisterServiceBarrierRetriever(name string, retriever ServiceBarrierRetriever) {
	serviceBarrierRetrievers[name] = retriever
}

type ServiceBarrierRetriever func() (b ServiceBarrier)

func localServiceBarrierRetriever() (b ServiceBarrier) {
	b = newLocalServiceBarrier()
	return
}

type ServiceBarrier interface {
	Do(key string, fn func() (v interface{}, err error)) (v interface{}, err error, shared bool)
	Forget(key string)
}

func newLocalServiceBarrier() ServiceBarrier {
	return &localServiceBarrier{
		v: &singleflight.Group{},
	}
}

type localServiceBarrier struct {
	v *singleflight.Group
}

func (b *localServiceBarrier) Do(key string, fn func() (v interface{}, err error)) (v interface{}, err error, shared bool) {
	v, err, shared = b.v.Do(key, fn)
	return
}

func (b *localServiceBarrier) Forget(key string) {
	b.v.Forget(key)
}
