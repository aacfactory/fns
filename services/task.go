package services

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/futures"
	"github.com/aacfactory/fns/services/tracings"
)

type fnTask struct {
	fn      Fn
	promise futures.Promise
}

func (task fnTask) Execute(ctx context.Context) {
	req := LoadRequest(ctx)
	// tracing
	trace, hasTrace := tracings.Load(req)
	if hasTrace {
		trace.Waited()
	}
	r, err := task.fn.Handle(req)
	if err != nil {
		if hasTrace {
			codeErr := errors.Map(err)
			trace.Finish("succeed", "false", "cause", codeErr.Name())
		}
		task.promise.Failed(err)
	} else {
		if hasTrace {
			trace.Finish("succeed", "true")
		}
		task.promise.Succeed(r)
	}
}
