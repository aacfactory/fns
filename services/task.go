package services

import (
	sc "context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/futures"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services/tracings"
)

type FnTask struct {
	Fn      Fn
	Promise futures.Promise
}

func (task FnTask) Execute(ctx sc.Context) {
	req := LoadRequest(context.Wrap(ctx))
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
