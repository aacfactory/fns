package services

import (
	sc "context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
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

func WithComponents(ctx sc.Context, components Components) sc.Context {
	return context.WithValue(ctx, bytex.FromString(contextComponentsKey), components)
}

func LoadComponents(ctx sc.Context) Components {
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

func LoadLoadComponent(ctx sc.Context, name string) (Component, bool) {
	v := LoadComponents(ctx)
	if len(v) == 0 {
		return nil, false
	}
	return v.Get(name)
}
