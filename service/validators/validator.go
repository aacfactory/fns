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

package validators

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
)

type ValidateRegister func(validate *validator.Validate) *validator.Validate

func AddValidateRegister(register ValidateRegister) {
	_validator.validate = register(_validator.validate)
}

var _validator *validate = nil

func ValidateWithErrorTitle(value interface{}, title string) (err errors.CodeError) {
	err = _validator.Validate(value, title)
	return
}

func Validate(value interface{}) (err errors.CodeError) {
	err = ValidateWithErrorTitle(value, "invalid")
	return
}

type Validator interface {
	Validate(v interface{}) (err errors.CodeError)
}

type validate struct {
	validate *validator.Validate
}

func (v *validate) Validate(value interface{}, title string) (err errors.CodeError) {
	validateErr := v.validate.Struct(value)
	if validateErr == nil {
		return
	}
	validationErrors, ok := validateErr.(validator.ValidationErrors)
	if !ok {
		err = errors.Warning(fmt.Sprintf("fns: validate value failed")).WithCause(validateErr)
		return
	}
	err = errors.BadRequest(title)
	for _, validationError := range validationErrors {
		sf := validationError.Namespace()
		idx := strings.Index(sf, ".")
		if idx < 0 {
			continue
		}
		exp := sf[idx+1:]
		key, message := validateFieldMessage(reflect.TypeOf(value), exp)
		if key == "" {
			err = errors.Warning(fmt.Sprintf("fns: validate value failed for json tag of %s was not founed", sf))
			return
		}
		if message == "" {
			err = errors.Warning(fmt.Sprintf("fns: validate value failed for message tag of %s was not founed", sf))
			return
		}
		err = err.WithMeta(key, message)
	}
	return
}

func validateFieldMessage(_type reflect.Type, exp string) (key string, msg string) {
	if _type.Kind() == reflect.Ptr {
		_type = _type.Elem()
	}
	fieldName := ""
	idx := strings.Index(exp, ".")
	if idx > 0 {
		fieldName = exp[0:idx]
	} else {
		fieldName = exp
	}
	field, has := _type.FieldByName(fieldName)
	if !has {
		return
	}
	xk := field.Tag.Get("json")
	if pos := strings.Index(xk, ","); pos > 0 {
		xk = xk[0:pos]
	}

	if idx > 0 {
		key, msg = validateFieldMessage(field.Type, exp[idx+1:])
		key = xk + "." + key
		return
	}
	key = xk
	msg = field.Tag.Get("message")
	return
}
