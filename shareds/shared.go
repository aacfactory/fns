package shareds

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/logs"
)

type Options struct {
	Log    logs.Logger
	Config configures.Config
}

type Shared interface {
	Construct(options Options) (err error)
	Lockers() (lockers Lockers)
	Store() (store Store)
}

func Local() Shared {
	return &localShared{
		lockers: LocalLockers(),
		store:   LocalStore(),
	}
}

type localShared struct {
	lockers Lockers
	store   Store
}

func (s localShared) Construct(_ Options) (err error) {
	return
}

func (s localShared) Lockers() (lockers Lockers) {
	return s.lockers
}

func (s localShared) Store() (store Store) {
	return s.store
}
