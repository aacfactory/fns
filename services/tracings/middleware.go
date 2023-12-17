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

package tracings

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
)

type Config struct {
	Enable      bool            `json:"enable"`
	BatchSize   int             `json:"batchSize"`
	ChannelSize int             `json:"channelSize"`
	Reporter    json.RawMessage `json:"reporter"`
}

type Middleware struct {
	log      logs.Logger
	enable   bool
	events   chan *Trace
	cancel   context.CancelFunc
	reporter Reporter
}

func (middle *Middleware) Name() string {
	return "tracings"
}

func (middle *Middleware) Construct(options transports.MiddlewareOptions) (err error) {
	middle.log = options.Log
	config := Config{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: tracing middleware construct failed").WithCause(configErr)
		return
	}
	if config.Enable {
		reporterConfig, reporterConfigErr := configures.NewJsonConfig(config.Reporter)
		if configErr != nil {
			err = errors.Warning("fns: tracing middleware construct failed").WithCause(reporterConfigErr)
			return
		}
		reportErr := middle.reporter.Construct(ReporterOptions{
			Log:    middle.log,
			Config: reporterConfig,
		})
		if reportErr != nil {
			err = errors.Warning("fns: tracing middleware construct failed").WithCause(reportErr)
			return
		}
		batchSize := config.BatchSize
		if batchSize < 0 {
			batchSize = 4
		}
		chs := config.ChannelSize
		if chs < 0 {
			chs = 4096
		}
		middle.events = make(chan *Trace, chs)
		ctx, cancel := context.WithCancel(context.TODO())
		middle.cancel = cancel
		for i := 0; i < batchSize; i++ {
			middle.listen(ctx)
		}
	}
	return
}

func (middle *Middleware) Handler(next transports.Handler) transports.Handler {
	if middle.enable {
		return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
			id := r.Header().Get(transports.RequestIdHeaderName)
			if len(id) == 0 {
				next.Handle(w, r)
				return
			}
			tracer := New(id)
			With(r, tracer)
			next.Handle(w, r)
			trace := tracer.Trace()
			middle.events <- trace
		})
	}
	return next
}

func (middle *Middleware) Close() {
	if middle.cancel != nil {
		middle.cancel()
	}
}

func (middle *Middleware) listen(ctx context.Context) {
	go func(ctx context.Context, events chan *Trace, reporter Reporter) {
		stop := false
		for {
			select {
			case <-ctx.Done():
				stop = true
				break
			case trace, ok := <-events:
				if !ok {
					stop = true
					break
				}
				reporter.Report(ctx, trace)
				break
			}
			if stop {
				break
			}
		}
	}(ctx, middle.events, middle.reporter)
}
