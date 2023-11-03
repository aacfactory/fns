package handlers

import (
	"bytes"
	"context"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/transports"
	"time"
)

var (
	healthPath = bytex.FromString("/application/health")
)

func CheckHealth(ctx context.Context, client transports.Client) (ok bool) {
	status, _, _, _ := client.Do(ctx, methodGet, healthPath, nil, nil)
	ok = status == 200
	return
}

func NewHealthHandler() transports.MuxHandler {
	return &HealthHandler{
		launch: time.Now(),
	}
}

type HealthHandler struct {
	launch time.Time
}

func (handler *HealthHandler) Name() string {
	return "health"
}

func (handler *HealthHandler) Construct(_ transports.MuxHandlerOptions) error {
	return nil
}

func (handler *HealthHandler) Match(method []byte, path []byte, _ transports.Header) bool {
	ok := bytes.Equal(method, methodGet) && bytes.Equal(path, healthPath)
	return ok
}

func (handler *HealthHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	rt := runtime.Load(r)
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
