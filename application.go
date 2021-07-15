package fns

import "time"

type Application interface {
	Deploy(service ...Service)
	Run() (err error)
	Sync() (err error)
	SyncWithTimeout(timeout time.Duration) (err error)
}
