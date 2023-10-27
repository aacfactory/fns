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
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/futures"
	"golang.org/x/sync/singleflight"
)

type fnTask struct {
	group    *singleflight.Group
	endpoint Endpoint
	request  Request
	promise  futures.Promise
}

func (f fnTask) Execute(ctx context.Context) {
	req := f.request
	v, err, shared := f.group.Do(bytex.ToString(append(req.Hash(), req.Header().DeviceId()...)), func() (interface{}, error) {
		service, ok := f.endpoint.(Service)
		if ok {
			components := service.Components()
			if len(components) > 0 {
				ctx = WithComponents(ctx, components)
			}
		}
		_, fn := f.request.Fn()
		return f.endpoint.Handle(ctx, fn, f.request.Argument())
	})
	if err == nil {
		f.promise.Succeed(v)
	} else {
		f.promise.Failed(err)
	}

	// todo tracer
	// add shared in span tag
	// todo stats
	// add shared in stats

}
