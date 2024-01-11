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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/files"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func NewWorkFile(path string, dir string) (f *WorkFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fns: new work file failed").WithCause(err).WithMeta("dir", dir)
			return
		}
	}
	fmt.Println("work:", dir)
	fmt.Println("work:", filepath.ToSlash(filepath.Join(filepath.Dir(dir), "go.work")))

	f = &WorkFile{
		path:     path,
		modDir:   dir,
		filename: filepath.ToSlash(filepath.Join(filepath.Dir(dir), "go.work")),
	}
	return
}

type WorkFile struct {
	path     string
	modDir   string
	filename string
}

func (f *WorkFile) Name() (name string) {
	name = f.filename
	return
}

func (f *WorkFile) Write(ctx context.Context) (err error) {
	var p []byte
	if files.ExistFile(f.filename) {
		p, err = os.ReadFile(f.filename)
		if err != nil {
			err = errors.Warning("fns: work file write failed").WithCause(err).WithMeta("filename", f.filename)
			return
		}
	}
	wf, parseErr := modfile.ParseWork(f.filename, p, nil)
	if parseErr != nil {
		err = errors.Warning("fns: work file write failed").WithCause(parseErr).WithMeta("filename", f.filename)
		return
	}
	if wf.Go == nil {
		goVersion := runtime.Version()[2:]
		goVersionItems := strings.Split(goVersion, ".")
		if len(goVersionItems) < 2 {
			err = errors.Warning("fns: work file write failed").WithCause(errors.Warning("invalid go runtime version").WithMeta("version", runtime.Version())).WithMeta("filename", f.filename)
			return
		}
		versionErr := wf.AddGoStmt(strings.Join(goVersionItems[0:2], "."))
		if versionErr != nil {
			err = errors.Warning("fns: work file write failed").WithCause(versionErr).WithMeta("filename", f.filename)
			return
		}
	}
	wf.AddNewUse(f.modDir, f.path)

	b := modfile.Format(wf.Syntax)
	writeErr := os.WriteFile(f.filename, b, 0644)
	if writeErr != nil {
		err = errors.Warning("fns: work file write failed").WithCause(writeErr).WithMeta("filename", f.filename)
		return
	}
	return
}
