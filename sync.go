package fns

import "sync"

func NewSyncBool() *SyncBool {
	return &SyncBool{
		lock: sync.RWMutex{},
		v:    false,
	}
}

type SyncBool struct {
	lock sync.RWMutex
	v    bool
}

func (s *SyncBool) Set(v bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.v = v
}

func (s *SyncBool) Get() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.v
}
