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

package runtime

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/workers"
	"time"
)

var (
	contextKey = []byte("@fns:context:runtime")
)

func With(ctx context.Context, rt *Runtime) context.Context {
	ctx.SetLocalValue(contextKey, rt)
	return ctx
}

func Load(ctx context.Context) *Runtime {
	v := ctx.LocalValue(contextKey)
	if v == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: there is no runtime in context")))
		return nil
	}
	rt, ok := v.(*Runtime)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: runtime in context is not github.com/aacfactory/fns/runtime.Runtime")))
		return nil
	}
	return rt
}

func Endpoints(ctx context.Context) services.Endpoints {
	rt := Load(ctx)
	return rt.Endpoints()
}

func TryExecute(ctx context.Context, task workers.Task) bool {
	rt := Load(ctx)
	return rt.TryExecute(context.TODO(), task)
}

func Execute(ctx context.Context, task workers.Task) {
	rt := Load(ctx)
	rt.Execute(context.TODO(), task)
}

func Barrier(ctx context.Context, key []byte, fn func() (result interface{}, err error)) (result barriers.Result, err error) {
	rt := Load(ctx)
	barrier := rt.Barrier()
	result, err = barrier.Do(ctx, key, fn)
	barrier.Forget(ctx, key)
	return
}

func AcquireLocker(ctx context.Context, key []byte, ttl time.Duration) (locker shareds.Locker, err error) {
	rt := Load(ctx)
	locker, err = rt.Shared().Lockers().Acquire(ctx, key, ttl)
	return
}

func SharedStore(ctx context.Context) (store shareds.Store) {
	rt := Load(ctx)
	store = rt.Shared().Store()
	return
}
