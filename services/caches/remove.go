package caches

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
)

func Remove(ctx context.Context, param interface{}) (err error) {
	if param == nil {
		err = errors.Warning("fns: remove cache failed").WithCause(fmt.Errorf("param is nil"))
		return
	}
	kp, ok := param.(KeyParam)
	if !ok {
		err = errors.Warning("fns: remove cache failed").WithCause(fmt.Errorf("param dose not implement caches.KeyParam"))
		return
	}
	key, keyErr := kp.CacheKey(ctx)
	if keyErr != nil {
		err = errors.Warning("fns: remove cache failed").WithCause(keyErr)
		return
	}
	eps := runtime.Endpoints(ctx)
	_, doErr := eps.Request(ctx, endpointName, remFnName, removeFnParam{
		Key: bytex.ToString(key),
	}, services.WithInternalRequest())
	if doErr != nil {
		err = doErr
		return
	}
	return
}

type removeFnParam struct {
	Key string `json:"key"`
}

type removeFn struct {
	store Store
}

func (fn *removeFn) Name() string {
	return string(remFnName)
}

func (fn *removeFn) Internal() bool {
	return true
}

func (fn *removeFn) Readonly() bool {
	return false
}

func (fn *removeFn) Handle(r services.Request) (v interface{}, err error) {
	if !r.Param().Exist() {
		err = errors.Warning("fns: remove cache failed").WithCause(errors.Warning("param is invalid"))
		return
	}
	param := removeFnParam{}
	paramErr := r.Param().Scan(&param)
	if paramErr != nil {
		err = errors.Warning("fns: remove cache failed").WithCause(paramErr)
		return
	}
	key := bytex.FromString(param.Key)
	if len(key) == 0 {
		err = errors.Warning("fns: remove cache failed").WithCause(errors.Warning("param is invalid"))
		return
	}
	removeErr := fn.store.Remove(r, key)
	if removeErr != nil {
		err = errors.Warning("fns: remove cache failed").WithCause(removeErr)
		return
	}
	return
}
