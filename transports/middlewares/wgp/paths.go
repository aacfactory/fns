package wgp

import (
	"bytes"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"net/http"
	"reflect"
	"sync"
)

type Path struct {
	Name      []byte
	ParamType reflect.Type
}

func (path Path) Param() interface{} {
	if path.ParamType == nil {
		return nil
	}
	v := reflect.New(path.ParamType).Elem()
	return v.Interface()
}

func NewPaths() *Paths {
	return &Paths{
		locker: new(sync.Mutex),
		values: make([]Path, 0, 1),
	}
}

type Paths struct {
	locker sync.Locker
	values []Path
}

func (paths *Paths) Add(service string, fn string, param interface{}) {
	paths.locker.Lock()
	defer paths.locker.Unlock()
	path := Path{
		Name: bytex.FromString(fmt.Sprintf("/%s/%s", service, fn)),
	}
	if param != nil {
		path.ParamType = reflect.TypeOf(param)
	}
	paths.values = append(paths.values, path)
}

func (paths *Paths) WrapRequest(r transports.Request) (err error) {
	if !bytes.Equal(r.Method(), bytex.FromString(http.MethodGet)) {
		return
	}
	if len(r.Header().Get(bytex.FromString(transports.UpgradeHeaderName))) > 0 {
		return
	}
	for _, path := range paths.values {
		if bytes.Equal(path.Name, r.Path()) {
			param := path.Param()
			decodeErr := transports.DecodeParams(r.Params(), &param)
			if decodeErr != nil {
				err = errors.Warning("fns: wrap request failed").WithMeta("path", string(r.Path())).WithCause(decodeErr)
				return
			}
			p, encodeErr := json.Marshal(param)
			if encodeErr != nil {
				err = errors.Warning("fns: wrap request failed").WithMeta("path", string(r.Path())).WithCause(encodeErr)
				return
			}
			r.Header().Set(bytex.FromString(transports.ContentTypeHeaderName), bytex.FromString(transports.ContentTypeJsonHeaderValue))
			r.SetMethod(bytex.FromString(http.MethodPost))
			r.SetBody(p)
			break
		}
	}
	return
}

var paths = NewPaths()

func Add(service string, fn string, param interface{}) {
	paths.Add(service, fn, param)
}
