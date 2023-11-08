package shareds

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
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
	Store json.RawMessage `json:"store,omitempty" yaml:"store,omitempty"`
}

func Local(log logs.Logger, config LocalSharedConfig) (v Shared, err error) {
	if len(config.Store) == 0 {
		config.Store = []byte{'{', '}'}
	}
	storeConfig, storeConfigErr := configures.NewJsonConfig(config.Store)
	if storeConfigErr != nil {
		err = errors.Warning("fns: build local shared failed").WithCause(storeConfigErr)
		return
	}
	store, storeErr := localStoreBuilder(log, storeConfig)
	if storeErr != nil {
		err = errors.Warning("fns: build local shared failed").WithCause(storeErr)
		return
	}
	v = &localShared{
		lockers: LocalLockers(),
		store:   store,
	}
	return
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
