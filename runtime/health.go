package runtime

import (
	"bytes"
	"context"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"time"
)

var (
	healthPath = bytex.FromString("/health")
)

func CheckHealth(ctx context.Context, client transports.Client) (ok bool) {
	status, _, _, _ := client.Do(ctx, transports.MethodGet, healthPath, nil, nil)
	ok = status == 200
	return
}

func HealthHandler() transports.MuxHandler {
	return &healthHandler{
		launch: time.Now(),
	}
}

type healthHandler struct {
	launch time.Time
}

func (handler *healthHandler) Name() string {
	return "health"
}

func (handler *healthHandler) Construct(_ transports.MuxHandlerOptions) error {
	return nil
}

func (handler *healthHandler) Match(method []byte, path []byte, _ transports.Header) bool {
	ok := bytes.Equal(method, transports.MethodGet) && bytes.Equal(path, healthPath)
	return ok
}

func (handler *healthHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	rt := Load(r)
	running, _ := rt.Running()
	w.Succeed(Health{
		Id:      rt.AppId(),
		Name:    rt.AppName(),
		Version: rt.AppVersion().String(),
		Running: running,
		Launch:  handler.launch,
		Now:     time.Now(),
	})
	return
}

type Health struct {
	Id      string    `json:"id"`
	Name    string    `json:"name"`
	Version string    `json:"version"`
	Running bool      `json:"running"`
	Launch  time.Time `json:"launch"`
	Now     time.Time `json:"now"`
}
