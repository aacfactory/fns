package transports

import (
	"bytes"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/valyala/bytebufferpool"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Params interface {
	Get(name []byte) []byte
	Set(name []byte, value []byte)
	Remove(name []byte)
	Len() int
	Encode() (p []byte)
}

type param struct {
	key []byte
	val []byte
}

func NewParams() Params {
	pp := make(defaultParams, 0, 1)
	return &pp
}

type defaultParams []param

func (params *defaultParams) Less(i, j int) bool {
	pp := *params
	return bytes.Compare(pp[i].key, pp[j].key) < 0
}

func (params *defaultParams) Swap(i, j int) {
	pp := *params
	pp[i], pp[j] = pp[j], pp[i]
	*params = pp
}

func (params *defaultParams) Get(name []byte) []byte {
	if name == nil {
		return nil
	}
	if len(name) == 0 {
		return nil
	}
	pp := *params
	for _, p := range pp {
		if bytes.Equal(p.key, name) {
			return p.val
		}
	}
	return nil
}

func (params *defaultParams) Set(name []byte, value []byte) {
	if name == nil || value == nil {
		return
	}
	if len(name) == 0 {
		return
	}
	pp := *params
	for _, p := range pp {
		if bytes.Equal(p.key, name) {
			p.val = value
			*params = pp
			return
		}
	}
	pp = append(pp, param{
		key: name,
		val: value,
	})
	*params = pp
}

func (params *defaultParams) Remove(name []byte) {
	if name == nil {
		return
	}
	if len(name) == 0 {
		return
	}
	pp := *params
	n := -1
	for i, p := range pp {
		if bytes.Equal(p.key, name) {
			n = i
			break
		}
	}
	if n == -1 {
		return
	}
	pp = append(pp[:n], pp[n+1:]...)
	*params = pp
}

func (params *defaultParams) Len() int {
	return len(*params)
}

func (params *defaultParams) Encode() []byte {
	if params.Len() == 0 {
		return nil
	}
	sort.Sort(params)
	pp := *params
	buf := bytebufferpool.Get()
	for _, p := range pp {
		_, _ = buf.WriteString(fmt.Sprintf("&%s=%s", bytex.ToString(p.key), bytex.ToString(p.val)))
	}
	p := buf.Bytes()[1:]
	bytebufferpool.Put(buf)
	return p
}

var (
	stringType  = reflect.TypeOf("")
	boolType    = reflect.TypeOf(false)
	intType     = reflect.TypeOf(0)
	int32Type   = reflect.TypeOf(int32(0))
	int64Type   = reflect.TypeOf(int64(0))
	float32Type = reflect.TypeOf(float32(0.0))
	float64Type = reflect.TypeOf(float64(0))
	uintType    = reflect.TypeOf(uint(0))
	uint32Type  = reflect.TypeOf(uint32(0))
	uint64Type  = reflect.TypeOf(uint64(0))
	timeType    = reflect.TypeOf(time.Time{})
)

func DecodeParams(params Params, dst interface{}) (err error) {
	if dst == nil {
		err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("dst target is nil"))
		return
	}
	if params.Len() == 0 {
		return
	}
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer {
		err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("dst target is not pointer"))
		return
	}
	rv := rv.Elem()
	if rv.Kind() != reflect.Struct {
		err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("dst target is not pointer struct"))
		return
	}
	rt := rv.Type()
	fieldNum := rt.NumField()
	for i := 0; i < fieldNum; i++ {
		ft := rt.Field(i)
		if !ft.IsExported() {
			continue
		}
		if ft.Anonymous {
			if ft.Type.Kind() != reflect.Struct {
				err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("dst target can not has not struct typed anonymous field"))
				return
			}
			anonymous := reflect.NewAt(ft.Type, rv.Field(i).UnsafePointer())
			err = DecodeParams(params, anonymous.Interface())
			if err != nil {
				return
			}
		}
		name := ft.Name
		tag, hasTag := ft.Tag.Lookup("json")
		if hasTag {
			if tag == "-" {
				continue
			}
			n := strings.Index(tag, ",")
			if n > 0 {
				tag = tag[0:n]
			}
			name = tag
		}
		pv := bytes.TrimSpace(params.Get(bytex.FromString(name)))
		if len(pv) == 0 {
			continue
		}
		fv := rv.Field(i)
		if ft.Type.ConvertibleTo(stringType) {
			fv.SetString(bytex.ToString(pv))
		} else if ft.Type.ConvertibleTo(boolType) {
			b, parseErr := strconv.ParseBool(bytex.ToString(pv))
			if parseErr != nil {
				err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not bool", name))
				return
			}
			fv.SetBool(b)
		} else if ft.Type.ConvertibleTo(intType) || ft.Type.ConvertibleTo(int32Type) || ft.Type.ConvertibleTo(int64Type) {
			n, parseErr := strconv.ParseInt(bytex.ToString(pv), 10, 64)
			if parseErr != nil {
				err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not int", name))
				return
			}
			fv.SetInt(n)
		} else if ft.Type.ConvertibleTo(float32Type) || ft.Type.ConvertibleTo(float64Type) {
			f, parseErr := strconv.ParseFloat(bytex.ToString(pv), 64)
			if parseErr != nil {
				err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not float", name))
				return
			}
			fv.SetFloat(f)
		} else if ft.Type.ConvertibleTo(uintType) || ft.Type.ConvertibleTo(uint32Type) || ft.Type.ConvertibleTo(uint64Type) {
			u, parseErr := strconv.ParseUint(bytex.ToString(pv), 10, 64)
			if parseErr != nil {
				err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not uint", name))
				return
			}
			fv.SetUint(u)
		} else if ft.Type.ConvertibleTo(timeType) {
			t, parseErr := time.Parse(time.RFC3339, bytex.ToString(pv))
			if parseErr != nil {
				err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not RFC3339 time", name))
				return
			}
			fv.Set(reflect.ValueOf(t))
		} else if ft.Type.Kind() == reflect.Slice {

		} else {

		}
	}
	return
}
