package metrics

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/services"
)

var (
	endpointName = []byte("metrics")
	reportFnName = []byte("report")
)

func Service(reporter Reporter) services.Service {
	return &service{
		Abstract: services.NewAbstract(string(endpointName), true, reporter),
	}
}

type service struct {
	services.Abstract
}

func (svc *service) Construct(options services.Options) (err error) {
	err = svc.Abstract.Construct(options)
	if err != nil {
		return
	}
	var reporter Reporter
	has := false
	components := svc.Abstract.Components()
	for _, component := range components {
		reporter, has = component.(Reporter)
		if has {
			break
		}
	}
	if reporter == nil {
		err = errors.Warning("metrics: service need reporter component")
		return
	}
	svc.AddFunction(&reportFn{
		reporter: reporter,
	})
	return
}

type reportFn struct {
	reporter Reporter
}

func (fn *reportFn) Name() string {
	return string(reportFnName)
}

func (fn *reportFn) Internal() bool {
	return true
}

func (fn *reportFn) Readonly() bool {
	return false
}

func (fn *reportFn) Handle(r services.Request) (v interface{}, err error) {
	if !r.Param().Exist() {
		return
	}
	metric := Metric{}
	paramErr := r.Param().Scan(&metric)
	if paramErr != nil {
		return
	}
	fn.reporter.Report(r, metric)
	return
}
