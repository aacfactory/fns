package metrics

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"time"
)

const (
	contextKey = "@fns:context:metrics"
)

// Metric
// todo
// as service
// tryExecute in service
// report channel listen in service
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
	Shared    bool   `json:"shared"`
	beg       time.Time
}

func (m *Metric) Finish() {
	m.Succeed = true
	m.Latency = time.Now().Sub(m.beg).Milliseconds()
}

func Begin(ctx context.Context, endpoint []byte, fn []byte, deviceId []byte, deviceIp []byte, remoted bool) Metric {
	metric := Metric{
		Endpoint:  bytex.ToString(endpoint),
		Fn:        bytex.ToString(fn),
		Latency:   0,
		Succeed:   false,
		ErrorCode: 0,
		ErrorName: "",
		DeviceId:  string(deviceId),
		DeviceIp:  string(deviceIp),
		Shared:    false,
		beg:       time.Now(),
	}
	return metric
}
