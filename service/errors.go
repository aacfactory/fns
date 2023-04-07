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

package service

import (
	"github.com/aacfactory/errors"
	"net/http"
)

var (
	ErrServiceOverload = errors.Unavailable("fns: service is overload").WithMeta("fns", "overload")
	ErrTooEarly        = errors.New(http.StatusTooEarly, "***TOO EARLY***", "fns: service is not ready, try later again")
	ErrUnavailable     = errors.Unavailable("fns: service is closed")
	ErrDeviceId        = errors.Warning("fns: device id was required")
	ErrNotFound        = errors.NotFound("fns: no handlers accept request")
)