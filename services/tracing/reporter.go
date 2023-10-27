package tracing

import "context"

type Reporter interface {
	Report(ctx context.Context, tracer Tracer) (err error)
}

func TryReport() {

}
