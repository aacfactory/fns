package fns

type Service interface {
	Name() (name string)
	//Index asc sort key
	Index() (idx int)
	Start(context Context, env Environment) (err error)
	Stop(context Context) (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+
