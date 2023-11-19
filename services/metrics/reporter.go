package metrics

import (
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"sync"
	"time"
)

type Reporter interface {
	services.Component
	Report(ctx context.Context, metric Metric)
}

type Event struct {
	rt     *runtime.Runtime
	metric Metric
}

var (
	events = make(chan Event, 4096)
	timers = sync.Pool{New: func() any {
		return time.NewTimer(10 * time.Microsecond)
	}}
)

func listen() {
	for {
		event, ok := <-events
		if !ok {
			break
		}
		rt := event.rt
		eps := rt.Endpoints()
		ctx := runtime.With(context.TODO(), event.rt)
		_, _ = eps.Request(ctx, endpointName, reportFnName, event.metric)
	}
}

func report(ctx context.Context, metric Metric) {
	rt := runtime.Load(ctx)
	if rt == nil {
		return
	}
	timer := timers.Get().(*time.Timer)
	select {
	case <-timer.C:
		break
	case events <- Event{
		rt:     rt,
		metric: metric,
	}:
		break
	}
	timer.Reset(10 * time.Microsecond)
	timers.Put(timer)
}
