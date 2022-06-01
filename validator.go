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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
)

func defaultValidator() (v Validator) {
	validate := validator.New()
	validate = commons.ValidateRegisterRegex(validate)
	validate = commons.ValidateRegisterNotBlank(validate)
	validate = commons.ValidateRegisterNotEmpty(validate)
	validate = commons.ValidateRegisterDefault(validate)
	v = &argumentValidator{
		validate: validate,
	}
	return
}

type Validator interface {
	Validate(v interface{}) (err errors.CodeError)
}

type argumentValidator struct {
	validate *validator.Validate
}

func (av *argumentValidator) Validate(v interface{}) (err errors.CodeError) {
	validateErr := av.validate.Struct(v)
	if validateErr == nil {
		return
	}
	validationErrors, ok := validateErr.(validator.ValidationErrors)
	if !ok {
		err = errors.Warning(fmt.Sprintf("fns: validate argument failed")).WithCause(validateErr)
		return
	}
	err = errors.BadRequest("fns: argument is invalid")
	for _, validationError := range validationErrors {
		sf := validationError.Namespace()
		exp := sf[strings.Index(sf, ".")+1:]
		key, message := commons.ValidateFieldMessage(reflect.TypeOf(v), exp)
		if key == "" {
			err = errors.Warning(fmt.Sprintf("fns: validate argument failed for json tag of %s was not founed", sf))
			return
		}
		if message == "" {
			err = errors.Warning(fmt.Sprintf("fns: validate argument failed for message tag of %s was not founed", sf))
			return
		}
		err = err.WithMeta(key, message)
	}
	return
}
