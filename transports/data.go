package transports

type UserValues interface {
	UserValue(key []byte) (val any)
	SetUserValue(key []byte, val any)
	RemoveUserValue(key []byte)
	ForeachUserValues(fn func(key []byte, val any))
}
