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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/processes"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"github.com/charmbracelet/huh/spinner"
	"path/filepath"
	"time"
)

const (
	DefaultDir = "modules"
)

func NewGenerator(dir string, annotations FnAnnotationCodeWriters, verbose bool) *Generator {
	if dir == "" {
		dir = DefaultDir
	}
	return &Generator{
		dir:         dir,
		annotations: annotations,
		verbose:     verbose,
	}
}

type Generator struct {
	verbose     bool
	dir         string
	annotations FnAnnotationCodeWriters
}

func (generator *Generator) Generate(ctx context.Context, mod *sources.Module) (err error) {
	// parse services
	services, parseServicesErr := Load(mod, generator.dir)
	if parseServicesErr != nil {
		err = errors.Warning("modules: generate failed").WithCause(parseServicesErr)
		return
	}
	if len(services) == 0 {
		return
	}
	// make process controller
	process := processes.New()
	functionParseUnits := make([]processes.Unit, 0, 1)
	serviceCodeFileUnits := make([]processes.Unit, 0, 1)
	for _, service := range services {
		for _, function := range service.Functions {
			functionParseUnits = append(functionParseUnits, function)
		}
		serviceCodeFileUnits = append(serviceCodeFileUnits, Unit(NewServiceFile(service, generator.annotations)))
	}
	process.Add("generates: parsing", functionParseUnits...)
	process.Add("generates: writing", serviceCodeFileUnits...)
	process.Add("generates: deploys", Unit(NewDeploysFile(filepath.ToSlash(filepath.Join(mod.Dir, "modules")), services)))

	_ = spinner.New().Type(spinner.Dots).Title("Generating...").Action(func() {
		results := process.Start(ctx)
		for {
			result, ok := <-results
			if !ok {
				if generator.verbose {
					fmt.Println("generates: generate finished")
				}
				break
			}
			if generator.verbose {
				fmt.Println(result, "->", fmt.Sprintf("[%d/%d]", result.UnitNo, result.UnitNum), result.Data)
			}
			if result.Error != nil {
				err = result.Error
				_ = process.Abort(1 * time.Second)
				return
			}
		}
	}).Run()

	return
}
