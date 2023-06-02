package cryptos

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
)

const (
	name     = "cryptos"
	createFn = "create"
	uploadFn = "upload"
	getFn    = "get"
	listFN   = "list"
	removeFn = "remove"
)

func Service(store KeyStore) service.Service {
	return &_service{
		Abstract: service.NewAbstract(name, true, store),
		store:    store,
	}
}

type _service struct {
	service.Abstract
	store KeyStore
}

func (svc _service) Handle(ctx context.Context, fn string, argument service.Argument) (v interface{}, err errors.CodeError) {

	return
}
