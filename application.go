package fns

import "time"

type Application interface {

	Run() (err error)
	Sync() (err error)
	SyncWithTimeout(timeout time.Duration) (err error)
}
