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

package modules

import (
	"context"
	"github.com/aacfactory/fns/cmd/generates/processes"
)

type CodeFileWriter interface {
	Name() (name string)
	Write(ctx context.Context) (err error)
}

type CodeFileUnit struct {
	cf CodeFileWriter
}

func (unit *CodeFileUnit) Handle(ctx context.Context) (result interface{}, err error) {
	err = unit.cf.Write(ctx)
	if err != nil {
		return
	}
	result = unit.cf.Name()
	return
}

func Unit(file CodeFileWriter) (unit processes.Unit) {
	return &CodeFileUnit{
		cf: file,
	}
}
