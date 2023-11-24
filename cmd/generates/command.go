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

package generates

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/modules"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"github.com/urfave/cli/v2"
	"path/filepath"
	"runtime"
	"strings"
)

func WithName(name string) Option {
	return func(options *Options) {
		options.name = name
	}
}

func WithModulesDir(dir string) Option {
	return func(options *Options) {
		options.modulesDir = dir
	}
}

func WithAnnotations(annotations ...modules.FnAnnotationCodeWriter) Option {
	return func(options *Options) {
		if options.annotations == nil {
			options.annotations = make([]modules.FnAnnotationCodeWriter, 0, 1)
		}
		options.annotations = append(options.annotations, annotations...)
	}
}

func WithBuiltinTypes(builtinTypes ...*sources.Type) Option {
	return func(options *Options) {
		if options.builtinTypes == nil {
			options.builtinTypes = make([]*sources.Type, 0, 1)
		}
		options.builtinTypes = append(options.builtinTypes, builtinTypes...)
	}
}

func WithGenerator(generator Generator) Option {
	return func(options *Options) {
		if options.generators == nil {
			options.generators = make([]Generator, 0, 1)
		}
		options.generators = append(options.generators, generator)
	}
}

type Option func(options *Options)

type Options struct {
	name         string
	modulesDir   string
	annotations  []modules.FnAnnotationCodeWriter
	builtinTypes []*sources.Type
	generators   []Generator
}

func New(options ...Option) (cmd Command) {
	opt := Options{}
	for _, option := range options {
		option(&opt)
	}
	name := opt.name
	if name == "" {
		name = callerPKG()
	}
	act := &action{
		modulesDir:   opt.modulesDir,
		annotations:  opt.annotations,
		builtinTypes: opt.builtinTypes,
		generators:   opt.generators,
	}
	// app
	app := cli.NewApp()
	app.Name = name
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:     "verbose",
			EnvVars:  []string{"FNS_VERBOSE"},
			Aliases:  []string{"v"},
			Usage:    "verbose output",
			Required: false,
		},
		&cli.StringFlag{
			Name:      "work",
			Aliases:   []string{"w"},
			Usage:     "set workspace file path",
			Required:  false,
			EnvVars:   []string{"FNS_WORK"},
			TakesFile: false,
		},
	}
	app.Usage = fmt.Sprintf("%s {project path}", name)
	app.Action = act.Handle

	// cmd
	cmd = &cliCommand{
		app: app,
	}
	return
}

type Command interface {
	Execute(ctx context.Context, args ...string) (err error)
}

type cliCommand struct {
	app *cli.App
}

func (c *cliCommand) Execute(ctx context.Context, args ...string) (err error) {
	err = c.app.RunContext(ctx, args)
	return
}

type action struct {
	modulesDir   string
	annotations  []modules.FnAnnotationCodeWriter
	builtinTypes []*sources.Type
	generators   []Generator
}

func (act *action) Handle(c *cli.Context) (err error) {
	// verbose
	verbose := c.Bool("verbose")
	// project dir
	projectDir := strings.TrimSpace(c.Args().First())
	if projectDir == "" {
		projectDir = "."
	}
	if !filepath.IsAbs(projectDir) {
		projectDir, err = filepath.Abs(projectDir)
		if err != nil {
			err = errors.Warning("generates: generate failed").WithCause(err).WithMeta("dir", projectDir)
			return
		}
	}
	projectDir = filepath.ToSlash(projectDir)
	work := c.String("work")
	if work != "" {
		work = strings.TrimSpace(work)
		if work == "" {
			err = errors.Warning("generates: generate failed").WithCause(errors.Warning("work option is invalid"))
			return
		}
	}
	ctx := c.Context
	// parse mod
	moduleFilename := filepath.Join(projectDir, "go.mod")
	var mod *sources.Module
	if work != "" {
		mod, err = sources.NewWithWork(moduleFilename, work)
	} else {
		mod, err = sources.New(moduleFilename)
	}
	if err != nil {
		err = errors.Warning("generates: generate failed").WithCause(err)
		return
	}
	parseModErr := mod.Parse(ctx)
	if parseModErr != nil {
		err = errors.Warning("generates: generate failed").WithCause(parseModErr)
		return
	}
	if len(act.builtinTypes) > 0 {
		for _, builtinType := range act.builtinTypes {
			mod.RegisterBuiltinType(builtinType)
		}
	}
	// services
	services := modules.NewGenerator(act.modulesDir, act.annotations, verbose)
	servicesErr := services.Generate(ctx, mod)
	if servicesErr != nil {
		err = errors.Warning("generates: generate failed").WithCause(servicesErr)
		return
	}
	// extras
	for _, generator := range act.generators {
		err = generator.Generate(ctx, mod)
		if err != nil {
			err = errors.Warning("generates: generate failed").WithCause(err)
			return
		}
	}
	return
}

func callerPKG() string {
	_, file, _, ok := runtime.Caller(2)
	if ok {
		dir := filepath.Dir(file)
		if dir == "" {
			return "fns"
		}
		return filepath.Base(dir)
	}
	return "fns"
}
