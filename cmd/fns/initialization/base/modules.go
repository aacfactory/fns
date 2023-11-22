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

package base

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/files"
	"os"
	"path/filepath"
	"strings"
)

func NewModulesFile(path string, dir string) (mf *ModulesFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fns: new modules file failed").WithCause(err).WithMeta("dir", dir)
			return
		}
	}
	dir = filepath.ToSlash(filepath.Join(dir, "modules"))
	mf = &ModulesFile{
		path: path,
		dir:  dir,
	}
	return
}

type ModulesFile struct {
	path string
	dir  string
}

func (f *ModulesFile) Name() (name string) {
	name = f.dir
	return
}

func (f *ModulesFile) Write(ctx context.Context) (err error) {
	if !files.ExistFile(f.dir) {
		mdErr := os.MkdirAll(f.dir, 0644)
		if mdErr != nil {
			err = errors.Warning("fns: modules file write failed").WithCause(mdErr).WithMeta("dir", f.dir)
			return
		}
	}
	err = f.writeServices(ctx)
	if err != nil {
		return
	}
	err = f.writeExamples(ctx)
	if err != nil {
		return
	}
	return
}

func (f *ModulesFile) writeServices(_ context.Context) (err error) {
	// services
	const (
		exports = `package modules

import (
	"github.com/aacfactory/fns/services"
)

func Services() (v []services.Service) {
	v = append(
		dependencies(),
		endpoints()...,
	)
	return
}

func dependencies() (v []services.Service) {
	v = []services.Service{
		// add dependencies here
	}
	return
}`
	)
	servicesFilename := filepath.ToSlash(filepath.Join(f.dir, "services.go"))
	writeErr := os.WriteFile(servicesFilename, []byte(exports), 0644)
	if writeErr != nil {
		err = errors.Warning("fns: modules file write failed").WithCause(writeErr).WithMeta("filename", servicesFilename)
		return
	}
	// fns
	const (
		fns = `// NOTE: this file has been automatically generated, DON'T EDIT IT!!!

package modules

import (
	"#path#/modules/examples"
	"github.com/aacfactory/fns/services"
)

func endpoints() (v []services.Service) {
	v = []services.Service{
		
	}
	return
}
`
	)
	fnsFilename := filepath.ToSlash(filepath.Join(f.dir, "fns.go"))
	writeErr = os.WriteFile(fnsFilename, []byte(strings.ReplaceAll(fns, "#path#", f.path)), 0644)
	if writeErr != nil {
		err = errors.Warning("fns: modules file write failed").WithCause(writeErr).WithMeta("filename", servicesFilename)
		return
	}
	return
}

func (f *ModulesFile) writeExamples(_ context.Context) (err error) {
	dir := filepath.ToSlash(filepath.Join(f.dir, "examples"))
	if !files.ExistFile(dir) {
		mdErr := os.MkdirAll(dir, 0600)
		if mdErr != nil {
			err = errors.Warning("fns: new modules file failed").WithCause(mdErr).WithMeta("dir", dir)
			return
		}
	}
	// doc
	const (
		doc = `// Package examples
// @service examples
// @title Examples
// @description Example service
package examples`
	)
	err = os.WriteFile(filepath.ToSlash(filepath.Join(dir, "doc.go")), []byte(doc), 0644)
	if err != nil {
		err = errors.Warning("fns: modules file write failed").WithCause(err).WithMeta("filename", filepath.ToSlash(filepath.Join(dir, "doc.go")))
		return
	}
	// hello
	const (
		hello = `package examples

import (
	"github.com/aacfactory/fns/context"
	"fmt"
	"github.com/aacfactory/errors"
)

// HelloParam
// @title Hello function param
// @description Hello function param
type HelloParam struct {
	// World
	// @title Name
	// @description Name
	// @validate-message-i18n >>>
	// zh: 世界是必要的
	// en: world is required
	// <<<
	World string ` + "`" + `json:"world" validate:"required" validate-message:"world_required"` + "`" + `
}

// HelloResults
// @title Hello Results
// @description Hello Results
type HelloResults []string

// hello
// @fn hello
// @readonly
// @barrier
// @errors >>>
// examples_hello_failed
// zh: 错误
// en: failed
// <<<
// @title Hello
// @description >>>
// Hello
// <<<
func hello(ctx context.Context, param HelloParam) (result HelloResults, err error) {
	if param.World == "error" {
		err = errors.ServiceError("examples_hello_failed")
		return
	}
	result = HelloResults{fmt.Sprintf("hello %s!", param.World)}
	return
}
`
	)

	err = os.WriteFile(filepath.ToSlash(filepath.Join(dir, "hello.go")), []byte(hello), 0644)
	if err != nil {
		err = errors.Warning("fns: modules file write failed").WithCause(err).WithMeta("filename", filepath.ToSlash(filepath.Join(dir, "hello.go")))
		return
	}

	return
}
