package fns

import (
	"sync"
)

var _once = new(sync.Once)

func init()  {
	_once.Do(func() {
		initJsonApi()
	})
}
