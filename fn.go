/*
 * Copyright 2021 Wang Min Xiang
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
 */

package fns

import (
	"fmt"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	httpHeaderNamespace     = "X-Fns-Namespace"
	httpHeaderFnName        = "X-Fns-Name"
	httpHeaderRequestId     = "X-Fns-Request-Id"
	httpHeaderAuthorization = "Authorization"
)

// +-------------------------------------------------------------------------------------------------------------------+

type RequestHeader interface {
	Get(name string) (value string, has bool)
}

type httpRequestHeader struct {
	header *fasthttp.RequestHeader
}

func (h *httpRequestHeader) Get(name string) (value string, has bool) {
	v := h.header.Peek(name)
	if v == nil || len(v) == 0 {
		return
	}
	value = string(v)
	has = true
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type Argument interface {
	Scan(v interface{}) (err error)
	Data() (data []byte)
}

func EmptyArgument() (arg Argument) {
	arg = &requestArgument{
		isJson: true,
		raw:    []byte{'{', '}'},
		values: nil,
		files:  nil,
	}
	return
}

func NewArgument(v interface{}) (arg Argument) {
	if v == nil {
		arg = EmptyArgument()
		return
	}
	arg = &requestArgument{
		isJson: true,
		raw:    JsonEncode(v),
		values: nil,
		files:  nil,
	}
	return
}

func newArgumentFromHttpRequest(request *fasthttp.Request) (arg Argument, err error) {
	requestContentType := strings.ToLower(strings.TrimSpace(string(request.Header.ContentType())))
	if requestContentType == "application/json" {
		raw := request.Body()
		if !JsonAPI().Valid(raw) {
			err = fmt.Errorf("request body is not json")
			return
		}
		arg = &requestArgument{
			isJson: true,
			raw:    raw,
		}
		return
	}
	if requestContentType == "multipart/form-data" {
		arg0 := &requestArgument{
			isJson: false,
			values: make(map[string][]string),
			files:  make(map[string][]FnFile),
		}
		mf, getMfErr := request.MultipartForm()
		if getMfErr != nil {
			err = fmt.Errorf("request body has no multi part form")
			return
		}
		if mf == nil || mf.File == nil || len(mf.File) == 0 {
			err = fmt.Errorf("request body has no file")
			return
		}
		for key, fhs := range mf.File {
			fnFiles := make([]FnFile, 0, 1)
			for _, fh := range fhs {
				fileName := fh.Filename
				fileSuffix := path.Ext(fileName)
				tmpFile, openErr := fh.Open()
				if openErr != nil {
					err = fmt.Errorf("open request file failed, %v", openErr)
					return
				}
				fileContent, readErr := ioutil.ReadAll(tmpFile)
				if readErr != nil {
					err = fmt.Errorf("read request file failed, %v", readErr)
					_ = tmpFile.Close()
					return
				}
				_ = tmpFile.Close()
				fileType := http.DetectContentType(fileContent)
				fnFiles = append(fnFiles, FnFile{
					Name:    fileName,
					Suffix:  fileSuffix,
					Type:    fileType,
					Content: fileContent,
				})
			}
			arg0.files[key] = fnFiles
		}

		postArg := request.PostArgs()
		if postArg != nil && postArg.Len() > 0 {
			keys := make([][]byte, 0, 1)
			postArg.VisitAll(func(key []byte, _ []byte) {
				_, has := arg0.files[string(key)]
				if has {
					return
				}
				keys = append(keys, key)
			})
			for _, key := range keys {
				multiBytes := postArg.PeekMultiBytes(key)
				if multiBytes == nil || len(multiBytes) == 0 {
					return
				}
				values := make([]string, 0, len(multiBytes))
				for _, multiByte := range multiBytes {
					values = append(values, string(multiByte))
				}
				arg0.values[string(key)] = values
			}
		}

		arg = arg0
		return
	}
	if requestContentType == "application/x-www-form-urlencoded" {
		arg0 := &requestArgument{
			isJson: false,
			values: make(map[string][]string),
			files:  nil,
		}
		postArg := request.PostArgs()
		if postArg != nil && postArg.Len() > 0 {
			keys := make([][]byte, 0, 1)
			postArg.VisitAll(func(key []byte, _ []byte) {
				keys = append(keys, key)
			})
			for _, key := range keys {
				multiBytes := postArg.PeekMultiBytes(key)
				if multiBytes == nil || len(multiBytes) == 0 {
					return
				}
				values := make([]string, 0, len(multiBytes))
				for _, multiByte := range multiBytes {
					values = append(values, string(multiByte))
				}
				arg0.values[string(key)] = values
			}
		}
		arg = arg0
		return
	}

	err = fmt.Errorf("fns create argument failed, request content type (%s) is not support", requestContentType)

	return
}

type requestArgument struct {
	isJson bool
	raw    []byte
	values map[string][]string
	files  map[string][]FnFile
}

func (arg *requestArgument) Data() (data []byte) {
	if arg.isJson {
		data = arg.raw
		return
	}
	obj := NewJsonObject()
	if arg.values != nil && len(arg.values) > 0 {
		for k, v := range arg.values {
			_ = obj.Put(k, v)
		}
	}
	if arg.files != nil && len(arg.files) > 0 {
		for k, v := range arg.files {
			_ = obj.Put(k, v)
		}
	}
	data = obj.Raw()
	return
}

func (arg *requestArgument) Scan(v interface{}) (err error) {
	if arg.isJson {
		err = arg.scanJson(v)
	} else {
		err = arg.scanForm(v)
	}
	decodeErr := JsonAPI().Unmarshal(arg.raw, v)
	if decodeErr != nil {
		err = fmt.Errorf("fns argument scan failed, %v", decodeErr)
		return
	}
	return
}

func (arg *requestArgument) scanJson(v interface{}) (err error) {
	decodeErr := JsonAPI().Unmarshal(arg.raw, v)
	if decodeErr != nil {
		err = fmt.Errorf("fns argument scan failed, %v", decodeErr)
		return
	}
	return
}

const (
	argumentsStructTagForm = "form"
)

func (arg *requestArgument) scanForm(v interface{}) (err error) {
	if v == nil {
		err = fmt.Errorf("fns scan fn arguments failed, target is nil")
		return
	}
	targetType := reflect.TypeOf(v)
	if targetType.Kind() != reflect.Ptr {
		err = fmt.Errorf("fns scan fn arguments failed, type of target is not ptr")
		return
	}
	targetElemType := targetType.Elem()
	if targetElemType.Kind() != reflect.Struct {
		err = fmt.Errorf("fns scan fn arguments failed, type of target element is not struct")
		return
	}

	argFields, argFieldsErr := fnArgumentsGetField(targetElemType)
	if argFieldsErr != nil {
		err = argFieldsErr
		return
	}

	if len(argFields) == 0 {
		return
	}

	targetValue := reflect.ValueOf(v)

	for _, field := range argFields {

		if field.isFile() {
			values, has := arg.files[field.name]
			if has {
				setFieldValueErr := setFnHttpRequestArgumentsFieldWithFile(values, field, targetValue)
				if setFieldValueErr != nil {
					err = setFieldValueErr
					return
				}
			}
		} else {
			values, has := arg.values[field.name]
			if has {
				setFieldValueErr := setFnHttpRequestArgumentsFieldWithForm(values, field, targetValue)
				if setFieldValueErr != nil {
					err = setFieldValueErr
					return
				}
			}
		}
	}

	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type FnFiles []FnFile

type FnFile struct {
	Name    string `json:"name,omitempty"`
	Suffix  string `json:"suffix,omitempty"`
	Type    string `json:"fileType,omitempty"`
	Content []byte `json:"content,omitempty"`
}

func (f FnFile) Exist() (ok bool) {
	ok = len(f.Content) > 0
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

var (
	fnFileType   = reflect.TypeOf(FnFile{})
	fnFilesType  = reflect.TypeOf([]FnFile{})
	fnFiles2Type = reflect.TypeOf(FnFiles{})
)

type fnArgumentsField struct {
	fieldIdx  int
	fieldType reflect.Type
	name      string
}

func (f *fnArgumentsField) isFile() (ok bool) {
	ok = f.fieldType == fnFileType || f.fieldType == fnFilesType || f.fieldType == fnFiles2Type
	return
}

func fnArgumentsGetField(_type reflect.Type) (argFields []fnArgumentsField, err error) {
	fieldNum := _type.NumField()
	for i := 0; i < fieldNum; i++ {
		field := _type.Field(i)
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			err = fmt.Errorf("fns scan fn arguments failed, %s of %s can not be ptr", field.Name, _type.Name())
			return
		}
		fieldTag := field.Tag
		key, has := fieldTag.Lookup(argumentsStructTagForm)
		if !has {
			continue
		}
		argField := fnArgumentsField{
			fieldIdx:  i,
			fieldType: fieldType,
			name:      key,
		}

		argFields = append(argFields, argField)
	}
	return
}

func setFnHttpRequestArgumentsFieldWithFile(values []FnFile, field fnArgumentsField, targetValue reflect.Value) (err error) {
	if field.fieldType == fnFileType {
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(values[0]))
		return
	}
	if field.fieldType == fnFilesType {
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(values))
		return
	}
	if field.fieldType == fnFiles2Type {
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(FnFiles(values)))
		return
	}
	return
}

func setFnHttpRequestArgumentsFieldWithForm(values []string, field fnArgumentsField, targetValue reflect.Value) (err error) {
	// string and array
	if field.fieldType.Kind() == reflect.String {
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(values[0]))
		return
	}
	if field.fieldType == reflect.TypeOf([]string{}) {
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(values))
		return
	}
	// int and array
	if field.fieldType.Kind() == reflect.Int {
		fieldValue, changeErr := strconv.Atoi(values[0])
		if changeErr != nil {
			err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
			return
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValue))
		return
	}
	if field.fieldType == reflect.TypeOf([]int{}) {
		fieldValues := make([]int, 0, 1)
		for _, value := range values {
			changedValue, changeErr := strconv.Atoi(value)
			if changeErr != nil {
				err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
				return
			}
			fieldValues = append(fieldValues, changedValue)
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValues))
		return
	}
	// int32 and array
	if field.fieldType.Kind() == reflect.Int32 {
		fieldValue, changeErr := strconv.ParseInt(values[0], 10, 32)
		if changeErr != nil {
			err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
			return
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValue))
		return
	}
	if field.fieldType == reflect.TypeOf([]int32{}) {
		fieldValues := make([]int64, 0, 1)
		for _, value := range values {
			changedValue, changeErr := strconv.ParseInt(value, 10, 32)
			if changeErr != nil {
				err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
				return
			}
			fieldValues = append(fieldValues, changedValue)
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValues))
		return
	}
	// int64 and array
	if field.fieldType.Kind() == reflect.Int64 {
		fieldValue, changeErr := strconv.ParseInt(values[0], 10, 64)
		if changeErr != nil {
			err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
			return
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValue))
		return
	}
	if field.fieldType == reflect.TypeOf([]int64{}) {
		fieldValues := make([]int64, 0, 1)
		for _, value := range values {
			changedValue, changeErr := strconv.ParseInt(value, 10, 64)
			if changeErr != nil {
				err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
				return
			}
			fieldValues = append(fieldValues, changedValue)
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValues))
		return
	}
	// uint and array
	if field.fieldType.Kind() == reflect.Uint {
		fieldValue, changeErr := strconv.ParseUint(values[0], 10, 64)
		if changeErr != nil {
			err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
			return
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(uint(fieldValue)))
		return
	}
	if field.fieldType == reflect.TypeOf([]uint{}) {
		fieldValues := make([]uint, 0, 1)
		for _, value := range values {
			changedValue, changeErr := strconv.ParseUint(value, 10, 64)
			if changeErr != nil {
				err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
				return
			}
			fieldValues = append(fieldValues, uint(changedValue))
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValues))
		return
	}
	// uint32 and array
	if field.fieldType.Kind() == reflect.Uint32 {
		fieldValue, changeErr := strconv.ParseUint(values[0], 10, 32)
		if changeErr != nil {
			err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
			return
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(uint32(fieldValue)))
		return
	}
	if field.fieldType == reflect.TypeOf([]uint32{}) {
		fieldValues := make([]uint32, 0, 1)
		for _, value := range values {
			changedValue, changeErr := strconv.ParseUint(value, 10, 32)
			if changeErr != nil {
				err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
				return
			}
			fieldValues = append(fieldValues, uint32(changedValue))
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValues))
		return
	}
	// uint64 and array
	if field.fieldType.Kind() == reflect.Int64 {
		fieldValue, changeErr := strconv.ParseUint(values[0], 10, 64)
		if changeErr != nil {
			err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
			return
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValue))
		return
	}
	if field.fieldType == reflect.TypeOf([]uint64{}) {
		fieldValues := make([]uint64, 0, 1)
		for _, value := range values {
			changedValue, changeErr := strconv.ParseUint(value, 10, 64)
			if changeErr != nil {
				err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
				return
			}
			fieldValues = append(fieldValues, changedValue)
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValues))
		return
	}
	// float32 and array
	if field.fieldType.Kind() == reflect.Float32 {
		fieldValue, changeErr := strconv.ParseFloat(values[0], 32)
		if changeErr != nil {
			err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
			return
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValue))
		return
	}
	if field.fieldType == reflect.TypeOf([]float32{}) {
		fieldValues := make([]float32, 0, 1)
		for _, value := range values {
			changedValue, changeErr := strconv.ParseFloat(value, 32)
			if changeErr != nil {
				err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
				return
			}
			fieldValues = append(fieldValues, float32(changedValue))
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValues))
		return
	}
	// float64 and array
	if field.fieldType.Kind() == reflect.Float64 {
		fieldValue, changeErr := strconv.ParseFloat(values[0], 64)
		if changeErr != nil {
			err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
			return
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValue))
		return
	}
	if field.fieldType == reflect.TypeOf([]float64{}) {
		fieldValues := make([]float64, 0, 1)
		for _, value := range values {
			changedValue, changeErr := strconv.ParseFloat(value, 64)
			if changeErr != nil {
				err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
				return
			}
			fieldValues = append(fieldValues, changedValue)
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValues))
		return
	}
	// bool
	if field.fieldType.Kind() == reflect.Bool {
		if strings.ToLower(strings.TrimSpace(values[0])) == "true" {
			targetValue.Field(field.fieldIdx).SetBool(true)
		}
		return
	}
	// time and array
	if field.fieldType == reflect.TypeOf(time.Time{}) {
		value := strings.TrimSpace(values[0])
		if value == "" {
			return
		}
		changedValue, changeErr := ParseTime(value)
		if changeErr != nil {
			err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
			return
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(changedValue))
		return
	}
	if field.fieldType == reflect.TypeOf([]time.Time{}) {
		fieldValues := make([]time.Time, 0, 1)
		for _, value := range values {
			changedValue, changeErr := ParseTime(value)
			if changeErr != nil {
				err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed", field.name)
				return
			}
			fieldValues = append(fieldValues, changedValue)
		}
		targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fieldValues))
		return
	}
	// unsupported
	err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed, type %s is unsupported", field.name, field.fieldType.String())
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type Result interface {
	Scan(v interface{}) (err CodeError)
	Succeed(v interface{})
	Failed(err CodeError)
}

func NewResult() (result Result) {
	result = &futureResult{
		ch: make(chan []byte, 1),
	}
	return
}

type futureResult struct {
	ch chan []byte
}

func (r *futureResult) Scan(v interface{}) (err CodeError) {
	if v == nil {
		err = ServiceError("result scan failed for target is nil")
		return
	}
	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		err = ServiceError("result scan failed for type kind of target is not ptr")
		return
	}

	data, ok := <-r.ch
	if !ok {
		return
	}

	if data[0] == 0 {
		codeErr, decodeOk := DecodeErrorFromJson(data[1:])
		if !decodeOk {
			codeErr = ServiceError("decode code error from result failed")
		}
		err = codeErr
		return
	}
	decodeErr := JsonAPI().Unmarshal(data[1:], v)
	if decodeErr != nil {
		err = ServiceError("decode result failed")
		return
	}
	return
}

func (r *futureResult) Succeed(v interface{}) {
	if v == nil {
		close(r.ch)
		return
	}
	buf := bytebufferpool.Get()

	switch v.(type) {
	case []byte:
		data := v.([]byte)
		if len(data) > 0 {
			_ = buf.WriteByte(1)
			_, _ = buf.Write(data)
		} else {
			_ = buf.WriteByte(0)
			_, _ = buf.Write(JsonEncode(ServiceError("empty result")))
		}
	default:
		data, encodeErr := JsonAPI().Marshal(v)
		if encodeErr != nil {
			_ = buf.WriteByte(0)
			_, _ = buf.Write(JsonEncode(ServiceError("empty result")))
		} else {
			_ = buf.WriteByte(1)
			_, _ = buf.Write(data)
		}
	}
	r.ch <- buf.Bytes()
	close(r.ch)

	bytebufferpool.Put(buf)
}

func (r *futureResult) Failed(err CodeError) {
	if err == nil {
		close(r.ch)
		return
	}
	buf := bytebufferpool.Get()

	_ = buf.WriteByte(0)
	_, _ = buf.Write(JsonEncode(err))

	r.ch <- buf.Bytes()
	close(r.ch)

	bytebufferpool.Put(buf)
}
