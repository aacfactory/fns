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
	"github.com/aacfactory/eventbus"
	"io/ioutil"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	argumentsStructTag      = "arg"
	argumentsStructTagHead  = "head"
	argumentsStructTagQuery = "query"
	argumentsStructTagBody  = "body"
	argumentsStructTagPath  = "path"
	argumentsStructTagFile  = "file"
)

type FnFile struct {
	Name    string `json:"name,omitempty"`
	Suffix  string `json:"suffix,omitempty"`
	Type    string `json:"fileType,omitempty"`
	Content []byte `json:"content,omitempty"`
}

type Arguments interface {
	Scan(v interface{}) (err error)
}

/*FnProxy
fn eb client：arguments 扫后调用 eventbus send
fn eb handle: event 调 FnProxy，然后 arguments 扫后调用 fn
*/
type FnProxy func(fc FnContext, arguments Arguments, tags ...string) (result interface{}, err error)

// +-------------------------------------------------------------------------------------------------------------------+

var (
	fnFileType  = reflect.TypeOf(FnFile{})
	fnFilesType = reflect.TypeOf([]FnFile{})
)

type fnArgumentsField struct {
	fieldIdx   int
	fieldType  reflect.Type
	name       string
	sourceType string
}

func fnArgumentsGetField(_type reflect.Type) (argFields []fnArgumentsField, err error) {
	argFields = make([]fnArgumentsField, 0, 1)
	fieldNum := _type.NumField()
	for i := 0; i < fieldNum; i++ {
		field := _type.Field(i)
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			err = fmt.Errorf("fns scan fn arguments failed, %s of %s can not be ptr", field.Name, _type.Name())
			return
		}
		fieldTag := field.Tag
		tag, has := fieldTag.Lookup(argumentsStructTag)
		if !has {
			continue
		}
		argField := fnArgumentsField{
			fieldIdx:  i,
			fieldType: fieldType,
		}
		if strings.Contains(tag, ",") {
			tagItems := strings.Split(tag, ",")
			name := strings.TrimSpace(tagItems[0])
			if name == "" {
				err = fmt.Errorf("fns scan fn arguments failed, struct tag of %s %s field has no name", _type.Name(), field.Name)
				return
			}
			argField.name = name
			for r := 1; r < len(tagItems); r++ {
				tagItem := strings.TrimSpace(tagItems[r])
				if argumentsStructTagHead == tagItem {
					argField.sourceType = argumentsStructTagHead
					if !(argField.fieldType.Kind() == reflect.String || argField.fieldType.String() == "[]string") {
						err = fmt.Errorf("fns scan fn arguments failed, %s of %s is head, so must be string or []string", field.Name, _type.Name())
						return
					}
				} else if argumentsStructTagFile == tagItem {
					if !(field.Type == fnFileType || field.Type == fnFilesType) {
						err = fmt.Errorf("fns scan fn arguments failed, %s of %s is file, so must be fns.FnFile or []fns.FnFile", field.Name, _type.Name())
						return
					}
					argField.sourceType = argumentsStructTagFile
				} else if argumentsStructTagQuery == tagItem {
					argField.sourceType = argumentsStructTagQuery
				} else if argumentsStructTagBody == tagItem {
					argField.sourceType = argumentsStructTagBody
				} else if argumentsStructTagPath == tagItem {
					argField.sourceType = argumentsStructTagPath
				}
			}

			argFields = append(argFields, argField)
		}
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func newFnEventArguments(event eventbus.Event) Arguments {
	if event == nil {
		panic(fmt.Sprintf("fns create event typed Arguments failed, event is nil"))
	}
	return &fnEventArguments{
		event: event,
	}
}

type fnEventArguments struct {
	event eventbus.Event
}

func (args *fnEventArguments) Scan(target interface{}) (err error) {
	if target == nil {
		err = fmt.Errorf("fns scan fn arguments failed, target is nil")
		return
	}
	targetType := reflect.TypeOf(target)
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

	targetValue := reflect.ValueOf(target)

	// body
	if args.event.Body() != nil && len(args.event.Body()) >= 2 {
		decodeBodyErr := JsonAPI().Unmarshal(args.event.Body(), target)
		if decodeBodyErr != nil {
			err = fmt.Errorf("fns scan fn arguments for decode body")
			return
		}
	}

	// head
	if args.event.Head() != nil {
		for _, argField := range argFields {
			if argField.sourceType == argumentsStructTagHead {
				field := targetValue.Field(argField.fieldIdx)
				if argField.fieldType.Kind() == reflect.String {
					headValue, hasValue := args.event.Head().Get(argField.name)
					if hasValue {
						field.SetString(headValue)
					}
				} else if argField.fieldType.String() == "[]string" {
					headValue, hasValue := args.event.Head().Values(argField.name)
					if hasValue {
						field.Set(reflect.ValueOf(headValue))
					}
				} else {
					err = fmt.Errorf("fns scan fn arguments failed, %s of %s is head, so must be string or []string", targetType.Field(argField.fieldIdx).Name, targetType.Name())
					return
				}
			}
		}
	}

	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func newFnHttpRequestArguments(request *http.Request) Arguments {
	if request == nil {
		panic(fmt.Sprintf("fns create event typed Arguments failed, http request is nil"))
	}
	return &fnHttpRequestArguments{
		request: request,
	}
}

type fnHttpRequestArguments struct {
	request *http.Request
}

func (args *fnHttpRequestArguments) Scan(target interface{}) (err error) {
	if target == nil {
		err = fmt.Errorf("fns scan fn arguments failed, target is nil")
		return
	}
	targetType := reflect.TypeOf(target)
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

	hasHead := false
	hasQuery := false
	hasBody := false
	hasFile := false
	for _, argField := range argFields {
		if argField.sourceType == argumentsStructTagHead && !hasHead {
			hasHead = true
			continue
		}
		if argField.sourceType == argumentsStructTagQuery && !hasQuery {
			hasQuery = true
			continue
		}
		if argField.sourceType == argumentsStructTagBody && !hasBody {
			hasBody = true
			continue
		}
		if argField.sourceType == argumentsStructTagFile && !hasFile {
			hasFile = true
			continue
		}
	}

	requestContentType := strings.ToLower(strings.TrimSpace(args.request.Header.Get("Content-Type")))
	formedBody := hasBody && !hasFile && requestContentType == "application/x-www-form-urlencoded"
	jsonBody := hasBody && !hasFile && requestContentType == "application/json"
	if hasFile {
		if requestContentType != "multipart/form-data" {
			err = fmt.Errorf("fns scan fn arguments failed, has file field but content type of request is not multipart/form-data")
			return
		}
		parseFormErr := args.request.ParseMultipartForm(HttpMaxRequestBodySize)
		if parseFormErr != nil {
			err = fmt.Errorf("fns scan fn arguments failed, has file field but parse http MultipartForm failed")
			return
		}
	}
	if formedBody {
		parseFormErr := args.request.ParseForm()
		if parseFormErr != nil {
			err = fmt.Errorf("fns scan fn arguments failed, has file field but parse http x-www-form-urlencoded form failed")
			return
		}
	}

	if hasFile || formedBody {

		targetValue := reflect.ValueOf(target)

		for _, field := range argFields {
			// file
			if field.sourceType == argumentsStructTagFile {
				requestFiles, hasFiles := args.request.MultipartForm.File[field.name]
				if !hasFiles {
					continue
				}
				if field.fieldType == fnFileType {
					fileInfo := requestFiles[0]
					fileName := fileInfo.Filename
					fileSuffix := path.Ext(fileName)
					tmpFile, openErr := fileInfo.Open()
					if openErr != nil {
						err = fmt.Errorf("fns scan fn arguments failed, open %s file failed", field.name)
						return
					}
					fileContent, readErr := ioutil.ReadAll(tmpFile)
					if readErr != nil {
						err = fmt.Errorf("fns scan fn arguments failed, read %s file failed", field.name)
						_ = tmpFile.Close()
						return
					}
					_ = tmpFile.Close()
					fileType := http.DetectContentType(fileContent)

					fnFile := FnFile{
						Name:    fileName,
						Suffix:  fileSuffix,
						Type:    fileType,
						Content: fileContent,
					}
					targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fnFile))
				} else if field.fieldType == fnFilesType {
					filesNum := len(requestFiles)
					fnFiles := make([]FnFile, 0, 1)
					for i := 0; i < filesNum; i++ {
						fileInfo := requestFiles[i]
						fileName := fileInfo.Filename
						fileSuffix := path.Ext(fileName)
						tmpFile, openErr := fileInfo.Open()
						if openErr != nil {
							err = fmt.Errorf("fns scan fn arguments failed, open %s file failed", field.name)
							return
						}
						fileContent, readErr := ioutil.ReadAll(tmpFile)
						if readErr != nil {
							err = fmt.Errorf("fns scan fn arguments failed, read %s file failed", field.name)
							_ = tmpFile.Close()
							return
						}
						_ = tmpFile.Close()
						fileType := http.DetectContentType(fileContent)

						fnFile := FnFile{
							Name:    fileName,
							Suffix:  fileSuffix,
							Type:    fileType,
							Content: fileContent,
						}
						fnFiles = append(fnFiles, fnFile)
					}
					targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(fnFiles))
				}
			}
			// body
			if field.sourceType == argumentsStructTagBody {
				values, hasValues := args.request.PostForm[field.name]
				if !hasValues || len(values) == 0 {
					continue
				}
				if field.fieldType.Kind() == reflect.String {
					targetValue.Field(field.fieldIdx).SetString(values[0])
					continue
				}
				if field.fieldType == reflect.TypeOf([]string{}) {
					targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(values))
					continue
				}
				setFieldValueErr := setFnHttpRequestArgumentsField(values, field, targetValue)
				if setFieldValueErr != nil {
					err = setFieldValueErr
					return
				}
			}
		}
	} else if jsonBody && args.request.ContentLength > 0 {
		bodyContent, readBodyErr := ioutil.ReadAll(args.request.Body)
		if readBodyErr != nil {
			_ = args.request.Body.Close()
			err = fmt.Errorf("fns scan fn arguments failed, read json body failed")
			return
		}
		_ = args.request.Body.Close()
		decodeErr := JsonAPI().Unmarshal(bodyContent, target)
		if decodeErr != nil {
			err = fmt.Errorf("fns scan fn arguments failed, deocde json body failed")
			return
		}
	}
	// head and query
	if hasHead || hasQuery {
		targetValue := reflect.ValueOf(target)
		for _, field := range argFields {
			if field.sourceType == argumentsStructTagHead {
				headers := args.request.Header.Values(field.name)
				if headers == nil || len(headers) == 0 {
					continue
				}
				if field.fieldType.Kind() == reflect.String {
					targetValue.Field(field.fieldIdx).SetString(headers[0])
				} else if field.fieldType == reflect.TypeOf([]string{}) {
					targetValue.Field(field.fieldIdx).Set(reflect.ValueOf(headers))
				}
				continue
			}
			if field.sourceType == argumentsStructTagQuery {
				queryValues, hasValues := args.request.URL.Query()[field.name]
				if !hasValues || queryValues == nil || len(queryValues) == 0 {
					continue
				}
				setFieldValueErr := setFnHttpRequestArgumentsField(queryValues, field, targetValue)
				if setFieldValueErr != nil {
					err = setFieldValueErr
					return
				}
				continue
			}
		}
	}

	// path not supported

	return
}

func setFnHttpRequestArgumentsField(values []string, field fnArgumentsField, targetValue reflect.Value) (err error) {
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
	// time
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
	// unsupported
	err = fmt.Errorf("fns scan fn arguments failed, change to %s type failed, type %s is unsupported", field.name, field.fieldType.String())
	return
}
