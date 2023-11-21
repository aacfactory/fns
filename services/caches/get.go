package caches

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/json"
)

func Get(ctx context.Context, param interface{}) (p []byte, has bool, err error) {
	if param == nil {
		err = errors.Warning("fns: get cache failed").WithCause(fmt.Errorf("param is nil"))
		return
	}
	kp, ok := param.(KeyParam)
	if !ok {
		err = errors.Warning("fns: get cache failed").WithCause(fmt.Errorf("param dose not implement caches.KeyParam"))
		return
	}
	key, keyErr := kp.CacheKey(ctx)
	if keyErr != nil {
		err = errors.Warning("fns: get cache failed").WithCause(keyErr)
		return
	}
	eps := runtime.Endpoints(ctx)
	response, doErr := eps.Request(ctx, endpointName, getFnName, getFnParam{
		Key: bytex.ToString(key),
	}, services.WithInternalRequest())
	if doErr != nil {
		err = doErr
		return
	}
	result := getResult{}
	scanErr := response.Scan(&result)
	if scanErr != nil {
		err = errors.Warning("fns: get cache failed").WithCause(scanErr)
		return
	}
	p = result.Value
	has = result.Has
	return
}

type getFnParam struct {
	Key string `json:"key"`
}

type getResult struct {
	Has   bool            `json:"has"`
	Value json.RawMessage `json:"value"`
}

type getFn struct {
	store Store
}

func (fn *getFn) Name() string {
	return string(getFnName)
}

func (fn *getFn) Internal() bool {
	return true
}

func (fn *getFn) Readonly() bool {
	return false
}

func (fn *getFn) Handle(r services.Request) (v interface{}, err error) {
	if !r.Param().Exist() {
		err = errors.Warning("fns: get cache failed").WithCause(errors.Warning("param is invalid"))
		return
	}
	param := getFnParam{}
	paramErr := r.Param().Scan(&param)
	if paramErr != nil {
		err = errors.Warning("fns: get cache failed").WithCause(paramErr)
		return
	}
	key := bytex.FromString(param.Key)
	if len(key) == 0 {
		err = errors.Warning("fns: get cache failed").WithCause(errors.Warning("param is invalid"))
		return
	}
	value, has, getErr := fn.store.Get(r, key)
	if getErr != nil {
		err = errors.Warning("fns: get cache failed").WithCause(getErr)
		return
	}
	v = getResult{
		Has:   has,
		Value: value,
	}
	return
}
