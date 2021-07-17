package fns

import "time"

const (
	B = 1 << (10 * iota)
	KB
	MB
	GB
	TB
	PB
	EB
)

type Application interface {
	Deploy(service ...Service)
	Run() (err error)
	Sync() (err error)
	SyncWithTimeout(timeout time.Duration) (err error)
}
