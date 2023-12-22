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

package metrics

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"time"
)

var (
	contextKey = []byte("@fns:context:metrics")
)

// Metric
// use @metric to enable in fn
type Metric struct {
	Endpoint  string `json:"endpoint" avro:"endpoint"`
	Fn        string `json:"fn" avro:"fn"`
	Latency   int64  `json:"latency" avro:"latency"`
	Succeed   bool   `json:"succeed" avro:"succeed"`
	ErrorCode int    `json:"errorCode" avro:"errorCode"`
	ErrorName string `json:"errorName" avro:"errorName"`
	DeviceId  string `json:"deviceId" avro:"deviceId"`
	DeviceIp  string `json:"deviceIp" avro:"deviceIp"`
	beg       time.Time
}

func Begin(ctx context.Context) {
	r, ok := services.TryLoadRequest(ctx)
	if !ok {
		return
	}
	ep, fn := r.Fn()
	metric := Metric{
		Endpoint:  bytex.ToString(ep),
		Fn:        bytex.ToString(fn),
		Latency:   0,
		Succeed:   false,
		ErrorCode: 0,
		ErrorName: "",
		DeviceId:  bytex.ToString(r.Header().DeviceId()),
		DeviceIp:  bytex.ToString(r.Header().DeviceIp()),
		beg:       time.Now(),
	}
	r.SetLocalValue(contextKey, metric)
	return
}

func End(ctx context.Context) {
	EndWithCause(ctx, nil)
}

func EndWithCause(ctx context.Context, cause error) {
	v := ctx.LocalValue(contextKey)
	if v == nil {
		return
	}
	metric, has := v.(Metric)
	if !has {
		return
	}
	r, ok := services.TryLoadRequest(ctx)
	if !ok {
		return
	}
	ep, fn := r.Fn()
	if metric.Endpoint != bytex.ToString(ep) && metric.Fn != bytex.ToString(fn) {
		return
	}
	metric.Latency = time.Now().Sub(metric.beg).Milliseconds()
	if cause == nil {
		metric.Succeed = true
	} else {
		err := errors.Wrap(cause)
		metric.ErrorCode = err.Code()
		metric.ErrorName = err.Name()
	}
	report(ctx, metric)
}
