package services

import (
	sc "context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
)

var (
	contextComponentsKey = []byte("@fns:service:components")
)

type Component interface {
	Name() (name string)
	Construct(options Options) (err error)
	Shutdown(ctx sc.Context)
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

func WithComponents(ctx context.Context, components Components) {
	ctx.SetLocalValue(contextComponentsKey, components)
}

func LoadComponents(ctx context.Context) Components {
	v := ctx.LocalValue(contextComponentsKey)
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

func LoadComponent(ctx context.Context, name string) (Component, bool) {
	v := LoadComponents(ctx)
	if len(v) == 0 {
		return nil, false
	}
	return v.Get(name)
}
