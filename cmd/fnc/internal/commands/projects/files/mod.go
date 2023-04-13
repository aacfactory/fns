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
	"github.com/aacfactory/fns/cmd/fnc/internal/libs/sources"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func NewModFile(path string, dir string) (mf *ModFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fnc: new mod file failed").WithCause(err).WithMeta("dir", dir)
			return
		}
	}
	mf = &ModFile{
		path:     path,
		filename: filepath.ToSlash(filepath.Join(dir, "go.mod")),
	}
	return
}

type ModFile struct {
	path     string
	filename string
}

func (mf *ModFile) Name() (name string) {
	name = mf.filename
	return
}

func (mf *ModFile) Write(ctx context.Context) (err error) {
	f := &modfile.File{}
	pathErr := f.AddModuleStmt(mf.path)
	if pathErr != nil {
		err = errors.Warning("fnc: mod file write failed").WithCause(pathErr).WithMeta("filename", mf.filename)
		return
	}
	goVersion := runtime.Version()[2:]
	goVersionItems := strings.Split(goVersion, ".")
	if len(goVersionItems) < 2 {
		err = errors.Warning("fnc: mod file write failed").WithCause(errors.Warning("invalid go runtime version").WithMeta("version", runtime.Version())).WithMeta("filename", mf.filename)
		return
	}
	versionErr := f.AddGoStmt(strings.Join(goVersionItems[0:2], "."))
	if versionErr != nil {
		err = errors.Warning("fnc: mod file write failed").WithCause(versionErr).WithMeta("filename", mf.filename)
		return
	}
	requires := []string{"github.com/aacfactory/fns"}
	for _, require := range requires {
		requireVersion, requireVersionErr := sources.LatestVersion(require)
		if requireVersionErr != nil {
			err = errors.Warning("fnc: mod file write failed").WithCause(requireVersionErr).WithMeta("filename", mf.filename)
			return
		}
		fnsRequireErr := f.AddRequire(require, requireVersion)
		if fnsRequireErr != nil {
			err = errors.Warning("fnc: mod file write failed").WithCause(fnsRequireErr).WithMeta("filename", mf.filename)
			return
		}
	}
	p, encodeErr := f.Format()
	if encodeErr != nil {
		err = errors.Warning("fnc: mod file write failed").WithCause(encodeErr).WithMeta("filename", mf.filename)
		return
	}
	writeErr := os.WriteFile(mf.filename, p, 0644)
	if writeErr != nil {
		err = errors.Warning("fnc: mod file write failed").WithCause(writeErr).WithMeta("filename", mf.filename)
		return
	}
	return
}
