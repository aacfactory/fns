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
	"github.com/aacfactory/workers"
)

func TryFork(ctx context.Context, task workers.Task) (ok bool) {
	if task == nil {
		return
	}
	rt := GetRuntime(ctx)
	nc := context.WithValue(context.TODO(), contextRuntimeKey, rt)
	log := rt.log
	taskName := "-"
	named, hasName := task.(workers.NamedTask)
	if hasName {
		taskName = named.Name()
	}
	if taskName == "" {
		taskName = "-"
	}
	log = log.With("task", taskName)
	nc = withLog(nc, log)
	vcm := ctx.Value(contextComponentsKey)
	if vcm != nil {
		nc = withComponents(ctx, vcm.(map[string]Component))
	}
	ok = rt.worker.Dispatch(nc, task)
	return
}

func Fork(ctx context.Context, task workers.Task) {
	if task == nil {
		return
	}
	rt := GetRuntime(ctx)
	nc := context.WithValue(context.TODO(), contextRuntimeKey, rt)
	log := rt.log
	taskName := "-"
	named, hasName := task.(workers.NamedTask)
	if hasName {
		taskName = named.Name()
	}
	if taskName == "" {
		taskName = "-"
	}
	log = log.With("task", taskName)
	nc = withLog(nc, log)
	vcm := ctx.Value(contextComponentsKey)
	if vcm != nil {
		nc = withComponents(ctx, vcm.(map[string]Component))
	}
	rt.worker.MustDispatch(nc, task)
	return
}