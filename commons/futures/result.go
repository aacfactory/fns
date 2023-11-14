package futures

import (
	"fmt"
	"github.com/aacfactory/json"
	"reflect"
)

type Result interface {
	json.Marshaler
	Exist() (ok bool)
	Scan(dst interface{}) (err error)
}

type result struct {
	value interface{}
}

func (r result) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.value)
}

func (r result) Exist() (ok bool) {
	if r.value == nil {
		return
	}
	rr, matched := r.value.(Result)
	if matched {
		ok = rr.Exist()
		return
	}
	ok = true
	return
}

func (r result) Scan(dst interface{}) (err error) {
	if dst == nil {
		return
	}
	if !r.Exist() {
		return
	}
	rr, matched := r.value.(Result)
	if matched {
		err = rr.Scan(dst)
		return
	}
	dpv := reflect.ValueOf(dst)
	if dpv.Kind() != reflect.Ptr {
		err = fmt.Errorf("copy failed for type of dst is not ptr")
		return
	}
	sv := reflect.ValueOf(r.value)
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
	err = fmt.Errorf("scan failed for type is not matched")
	return
}
