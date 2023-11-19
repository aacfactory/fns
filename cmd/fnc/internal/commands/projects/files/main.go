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
	"os"
	"path/filepath"
	"strings"
)

func NewMainFile(path string, dir string) (mf *MainFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fnc: new main file failed").WithCause(err).WithMeta("dir", dir)
			return
		}
	}
	mf = &MainFile{
		path:     path,
		filename: filepath.ToSlash(filepath.Join(dir, "main.go")),
	}
	return
}

type MainFile struct {
	path     string
	filename string
}

func (mf *MainFile) Name() (name string) {
	name = mf.filename
	return
}

func (mf *MainFile) Write(_ context.Context) (err error) {
	const (
		content = `package main

import (
	"context"
	"fmt"
	"github.com/aacfactory/fns"
	"#path#/modules"
)

var (
	// Version
	// go build -ldflags "-X main.Version=${VERSION}" -o bin
	Version string = "v0.0.1"
)

//go:generate fnc codes .
func main() {
	// set system environment to make config be active, e.g.: export FNS-ACTIVE=local
	app := fns.New(
		fns.Version(Version),
	)
	// deploy services
	if err := app.Deploy(modules.Services()...); err != nil {
		if err != nil {
			app.Log().Error().Caller().Message(fmt.Sprintf("%+v", err))
			return
		}
	}
	// run
	if err := app.Run(context.Background()); err != nil {
		app.Log().Error().Caller().Message(fmt.Sprintf("%+v", err))
		return
	}
	if app.Log().DebugEnabled() {
		app.Log().Debug().Caller().Message("running...")
	}
	// sync signals
	if err := app.Sync(); err != nil {
		app.Log().Error().Caller().Message(fmt.Sprintf("%+v", err))
		return
	}
	if app.Log().DebugEnabled() {
		app.Log().Debug().Message("stopped!!!")
	}
	return
}
`
	)
	writeErr := os.WriteFile(mf.filename, []byte(strings.ReplaceAll(content, "#path#", mf.path)), 0644)
	if writeErr != nil {
		err = errors.Warning("fnc: main file write failed").WithCause(writeErr).WithMeta("filename", mf.filename)
		return
	}
	return
}
