package tracings

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/logs"
)

type ReporterOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type Reporter interface {
	Construct(options ReporterOptions) (err error)
	Report(ctx context.Context, trace Trace)
}
