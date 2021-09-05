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
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"time"
)

type HookUnit struct {
	Namespace    string
	FnName       string
	RequestId    string
	RequestUser  User
	RequestSize  int64
	ResponseSize int64
	Latency      time.Duration
	HandleError  errors.CodeError
}

type Hook interface {
	Build(config configuares.Config) (err error)
	Handle(unit HookUnit)
	Close()
}
