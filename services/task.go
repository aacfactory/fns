package services

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/futures"
	"github.com/aacfactory/fns/services/tracings"
)

type FnTask struct {
	Fn      Fn
	Promise futures.Promise
}

func (task FnTask) Execute(ctx context.Context) {
	req := LoadRequest(ctx)
	// tracing
	trace, hasTrace := tracings.Load(req)
	if hasTrace {
		trace.Waited()
	}
	r, err := task.Fn.Handle(req)
	if err != nil {
		if hasTrace {
			codeErr := errors.Map(err)
			trace.Finish("succeed", "false", "cause", codeErr.Name())
		}
		task.Promise.Failed(err)
	} else {
		if hasTrace {
			trace.Finish("succeed", "true")
		}
		task.Promise.Succeed(r)
	}
}
