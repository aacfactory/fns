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

package sources

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/files"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
	"sync"
)

type Work struct {
	Filename string
	Uses     []*Module
	Replaces []*Module
	parsed   bool
}

func (work *Work) Use(path string) (v *Module, used bool) {
	for _, use := range work.Uses {
		if use.Path == path {
			v = use
			used = true
			break
		}
	}
	return
}

func (work *Work) Parse() (err error) {
	if work.parsed {
		return
	}
	path := work.Filename
	if !filepath.IsAbs(path) {
		absolute, absoluteErr := filepath.Abs(path)
		if absoluteErr != nil {
			err = errors.Warning("sources: parse work failed").
				WithCause(errors.Warning("sources: get absolute representation of work file failed").WithCause(absoluteErr).WithMeta("work", path))
			return
		}
		path = absolute
	}
	if !files.ExistFile(path) {
		err = errors.Warning("sources: parse work failed").
			WithCause(errors.Warning("sources: file was not found").WithMeta("work", path))
		return
	}
	dir := filepath.Dir(path)
	path = filepath.ToSlash(path)
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		err = errors.Warning("sources: parse work failed").WithMeta("work", path).WithCause(readErr)
		return
	}
	file, parseErr := modfile.ParseWork(path, data, nil)
	if parseErr != nil {
		err = errors.Warning("sources: parse work failed").WithMeta("work", path).WithCause(parseErr)
		return
	}
	work.Filename = path
	work.Uses = make([]*Module, 0, 1)
	work.Replaces = make([]*Module, 0, 1)
	if file.Use != nil && len(file.Use) > 0 {
		for _, use := range file.Use {
			usePath := use.Path
			if filepath.IsAbs(usePath) {
				usePath = filepath.ToSlash(usePath)
			} else {
				usePath = filepath.ToSlash(filepath.Join(dir, usePath))
			}
			moduleFile := filepath.ToSlash(filepath.Join(usePath, "go.mod"))
			if !files.ExistFile(moduleFile) {
				err = errors.Warning("sources: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("sources: mod file was not found").
						WithMeta("mod", moduleFile))
				return
			}
			modData, readModErr := os.ReadFile(moduleFile)
			if readModErr != nil {
				err = errors.Warning("sources: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("sources: read mod file failed").WithCause(readModErr).WithMeta("mod", moduleFile))
				return
			}
			mf, parseModErr := modfile.Parse(moduleFile, modData, nil)
			if parseModErr != nil {
				err = errors.Warning("sources: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("sources: parse mod file failed").WithCause(parseModErr).WithMeta("mod", moduleFile))
				return
			}
			mod := &Module{
				Dir:          usePath,
				Path:         mf.Module.Mod.Path,
				Version:      "",
				Requires:     nil,
				Work:         work,
				Replace:      nil,
				locker:       &sync.Mutex{},
				parsed:       false,
				sources:      nil,
				builtinTypes: nil,
				types:        nil,
			}
			registerBuiltinTypes(mod)
			work.Uses = append(work.Uses, mod)
		}
	}
	if file.Replace != nil && len(file.Replace) > 0 {
		for _, replace := range file.Replace {
			replaceDir := ""
			if replace.New.Version != "" {
				replaceDir = filepath.Join(PKG(), fmt.Sprintf("%s@%s", replace.New.Path, replace.New.Version))
			} else {
				replaceDir = filepath.Join(PKG(), replace.New.Path)
			}
			replaceDir = filepath.ToSlash(replaceDir)
			if !files.ExistFile(replaceDir) {
				err = errors.Warning("sources: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("sources: replace dir was not found").WithMeta("replace", replaceDir))
				return
			}
			moduleFile := filepath.ToSlash(filepath.Join(replaceDir, "mod.go"))
			if !files.ExistFile(moduleFile) {
				err = errors.Warning("sources: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("sources: replace mod file was not found").
						WithMeta("mod", moduleFile))
				return
			}
			modData, readModErr := os.ReadFile(moduleFile)
			if readModErr != nil {
				err = errors.Warning("sources: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("sources: read replace mod file failed").WithCause(readModErr).WithMeta("mod", moduleFile))
				return
			}
			mf, parseModErr := modfile.Parse(moduleFile, modData, nil)
			if parseModErr != nil {
				err = errors.Warning("sources: parse work failed").WithMeta("work", path).
					WithCause(errors.Warning("sources: parse replace mod file failed").WithCause(parseModErr).WithMeta("mod", moduleFile))
				return
			}
			work.Replaces = append(work.Replaces, &Module{
				Dir:      "",
				Path:     replace.Old.Path,
				Version:  replace.Old.Version,
				Requires: nil,
				Work:     nil,
				Replace: &Module{
					Dir:      replaceDir,
					Path:     mf.Module.Mod.Path,
					Version:  mf.Module.Mod.Version,
					Requires: nil,
					Work:     nil,
					Replace:  nil,
					locker:   &sync.Mutex{},
					parsed:   false,
					types:    nil,
				},
				locker:       &sync.Mutex{},
				parsed:       false,
				sources:      nil,
				builtinTypes: map[string]*Type{},
				types:        nil,
			})
		}
	}
	work.parsed = true
	return
}
