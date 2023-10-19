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

package services

import (
	"context"
	"github.com/aacfactory/errors"
	"golang.org/x/sync/singleflight"
)

type Barrier interface {
	Do(ctx context.Context, key string, fn func() (result interface{}, err errors.CodeError)) (result interface{}, err errors.CodeError)
	Forget(ctx context.Context, key string)
}

func defaultBarrier() Barrier {
	return &sfgBarrier{
		group: singleflight.Group{},
	}
}

// todo 外层本地，然后判断等级，如果是全局的，则使用shared，shared的key带时间戳
// shared key：1、取store里的时间戳，key为key本身，如果没有则新建，如果有，则拿存在的，用原本的key+时间戳构建新的key，用于存结果，有效期是3秒
// 等级分：本地、本地+device、全局、全局+device
// forget，只处理本地的，shared里的由TTL实现
type sfgBarrier struct {
	group singleflight.Group
}

func (barrier *sfgBarrier) Do(_ context.Context, key string, fn func() (result interface{}, err errors.CodeError)) (result interface{}, err errors.CodeError) {
	var doErr error
	result, doErr, _ = barrier.group.Do(key, func() (interface{}, error) {
		return fn()
	})
	if doErr != nil {
		err = errors.Map(doErr)
	}
	return
}

func (barrier *sfgBarrier) Forget(_ context.Context, key string) {
	barrier.group.Forget(key)
}
