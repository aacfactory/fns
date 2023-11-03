package shareds

type Shared interface {
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

func (s localShared) Lockers() (lockers Lockers) {
	return s.lockers
}

func (s localShared) Store() (store Store) {
	return s.store
}
