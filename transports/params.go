/*
 * Copyright 2023 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package transports

import (
	"bytes"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/scanner"
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
	Add(name []byte, value []byte)
	Values(name []byte) [][]byte
	Remove(name []byte)
	Len() int
	Encode() (p []byte)
}

func ParamsScanner(params Params) scanner.Scanner {
	return paramsScanner{
		value: params,
	}
}

type paramsScanner struct {
	value Params
}

func (p paramsScanner) Exist() (ok bool) {
	ok = p.value.Len() > 0
	return
}

func (p paramsScanner) Scan(dst interface{}) (err error) {
	err = DecodeParams(p.value, dst)
	return
}

func (p paramsScanner) MarshalJSON() ([]byte, error) {
	encoded := p.value.Encode()
	capacity := len(encoded) + 2
	b := make([]byte, capacity)
	b[0] = '"'
	b[capacity-1] = '"'
	copy(b[1:], encoded)
	return b, nil
}

type paramValues [][]byte

func (values paramValues) Len() int {
	return len(values)
}

func (values paramValues) Less(i, j int) bool {
	return bytes.Compare(values[i], values[j]) < 0
}

func (values paramValues) Swap(i, j int) {
	values[i], values[j] = values[j], values[i]
}

type param struct {
	key []byte
	val paramValues
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
			return p.val[0]
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
			p.val = [][]byte{value}
			*params = pp
			return
		}
	}
	pp = append(pp, param{
		key: name,
		val: [][]byte{value},
	})
	*params = pp
}

func (params *defaultParams) Add(name []byte, value []byte) {
	if name == nil || value == nil {
		return
	}
	if len(name) == 0 {
		return
	}
	pp := *params
	for i, p := range pp {
		if bytes.Equal(p.key, name) {
			p.val = append(p.val, value)
			pp[i] = p
			*params = pp
			return
		}
	}
	pp = append(pp, param{
		key: name,
		val: [][]byte{value},
	})
	*params = pp
}

func (params *defaultParams) Values(name []byte) [][]byte {
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
		if p.val.Len() == 1 {
			_, _ = buf.WriteString(fmt.Sprintf("&%s=%s", bytex.ToString(p.key), bytex.ToString(p.val[0])))
			continue
		}
		values := p.val
		sort.Sort(values)
		for _, value := range values {
			_, _ = buf.WriteString(fmt.Sprintf("&%s=%s", bytex.ToString(p.key), bytex.ToString(value)))
		}
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
	rv = rv.Elem()
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
			anonymous := rv.Field(i).Addr().Interface()
			err = DecodeParams(params, anonymous)
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
		switch ft.Type.Kind() {
		case reflect.String:
			s := bytex.ToString(pv)
			if ft.Type == stringType {
				fv.SetString(s)
			} else {
				fv.Set(reflect.ValueOf(s).Convert(ft.Type))
			}
			break
		case reflect.Bool:
			b, parseErr := strconv.ParseBool(bytex.ToString(pv))
			if parseErr != nil {
				err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not bool", name))
				return
			}
			if ft.Type == boolType {
				fv.SetBool(b)
			} else {
				fv.Set(reflect.ValueOf(b).Convert(ft.Type))
			}
			break
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, parseErr := strconv.ParseInt(bytex.ToString(pv), 10, 64)
			if parseErr != nil {
				err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not int", name))
				return
			}
			if ft.Type == intType || ft.Type == int32Type || ft.Type == int64Type {
				fv.SetInt(n)
			} else {
				fv.Set(reflect.ValueOf(n).Convert(ft.Type))
			}
			break
		case reflect.Float32, reflect.Float64:
			f, parseErr := strconv.ParseFloat(bytex.ToString(pv), 64)
			if parseErr != nil {
				err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not float", name))
				return
			}
			if ft.Type == float32Type || ft.Type == float64Type {
				fv.SetFloat(f)
			} else {
				fv.Set(reflect.ValueOf(f).Convert(ft.Type))
			}
			break
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			u, parseErr := strconv.ParseUint(bytex.ToString(pv), 10, 64)
			if parseErr != nil {
				err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not uint", name))
				return
			}
			if ft.Type == uintType || ft.Type == uint32Type || ft.Type == uint64Type {
				fv.SetUint(u)
			} else {
				fv.Set(reflect.ValueOf(u).Convert(ft.Type))
			}
			break
		case reflect.Struct:
			if ft.Type == timeType || timeType.ConvertibleTo(ft.Type) {
				t, parseErr := time.Parse(time.RFC3339, bytex.ToString(pv))
				if parseErr != nil {
					err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not RFC3339 time", name))
					return
				}
				if ft.Type == timeType {
					fv.Set(reflect.ValueOf(t))
				} else {
					fv.Set(reflect.ValueOf(t).Convert(ft.Type))
				}
			} else {
				err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("type of %s is not supported", name))
				return
			}
			break
		case reflect.Slice:
			// pvv values or splits
			pvv := params.Values(bytex.FromString(name))
			if len(pvv) == 1 {
				pvv = bytes.Split(pvv[0], []byte{','})
			}
			for pi, pvx := range pvv {
				pvv[pi] = bytes.TrimSpace(pvx)
			}
			eft := ft.Type.Elem()
			switch eft.Kind() {
			case reflect.String:
				ss := reflect.MakeSlice(ft.Type, 0, 1)
				for _, pvx := range pvv {
					s := bytex.ToString(pvx)
					e := reflect.New(eft).Elem()
					if e.Type() == stringType {
						e.SetString(s)
					} else {
						e.Set(reflect.ValueOf(s).Convert(e.Type()))
					}
					ss = reflect.Append(ss, e)
				}
				fv.Set(ss)
				break
			case reflect.Bool:
				bb := reflect.MakeSlice(ft.Type, 0, 1)
				for _, pvx := range pvv {
					b, parseErr := strconv.ParseBool(bytex.ToString(pvx))
					if parseErr != nil {
						err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not bool", name))
						return
					}
					e := reflect.New(eft).Elem()
					if e.Type() == boolType {
						e.SetBool(b)
					} else {
						e.Set(reflect.ValueOf(b).Convert(e.Type()))
					}
					bb = reflect.Append(bb, e)
				}
				fv.Set(bb)
				break
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				nn := reflect.MakeSlice(ft.Type, 0, 1)
				for _, pvx := range pvv {
					n, parseErr := strconv.ParseInt(bytex.ToString(pvx), 10, 64)
					if parseErr != nil {
						err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not int", name))
						return
					}
					e := reflect.New(eft).Elem()
					if e.Type() == intType || e.Type() == int32Type || e.Type() == int64Type {
						e.SetInt(n)
					} else {
						e.Set(reflect.ValueOf(n).Convert(e.Type()))
					}
					nn = reflect.Append(nn, e)
				}
				fv.Set(nn)
				break
			case reflect.Float32, reflect.Float64:
				ff := reflect.MakeSlice(ft.Type, 0, 1)
				for _, pvx := range pvv {
					f, parseErr := strconv.ParseFloat(bytex.ToString(pvx), 64)
					if parseErr != nil {
						err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not float", name))
						return
					}
					e := reflect.New(eft).Elem()
					if e.Type() == float32Type || e.Type() == float64Type {
						e.SetFloat(f)
					} else {
						e.Set(reflect.ValueOf(f).Convert(e.Type()))
					}
					ff = reflect.Append(ff, e)
				}
				fv.Set(ff)
				break
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				uu := reflect.MakeSlice(ft.Type, 0, 1)
				for _, pvx := range pvv {
					u, parseErr := strconv.ParseUint(bytex.ToString(pvx), 10, 64)
					if parseErr != nil {
						err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not uint", name))
						return
					}
					e := reflect.New(eft).Elem()
					if e.Type() == uintType || e.Type() == uint32Type || e.Type() == uint64Type {
						e.SetUint(u)
					} else {
						e.Set(reflect.ValueOf(u).Convert(e.Type()))
					}
					uu = reflect.Append(uu, e)
				}
				fv.Set(uu)
				break
			case reflect.Struct:
				if eft == timeType || timeType.ConvertibleTo(eft) {
					tt := reflect.MakeSlice(ft.Type, 0, 1)
					for _, pvx := range pvv {
						t, parseErr := time.Parse(time.RFC3339, bytex.ToString(pvx))
						if parseErr != nil {
							err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("%s is not RFC3339 time", name))
							return
						}
						e := reflect.New(eft).Elem()
						if e.Type() == timeType {
							e.Set(reflect.ValueOf(t))
						} else {
							e.Set(reflect.ValueOf(t).Convert(e.Type()))
						}
						tt = reflect.Append(tt, e)
					}
					fv.Set(tt)
				} else {
					err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("type of %s is not supported", name))
					return
				}
				break
			default:
				err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("type of %s is not supported", name))
				return
			}
			break
		default:
			err = errors.Warning("fns: decode param failed").WithCause(fmt.Errorf("type of %s is not supported", name))
			return
		}
	}
	return
}
