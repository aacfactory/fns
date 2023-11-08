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

type LocalSharedConfig struct {
	Store LocalSharedStoreConfig `json:"store,omitempty" yaml:"store,omitempty"`
}

func Local(config LocalSharedConfig) Shared {
	return &localShared{
		lockers: LocalLockers(),
		store:   LocalStore(config.Store),
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
