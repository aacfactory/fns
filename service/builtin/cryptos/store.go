package cryptos

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"time"
)

type Key struct {
	Id          string    `json:"id"`
	Owner       string    `json:"owner"`
	Kind        string    `json:"kind"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreateAT    time.Time `json:"createAT"`
	PEM         []byte    `json:"pem"`
}

type KeyStore interface {
	service.Component
	Get(ctx context.Context, id string) (key *Key, err errors.CodeError)
	List(ctx context.Context, owner string) (keys []*Key, err errors.CodeError)
	Save(ctx context.Context, key *Key) (err errors.CodeError)
	Remove(ctx context.Context, id string) (err errors.CodeError)
}
