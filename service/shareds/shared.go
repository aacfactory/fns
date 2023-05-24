package shareds

import "github.com/aacfactory/systems/memory"

type Shared interface {
	Lockers() (lockers Lockers)
	Store() (store Store)
	Caches() (cache Caches)
}

type Options struct {
	Scope string
}

type Option func(options *Options) (err error)

func WithScope(scope string) Option {
	return func(options *Options) (err error) {
		if scope == "" {
			scope = userScope
		}
		options.Scope = scope
		return
	}
}

const (
	systemScope = "fns/system"
	userScope   = "fns/user"
)

func SystemScope() Option {
	return WithScope(systemScope)
}

func NewOptions(opts []Option) (v *Options, err error) {
	v = &Options{
		Scope: userScope,
	}
	if opts == nil || len(opts) == 0 {
		return
	}
	for _, opt := range opts {
		err = opt(v)
		if err != nil {
			return
		}
	}
	return
}

func Local() (Shared, error) {
	maxCacheSize := uint64(0)
	mem, _ := memory.Stats()
	if mem != nil {
		maxCacheSize = mem.Available / 4
	}
	return &localShared{
		lockers: LocalLockers(),
		store:   LocalStore(),
		cache:   LocalCaches(maxCacheSize),
	}, nil
}

type localShared struct {
	lockers Lockers
	store   Store
	cache   Caches
}

func (s localShared) Lockers() (lockers Lockers) {
	return s.lockers
}

func (s localShared) Store() (store Store) {
	return s.store
}

func (s localShared) Caches() (cache Caches) {
	return s.cache
}
