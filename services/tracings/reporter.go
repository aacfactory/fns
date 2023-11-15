package tracings

import "context"

type Reporter interface {
	Report(ctx context.Context, trace Trace)
}
