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

package files

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/fnc/internal/libs/files"
	"os"
	"path/filepath"
	"strings"
)

func NewModulesFile(path string, dir string) (mf *ModulesFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fnc: new modules file failed").WithCause(err).WithMeta("dir", dir)
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

func (mf *ModulesFile) Name() (name string) {
	name = mf.dir
	return
}

func (mf *ModulesFile) Write(ctx context.Context) (err error) {
	if !files.ExistFile(mf.dir) {
		mdErr := os.MkdirAll(mf.dir, 0644)
		if mdErr != nil {
			err = errors.Warning("fnc: modules file write failed").WithCause(mdErr).WithMeta("dir", mf.dir)
			return
		}
	}
	err = mf.writeServices(ctx)
	if err != nil {
		return
	}
	err = mf.writeExamples(ctx)
	if err != nil {
		return
	}
	return
}

func (mf *ModulesFile) writeServices(ctx context.Context) (err error) {
	// services
	const (
		exports = `package modules

import (
	"github.com/aacfactory/fns/service"
)

func Services() (v []service.Service) {
	v = append(
		dependencies(),
		services()...,
	)
	return
}

func dependencies() (v []service.Service) {
	v = []service.Service{
		// add dependencies here
	}
	return
}`
	)
	servicesFilename := filepath.ToSlash(filepath.Join(mf.dir, "services.go"))
	writeErr := os.WriteFile(servicesFilename, []byte(exports), 0644)
	if writeErr != nil {
		err = errors.Warning("fnc: modules file write failed").WithCause(writeErr).WithMeta("filename", servicesFilename)
		return
	}
	// fns
	const (
		fns = `// NOTE: this file has been automatically generated, DON'T EDIT IT!!!

package modules

import (
	"#path#/modules/examples"
	"github.com/aacfactory/fns/service"
)

func services() (v []service.Service) {
	v = []service.Service{
		examples.Service(),
	}
	return
}
`
	)
	fnsFilename := filepath.ToSlash(filepath.Join(mf.dir, "fns.go"))
	writeErr = os.WriteFile(fnsFilename, []byte(strings.ReplaceAll(fns, "#path#", mf.path)), 0644)
	if writeErr != nil {
		err = errors.Warning("fnc: modules file write failed").WithCause(writeErr).WithMeta("filename", servicesFilename)
		return
	}
	return
}

func (mf *ModulesFile) writeExamples(ctx context.Context) (err error) {
	dir := filepath.ToSlash(filepath.Join(mf.dir, "examples"))
	if !files.ExistFile(dir) {
		mdErr := os.MkdirAll(dir, 0600)
		if mdErr != nil {
			err = errors.Warning("fnc: new modules file failed").WithCause(mdErr).WithMeta("dir", dir)
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
		err = errors.Warning("fnc: modules file write failed").WithCause(err).WithMeta("filename", filepath.ToSlash(filepath.Join(dir, "doc.go")))
		return
	}
	// hello
	const (
		hello = `package examples

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
)

// HelloArgument
// @title Hello function argument
// @description Hello function argument
type HelloArgument struct {
	// World
	// @title Name
	// @description Name
	// @validate-message-i18n >>>
	// zh: 世界是必须的
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
// @timeout 1s
// @barrier
// @title Hello
// @errors >>>
// + examples_hello_failed
// 	- zh: 错误
//	- en: failed
// <<<
// @description >>>
// Hello
// <<<
func hello(ctx context.Context, argument HelloArgument) (result HelloResults, err errors.CodeError) {
	if argument.World == "error" {
		err = errors.ServiceError("examples_hello_failed")
		return
	}
	result = HelloResults{fmt.Sprintf("hello %s!", argument.World)}
	return
}
`
	)

	err = os.WriteFile(filepath.ToSlash(filepath.Join(dir, "hello.go")), []byte(doc), 0644)
	if err != nil {
		err = errors.Warning("fnc: modules file write failed").WithCause(err).WithMeta("filename", filepath.ToSlash(filepath.Join(dir, "hello.go")))
		return
	}
	// fns
	const (
		fns = `// NOTE: this file has been automatically generated, DON'T EDIT IT!!!

package examples

import (
	"context"
	"time"

	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/fns/service/documents"
)

const (
	_name    = "examples"
	_helloFn = "hello"
)

func Hello(ctx context.Context, argument HelloArgument) (result HelloArgument, err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, _name)
	if !hasEndpoint {
		err = errors.Warning("examples: endpoint was not found").WithMeta("name", _name)
		return
	}
	fr, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, _name, _helloFn, service.NewArgument(argument)))
	if requestErr != nil {
		err = requestErr
		return
	}
	if !fr.Exist() {
		return
	}
	scanErr := fr.Scan(&result)
	if scanErr != nil {
		err = errors.Warning("examples: scan future result failed").
			WithMeta("service", _name).WithMeta("fn", _helloFn).
			WithCause(scanErr)
		return
	}
	return
}

func Service() (v service.Service) {
	v = &_service_{
		Abstract: service.NewAbstract(
			_name,
			false,
		),
	}
	return
}

type _service_ struct {
	service.Abstract
}

func (svc *_service_) Handle(ctx context.Context, fn string, argument service.Param) (v interface{}, err errors.CodeError) {
	switch fn {
	case _helloFn:
		// param
		param := HelloArgument{}
		paramErr := argument.As(&param)
		if paramErr != nil {
			err = errors.Warning("examples: decode request argument failed").WithCause(paramErr)
			break
		}
		// make timeout context
		var cancel context.CancelFunc = nil
		ctx, cancel = context.WithTimeout(ctx, time.Duration(1000000000))
		// barrier
		v, err = svc.Barrier(ctx, _helloFn, argument, func() (v interface{}, err errors.CodeError) {
			// execute function
			v, err = hello(ctx, param)
			return
		})
		// cancel timeout context
		cancel()
		break
	default:
		err = errors.Warning("examples: fn was not found").WithMeta("service", _name).WithMeta("fn", fn)
		break
	}
	return
}

func (svc *_service_) Document() (doc *documents.Document) {
	doc = documents.New(_name, "Example service", svc.AppVersion())
	// hello
	doc.AddFn(
		"hello", "Hello", "Hello", false, false,
		documents.Struct("#path#/modules/examples", "HelloArgument").
			SetTitle("Hello function argument").
			SetDescription("Hello function argument").
			AddProperty(
				"world",
				documents.String().
					SetTitle("Name").
					SetDescription("Name").
					AsRequired().
					SetValidation(documents.NewElementValidation("world_required", "zh", "世界是必须的", "en", "world is required")),
			),
		documents.Array(documents.String()).
			SetPath("#path#/modules/examples").
			SetName("HelloResults").
			SetTitle("Hello Results").
			SetDescription("Hello Results"),
		[]documents.FnError{
			{
				Name_: "examples_hello_failed",
				Descriptions_: map[string]string{
					"zh": "错误",
					"en": "failed",
				},
			},
		},
	)
	return
}
`
	)
	err = os.WriteFile(filepath.ToSlash(filepath.Join(dir, "fns.go")), []byte(strings.ReplaceAll(fns, "#path#", mf.path)), 0644)
	if err != nil {
		err = errors.Warning("fnc: modules file write failed").WithCause(err).WithMeta("filename", filepath.ToSlash(filepath.Join(dir, "fns.go")))
		return
	}
	return
}
