package services

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
)

const (
	contextComponentsKey = "@fns:service:components"
)

type Component interface {
	Name() (name string)
	Construct(options Options) (err error)
	Close()
}

type Components []Component

func (components Components) Get(key string) (v Component, has bool) {
	for _, component := range components {
		if component.Name() == key {
			v = component
			has = true
			return
		}
	}
	return
}

func WithComponents(ctx context.Context, components Components) context.Context {
	ctx = context.WithValue(ctx, contextComponentsKey, components)
	return ctx
}

func LoadComponents(ctx context.Context) Components {
	v := ctx.Value(contextComponentsKey)
	if v == nil {
		return nil
	}
	c, ok := v.(Components)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: components in context is not github.com/aacfactory/fns/services.Components")))
		return nil
	}
	return c
}
