package metrics

import (
	sc "context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"time"
)

const (
	ServiceName  = "metrics"
	ReportFnName = "report"
)

const (
	contextKey = "@fns:context:metrics"
)

type Metric struct {
	Service   []byte        `json:"service"`
	Fn        []byte        `json:"fn"`
	Latency   time.Duration `json:"latency"`
	Succeed   bool          `json:"succeed"`
	ErrorCode int           `json:"errorCode"`
	ErrorName string        `json:"errorName"`
	DeviceId  []byte        `json:"deviceId"`
	DeviceIp  []byte        `json:"deviceIp"`
	Shared    bool          `json:"shared"`
	Remoted   bool          `json:"remoted"`
	beg       time.Time
}

func Begin(ctx sc.Context, service []byte, fn []byte, deviceId []byte, deviceIp []byte, remoted bool) sc.Context {
	metric := Metric{
		Service:   service,
		Fn:        fn,
		Latency:   0,
		Succeed:   false,
		ErrorCode: 0,
		ErrorName: "",
		DeviceId:  deviceId,
		DeviceIp:  deviceIp,
		Shared:    false,
		Remoted:   remoted,
		beg:       time.Now(),
	}
	return context.WithValue(ctx, bytex.FromString(contextKey), &metric)
}

func Finish(ctx sc.Context, succeed bool, err error, shared bool) (Metric, bool) {
	v := ctx.Value(contextKey)
	if v == nil {
		return Metric{}, false
	}
	m, ok := v.(*Metric)
	if !ok {
		return Metric{}, false
	}
	m.Latency = time.Now().Sub(m.beg)
	m.Succeed = succeed
	m.Shared = shared
	if !succeed {
		if err == nil {
			m.ErrorCode = 500
			m.ErrorName = "unknown"
		} else {
			ec := errors.Map(err)
			m.ErrorCode = ec.Code()
			m.ErrorName = ec.Name()
		}
	}
	return *m, true
}
