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

package services_test

import (
	"fmt"
	"github.com/aacfactory/fns/services"
	"testing"
	"time"
)

func TestValueOfResponse(t *testing.T) {
	now := time.Now()
	resp := services.NewResponse(now)
	v, err := services.ValueOfResponse[time.Time](resp)
	fmt.Println(v, err)
}
