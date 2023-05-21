package shareds

import "github.com/aacfactory/systems/memory"

type Builder func() (Shared, error)

type Shared interface {
	Lockers() (lockers Lockers)
	Store() (store Store)
	Caches() (cache Caches)
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
