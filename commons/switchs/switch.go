package switchs

import (
	"sync/atomic"
)

type Switch struct {
	value uint64
}

func (s *Switch) On() {
	atomic.StoreUint64(&s.value, 2)
}

func (s *Switch) Off() {
	atomic.StoreUint64(&s.value, 1)
}

func (s *Switch) Confirm() {
	n := atomic.LoadUint64(&s.value)
	switch n {
	case 2:
		atomic.StoreUint64(&s.value, 3)
		break
	case 1:
		atomic.StoreUint64(&s.value, 0)
		break
	default:
		break
	}
}

func (s *Switch) IsOn() (ok bool, confirmed bool) {
	n := atomic.LoadUint64(&s.value)
	switch n {
	case 2:
		ok = true
		break
	case 3:
		ok = true
		confirmed = true
		break
	default:
		break
	}
	return
}

func (s *Switch) IsOff() (ok bool, confirmed bool) {
	n := atomic.LoadUint64(&s.value)
	switch n {
	case 1:
		ok = true
		break
	case 0:
		ok = true
		confirmed = true
		break
	default:
		break
	}
	return
}
