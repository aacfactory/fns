package scanner

import (
	stdjson "encoding/json"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"reflect"
)

type Scanner interface {
	Exist() (ok bool)
	Scan(dst interface{}) (err error)
	json.Marshaler
}

func New(src interface{}) Scanner {
	if src == nil {
		return object{}
	}
	s, ok := src.(Scanner)
	if ok {
		return s
	}
	return object{
		value: src,
	}
}

type object struct {
	value interface{}
}

func (obj object) Exist() (ok bool) {
	if obj.value == nil {
		return
	}
	pp, isParam := obj.value.(Scanner)
	if isParam {
		ok = pp.Exist()
		return
	}
	ok = true
	return
}

func (obj object) Scan(dst interface{}) (err error) {
	if dst == nil {
		err = errors.Warning("fns: scan object failed").WithCause(fmt.Errorf("dst is nil"))
		return
	}
	if !obj.Exist() {
		return
	}
	pp, isParam := obj.value.(Scanner)
	if isParam {
		err = pp.Scan(dst)
		if err != nil {
			err = errors.Warning("fns: scan object failed").WithCause(err)
			return
		}
		return
	}
	dpv := reflect.ValueOf(dst)
	if dpv.Kind() != reflect.Ptr {
		err = errors.Warning("fns: scan object failed").WithCause(fmt.Errorf("type of dst is not pointer"))
		return
	}
	sv := reflect.ValueOf(obj.value)
	dv := reflect.Indirect(dpv)
	if sv.Kind() == reflect.Ptr {
		if sv.IsNil() {
			return
		}
		sv = sv.Elem()
	}
	if sv.IsValid() && sv.Type().AssignableTo(dv.Type()) {
		dv.Set(sv)
		return
	}
	if dv.Kind() == sv.Kind() && sv.Type().ConvertibleTo(dv.Type()) {
		dv.Set(sv.Convert(dv.Type()))
		return
	}
	err = errors.Warning("fns: scan object failed").WithCause(fmt.Errorf("type of dst is not matched"))
	return
}

func (obj object) MarshalJSON() (data []byte, err error) {
	if !obj.Exist() {
		data = json.NullBytes
		return
	}
	switch obj.value.(type) {
	case struct{}, *struct{}:
		data = json.NullBytes
		break
	case []byte:
		value := obj.value.([]byte)
		if !json.Validate(value) {
			data, err = json.Marshal(obj.value)
			return
		}
		data = value
		break
	case json.RawMessage:
		data = obj.value.(json.RawMessage)
		break
	case stdjson.RawMessage:
		data = obj.value.(stdjson.RawMessage)
		break
	default:
		data, err = json.Marshal(obj.value)
		if err != nil {
			err = errors.Warning("fns: encode scanner object to json bytes failed").WithCause(err)
			return
		}
	}
	return
}
