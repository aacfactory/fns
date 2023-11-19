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
	Endpoint  string `json:"endpoint"`
	Fn        string `json:"fn"`
	Latency   int64  `json:"latency"`
	Succeed   bool   `json:"succeed"`
	ErrorCode int    `json:"errorCode"`
	ErrorName string `json:"errorName"`
	DeviceId  string `json:"deviceId"`
	DeviceIp  string `json:"deviceIp"`
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
		err := errors.Map(cause)
		metric.ErrorCode = err.Code()
		metric.ErrorName = err.Name()
	}
	report(ctx, metric)
}
