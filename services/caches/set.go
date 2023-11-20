package caches

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/json"
	"time"
)

func Set(ctx context.Context, param KeyParam, value interface{}, ttl time.Duration) (err error) {
	key, keyErr := param.CacheKey(ctx)
	if keyErr != nil {
		err = errors.Warning("fns: set cache failed").WithCause(keyErr)
		return
	}
	if value == nil {
		err = errors.Warning("fns: set cache failed").WithCause(fmt.Errorf("value is invalid"))
		return
	}
	p, encodeErr := json.Marshal(value)
	if encodeErr != nil {
		err = errors.Warning("fns: set cache failed").WithCause(encodeErr)
		return
	}
	if ttl < 1 {
		err = errors.Warning("fns: set cache failed").WithCause(fmt.Errorf("ttl is invalid"))
		return
	}
	eps := runtime.Endpoints(ctx)
	_, doErr := eps.Request(ctx, endpointName, setFnName, setFnParam{
		Key:   bytex.ToString(key),
		Value: p,
		TTL:   ttl,
	}, services.WithInternalRequest())
	if doErr != nil {
		err = doErr
		return
	}
	return
}

type setFnParam struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
	TTL   time.Duration   `json:"ttl"`
}

type setFn struct {
	store Store
}

func (fn *setFn) Name() string {
	return string(setFnName)
}

func (fn *setFn) Internal() bool {
	return true
}

func (fn *setFn) Readonly() bool {
	return false
}

func (fn *setFn) Handle(r services.Request) (v interface{}, err error) {
	if !r.Param().Exist() {
		err = errors.Warning("fns: set cache failed").WithCause(errors.Warning("param is invalid"))
		return
	}
	param := setFnParam{}
	paramErr := r.Param().Scan(&param)
	if paramErr != nil {
		err = errors.Warning("fns: set cache failed").WithCause(paramErr)
		return
	}
	key := bytex.FromString(param.Key)
	if len(key) == 0 {
		err = errors.Warning("fns: set cache failed").WithCause(errors.Warning("param is invalid"))
		return
	}
	value := param.Value
	if len(value) == 0 {
		err = errors.Warning("fns: set cache failed").WithCause(errors.Warning("param is invalid"))
		return
	}
	ttl := param.TTL
	if ttl < 1 {
		err = errors.Warning("fns: set cache failed").WithCause(errors.Warning("param is invalid"))
		return
	}
	setErr := fn.store.Set(r, key, value, ttl)
	if setErr != nil {
		err = errors.Warning("fns: set cache failed").WithCause(setErr)
		return
	}
	return
}