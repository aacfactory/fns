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

package validators

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"reflect"
)

func validateRegisterNotEmpty(validate *validator.Validate) *validator.Validate {
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
