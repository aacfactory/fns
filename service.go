package fns

type Service interface {
	Start(context Context, env Environment) (err error)
	Stop(context Context) (err error)
}
