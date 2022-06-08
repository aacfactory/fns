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
	"github.com/aacfactory/fns/service/tracing"
	"time"
)

type fnStats struct {
	Service_  string        `json:"service"`
	Fn_       string        `json:"fn"`
	Succeed_  bool          `json:"succeed"`
	ErrorCode int           `json:"errorCode"`
	ErrorName string        `json:"errorName"`
	Latency_  time.Duration `json:"latency"`
}

func (s *fnStats) Service() (name string) {
	name = s.Service_
	return
}

func (s *fnStats) Fn() (name string) {
	name = s.Fn_
	return
}

func (s *fnStats) Succeed() (ok bool) {
	ok = s.Succeed_
	return
}

func (s *fnStats) Error() (code int, name string) {
	code, name = s.ErrorCode, s.ErrorName
	return
}

func (s *fnStats) Latency() (v time.Duration) {
	v = s.Latency_
	return
}

func tryReportStats(ctx context.Context, service string, fn string, err errors.CodeError, span tracing.Span) {
	ec := 0
	en := ""
	if err != nil {
		ec = err.Code()
		en = err.Name()
	}
	TryFork(ctx, &reportStatsTask{
		s: &fnStats{
			Service_:  service,
			Fn_:       fn,
			Succeed_:  err == nil,
			ErrorCode: ec,
			ErrorName: en,
			Latency_:  span.Latency(),
		},
	})
}

type reportStatsTask struct {
	s *fnStats
}

func (task *reportStatsTask) Name() (name string) {
	name = "stats-reporter"
	return
}

func (task *reportStatsTask) Execute(ctx context.Context) {
	ts, hasService := GetEndpoint(ctx, "stats")
	if !hasService {
		return
	}
	_ = ts.Request(ctx, "report", NewArgument(task.s))
}
