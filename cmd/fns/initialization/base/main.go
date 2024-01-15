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
	"os"
	"path/filepath"
	"strings"
)

func NewMainFile(path string, dir string, useWork bool) (mf *MainFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fns: new main file failed").WithCause(err).WithMeta("dir", dir)
			return
		}
	}
	mf = &MainFile{
		path:     path,
		filename: filepath.ToSlash(filepath.Join(dir, "main.go")),
		useWork:  useWork,
	}
	return
}

type MainFile struct {
	path     string
	filename string
	useWork  bool
}

func (f *MainFile) Name() (name string) {
	name = f.filename
	return
}

func (f *MainFile) Write(_ context.Context) (err error) {
	const (
		content = `package main

import (
	"github.com/aacfactory/fns"
	"github.com/aacfactory/fns/context"
	"#path#/modules"
)

var (
	// Version
	// go build -ldflags "-s -w -X main.Version=${VERSION}" -o fapp
	Version = "v0.0.1"
)

//go:generate go run#mod_arg# #path#/internal/generator -v .
func main() {
	// set system environment to make config be active, e.g.: export FNS-ACTIVE=local
	fns.
		New(
			fns.Version(Version),
		).
		Deploy(modules.Services()...).
		Run(context.TODO()).
		Sync()
	return
}
`
	)
	modArg := ""
	if !f.useWork {
		modArg = " -mod=mod"
	}
	p := strings.ReplaceAll(content, "#mod_arg#", modArg)
	p = strings.ReplaceAll(p, "#path#", f.path)
	writeErr := os.WriteFile(f.filename, []byte(p), 0644)
	if writeErr != nil {
		err = errors.Warning("fns: main file write failed").WithCause(writeErr).WithMeta("filename", f.filename)
		return
	}
	return
}
