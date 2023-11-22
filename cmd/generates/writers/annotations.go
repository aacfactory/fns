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

package writers

import (
	"context"
	"github.com/aacfactory/gcg"
)

type FnAnnotationCodeWriter interface {
	Annotation() (annotation string)
	HandleBefore(ctx context.Context, params []string, hasFnParam bool, hasFnResult bool) (code gcg.Code, err error)
	HandleAfter(ctx context.Context, params []string, hasFnParam bool, hasFnResult bool) (code gcg.Code, err error)
	ProxyBefore(ctx context.Context, params []string, hasFnParam bool, hasFnResult bool) (code gcg.Code, err error)
	ProxyAfter(ctx context.Context, params []string, hasFnParam bool, hasFnResult bool) (code gcg.Code, err error)
}

type FnAnnotationCodeWriters []FnAnnotationCodeWriter

func (writers FnAnnotationCodeWriters) Get(annotation string) (w FnAnnotationCodeWriter, has bool) {
	for _, writer := range writers {
		if writer.Annotation() == annotation {
			w = writer
			has = true
			return
		}
	}
	return
}

const (
	fnAnnotationCodeWritersContextKey = "@fns:generates:writers:annotations"
)

func WithFnAnnotationCodeWriters(ctx context.Context, writers FnAnnotationCodeWriters) context.Context {
	return context.WithValue(ctx, fnAnnotationCodeWritersContextKey, writers)
}

func LoadFnAnnotationCodeWriters(ctx context.Context) (w FnAnnotationCodeWriters) {
	v := ctx.Value(fnAnnotationCodeWritersContextKey)
	if v == nil {
		w = make(FnAnnotationCodeWriters, 0, 1)
		return
	}
	ok := false
	w, ok = v.(FnAnnotationCodeWriters)
	if !ok {
		w = make(FnAnnotationCodeWriters, 0, 1)
	}
	return
}
