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

package shared

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"sync"
	"time"
)

const (
	lockerContextKey = "@fns_shared_locker"
)

func WithLocker(ctx context.Context, lockers Lockers) context.Context {
	return context.WithValue(ctx, lockerContextKey, lockers)
}

type Lockers interface {
	Get(ctx context.Context, key []byte, timeout time.Duration) (locker sync.Locker, err errors.CodeError)
}

func Locker(ctx context.Context, key []byte, timeout time.Duration) (locker sync.Locker, err errors.CodeError) {
	v := ctx.Value(lockerContextKey)
	if v == nil {
		err = errors.Warning("fns: get shared lockers failed").WithCause(fmt.Errorf("not exist"))
		return
	}
	lockers, ok := v.(Lockers)
	if !ok {
		err = errors.Warning("fns: get shared lockers failed").WithCause(fmt.Errorf("type is not matched"))
		return
	}
	locker, err = lockers.Get(ctx, key, timeout)
	if err != nil {
		err = errors.ServiceError("fns: locker failed").WithCause(err)
		return
	}
	return
}
