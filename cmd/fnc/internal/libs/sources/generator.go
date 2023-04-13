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

package sources

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/fnc/internal/libs/processes"
	"path/filepath"
	"strings"
)

type GenerateOptions struct {
	Workspace string
}

type GenerateOption func(options *GenerateOptions) (err error)

func WithWorkspace(workspace string) GenerateOption {
	return func(options *GenerateOptions) (err error) {
		workspace = strings.TrimSpace(workspace)
		if workspace == "" {
			err = errors.Warning("sources: workspace option is invalid")
			return
		}
		options.Workspace = workspace
		return
	}
}

func Generate(ctx context.Context, dir string, options ...GenerateOption) (controller processes.ProcessController, err error) {
	opt := &GenerateOptions{
		Workspace: "",
	}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			optionErr := option(opt)
			if optionErr != nil {
				err = errors.Warning("sources: generate failed").WithCause(optionErr)
				return
			}
		}
	}
	if dir == "" {
		err = errors.Warning("sources: generate failed").WithCause(errors.Warning("project dir is nil"))
		return
	}
	// parse mod
	moduleFilename := filepath.Join(dir, "go.mod")
	var mod *Module
	if opt.Workspace != "" {
		mod, err = NewWithWork(moduleFilename, opt.Workspace)
	} else {
		mod, err = New(moduleFilename)
	}
	if err != nil {
		err = errors.Warning("sources: generate failed").WithCause(err)
		return
	}
	parseModErr := mod.Parse(ctx)
	if parseModErr != nil {
		err = errors.Warning("sources: generate failed").WithCause(parseModErr)
		return
	}
	// parse services
	services, parseServicesErr := mod.Services()
	if parseServicesErr != nil {
		err = errors.Warning("sources: generate failed").WithCause(parseServicesErr)
		return
	}
	// make process controller
	process := processes.New()
	if len(services) == 0 {
		controller = process
		return
	}

	functionParseUnits := make([]processes.Unit, 0, 1)
	serviceCodeFileUnits := make([]processes.Unit, 0, 1)
	for _, service := range services {
		for _, function := range service.Functions {
			functionParseUnits = append(functionParseUnits, function)
		}
		serviceCodeFileUnits = append(serviceCodeFileUnits, Unit(NewServiceFile(service)))
	}
	process.Add("services: parsing", functionParseUnits...)
	process.Add("services: writing", serviceCodeFileUnits...)
	process.Add("services: deploys", Unit(NewDeploysFile(filepath.ToSlash(filepath.Join(mod.Dir, "modules")), services)))
	controller = process
	return
}
