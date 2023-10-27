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

package services

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
)

type Config map[string]json.RawMessage

func (config Config) Get(name string) (v configures.Config, err error) {
	p, exist := config[name]
	if !exist || len(p) == 0 {
		v, _ = configures.NewJsonConfig([]byte{'{', '}'})
		return
	}
	v, err = configures.NewJsonConfig(p)
	if err != nil {
		err = errors.Warning("fns: get service config failed").WithMeta("name", name).WithCause(err)
		return
	}
	return
}
