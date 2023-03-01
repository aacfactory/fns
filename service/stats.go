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
	"github.com/aacfactory/errors"
	"time"
)

func tryReportStats(ctx context.Context, service string, fn string, err errors.CodeError, span *Span) {
	ts, hasService := GetEndpoint(ctx, "stats")
	if !hasService {
		return
	}
	ec := 0
	en := ""
	if err != nil {
		ec = err.Code()
		en = err.Name()
	}
	latency := time.Duration(0)
	if span != nil {
		latency = span.Latency()
	}
	TryFork(ctx, &reportStatsTask{
		s: &Metric{
			Service:   service,
			Fn:        fn,
			Succeed:   err != nil,
			ErrorCode: ec,
			ErrorName: en,
			Latency:   latency,
		},
		endpoint: ts,
	})
}

type reportStatsTask struct {
	s        *Metric
	endpoint Endpoint
}

func (task *reportStatsTask) Name() (name string) {
	name = "stats-reporter"
	return
}

func (task *reportStatsTask) Execute(ctx context.Context) {
	_ = task.endpoint.Request(ctx, NewRequest(ctx, "stats", "report", NewArgument(task.s)))
}

type Metric struct {
	Service   string        `json:"service"`
	Fn        string        `json:"fn"`
	Succeed   bool          `json:"succeed"`
	ErrorCode int           `json:"errorCode"`
	ErrorName string        `json:"errorName"`
	Latency   time.Duration `json:"latency"`
}
