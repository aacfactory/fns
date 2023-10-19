package switchs

import "sync/atomic"

type Switch struct {
	value int64
}

func (s *Switch) On() {
	atomic.StoreInt64(&s.value, 1)
}

func (s *Switch) Off() {
	atomic.StoreInt64(&s.value, 0)
}

func (s *Switch) IsOn() (ok bool) {
	ok = atomic.LoadInt64(&s.value) == 1
	return
}

func (s *Switch) IsOff() (ok bool) {
	ok = atomic.LoadInt64(&s.value) == 0
	return
}
