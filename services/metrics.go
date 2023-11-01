package services

import (
	sc "context"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/services/metrics"
)

type metricReportTask struct {
	endpoint Endpoint
	metric   metrics.Metric
}

func (task metricReportTask) Execute(ctx sc.Context) {
	req := AcquireRequest(
		ctx,
		bytex.FromString(metrics.EndpointName), bytex.FromString(metrics.ReportFnName),
		NewArgument(task.metric),
		WithInternalRequest(),
	)
	_, _ = task.endpoint.Handle(req)
	ReleaseRequest(req)
}
