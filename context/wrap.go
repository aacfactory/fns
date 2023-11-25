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

package context

import (
	"context"
	"time"
)

var (
	Canceled         = context.Canceled
	DeadlineExceeded = context.DeadlineExceeded
)

func Wrap(ctx context.Context) Context {
	v, ok := ctx.(Context)
	if ok {
		return v
	}
	return &context_{
		Context: ctx,
		users:   make(Entries, 0, 1),
		locals:  make(Entries, 0, 1),
	}
}

func TODO() Context {
	return Wrap(context.TODO())
}

func WithValue(parent context.Context, key any, val any) Context {
	return &valueContext{
		Context: Wrap(parent),
		key:     key,
		val:     val,
	}
}

type CancelFunc context.CancelFunc

func WithCancel(parent Context) (Context, CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithTimeout(parent Context, ttl time.Duration) (Context, CancelFunc) {
	ctx, cancel := context.WithTimeout(parent, ttl)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithTimeoutCause(parent Context, ttl time.Duration, cause error) (Context, CancelFunc) {
	ctx, cancel := context.WithTimeoutCause(parent, ttl, cause)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithDeadline(parent Context, deadline time.Time) (Context, CancelFunc) {
	ctx, cancel := context.WithDeadline(parent, deadline)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithDeadlineCause(parent Context, deadline time.Time, cause error) (Context, CancelFunc) {
	ctx, cancel := context.WithDeadlineCause(parent, deadline, cause)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithoutCancel(parent Context) Context {
	return Wrap(context.WithoutCancel(parent))
}

func AfterFunc(ctx Context, f func()) (stop func() bool) {
	stop = context.AfterFunc(ctx, f)
	return
}
