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

package commons

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"reflect"
	"regexp"
	"strings"
)

func ValidateFieldMessage(_type reflect.Type, exp string) (key string, msg string) {
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
		key, msg = ValidateFieldMessage(field.Type, exp[idx+1:])
		key = xk + "." + key
		return
	}
	key = xk
	msg = field.Tag.Get("message")
	return
}

func ValidateRegisterRegex(validate *validator.Validate) *validator.Validate {
	err := validate.RegisterValidation("regexp", func(fl validator.FieldLevel) (ok bool) {
		if fl.Field().Type().Kind() != reflect.String {
			return
		}
		exp := fl.Param()
		if exp == "" {
			return
		}
		value := fl.Field().String()
		ok, _ = regexp.MatchString(exp, value)
		return
	})
	if err != nil {
		panic(fmt.Errorf("fns: validate register regex failed, %v", err))
	}
	return validate
}

func ValidateRegisterNotBlank(validate *validator.Validate) *validator.Validate {
	err := validate.RegisterValidation("not_blank", func(fl validator.FieldLevel) (ok bool) {
		if fl.Field().Type().Kind() != reflect.String {
			return
		}
		ok = strings.TrimSpace(fl.Field().String()) != ""
		return
	})
	if err != nil {
		panic(fmt.Errorf("fns: validate register not_blank failed, %v", err))
	}
	return validate
}

func ValidateRegisterNotEmpty(validate *validator.Validate) *validator.Validate {
	err := validate.RegisterValidation("not_empty", func(fl validator.FieldLevel) (ok bool) {
		if fl.Field().Type().Kind() != reflect.Slice {
			return
		}
		ok = !fl.Field().IsNil() && fl.Field().Len() > 0
		return
	})
	if err != nil {
		panic(fmt.Errorf("fns: validate register not_empty failed, %v", err))
	}
	return validate
}

func ValidateRegisterDefault(validate *validator.Validate) *validator.Validate {
	err := validate.RegisterValidation("default", func(fl validator.FieldLevel) (ok bool) {
		if fl.Field().Type().Kind() != reflect.String {
			return
		}
		v := strings.TrimSpace(fl.Field().String())
		if v == "" {
			fl.Field().SetString(strings.TrimSpace(fl.Param()))
		}
		ok = true
		return
	})
	if err != nil {
		panic(fmt.Errorf("fns: validate register default failed, %v", err))
	}
	return validate
}
