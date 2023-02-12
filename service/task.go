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

package service

import (
	"context"
	"github.com/aacfactory/workers"
)

type Workers interface {
	Dispatch(ctx context.Context, task workers.Task) (ok bool)
	MustDispatch(ctx context.Context, task workers.Task)
}

type Task interface {
	workers.Task
	Name() (name string)
}

func TryFork(ctx context.Context, task Task) (ok bool) {
	if task == nil {
		return
	}
	rt := getRuntime(ctx)
	nc := context.WithValue(context.TODO(), contextRuntimeKey, rt)
	log := rt.log
	taskName := task.Name()
	if taskName == "" {
		taskName = "-"
	}
	log = log.With("task", taskName)
	nc = SetLog(nc, log)
	vcm := ctx.Value(contextComponentsKey)
	if vcm != nil {
		nc = setComponents(ctx, vcm.(map[string]Component))
	}
	ok = rt.worker.Dispatch(ctx, task)
	return
}

func Fork(ctx context.Context, task Task) {
	if task == nil {
		return
	}
	rt := getRuntime(ctx)
	nc := context.WithValue(context.TODO(), contextRuntimeKey, rt)
	log := rt.log
	taskName := task.Name()
	if taskName == "" {
		taskName = "-"
	}
	log = log.With("task", taskName)
	nc = SetLog(nc, log)
	vcm := ctx.Value(contextComponentsKey)
	if vcm != nil {
		nc = setComponents(ctx, vcm.(map[string]Component))
	}
	rt.worker.MustDispatch(ctx, task)
	return
}
