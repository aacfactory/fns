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
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/files"
	"golang.org/x/mod/modfile"
	"golang.org/x/sync/singleflight"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

func New(path string) (v *Module, err error) {
	v, err = NewWithWork(path, "")
	return
}

func NewWithWork(path string, workPath string) (v *Module, err error) {
	path = filepath.ToSlash(path)
	if !filepath.IsAbs(path) {
		absolute, absoluteErr := filepath.Abs(path)
		if absoluteErr != nil {
			err = errors.Warning("sources: new module failed").
				WithCause(errors.Warning("sources: get absolute representation of module file path failed").WithCause(absoluteErr).WithMeta("path", path))
			return
		}
		path = absolute
	}
	if !files.ExistFile(path) {
		err = errors.Warning("sources: new module failed").
			WithCause(errors.Warning("sources: file was not found").WithMeta("path", path))
		return
	}
	pkgErr := initPkgDir()
	if pkgErr != nil {
		err = errors.Warning("sources: new module failed").
			WithCause(pkgErr)
		return
	}
	if workPath != "" {
		work := &Work{
			Filename: workPath,
			Uses:     nil,
			Replaces: nil,
			parsed:   false,
		}
		parseWorkErr := work.Parse()
		if parseWorkErr != nil {
			err = errors.Warning("sources: new module failed").
				WithCause(parseWorkErr)
			return
		}
		dir := filepath.Dir(path)
		for _, use := range work.Uses {
			if use.Dir == dir {
				v = use
				break
			}
		}
		if v == nil {
			err = errors.Warning("sources: new module failed").
				WithCause(errors.Warning("can not find in workspace"))
			return
		}
	} else {
		v = &Module{
			Dir:      filepath.ToSlash(filepath.Dir(path)),
			Path:     "",
			Version:  "",
			Requires: nil,
			Work:     nil,
			Replace:  nil,
			locker:   &sync.Mutex{},
			parsed:   false,
			services: nil,
			types:    nil,
		}
	}
	return
}

type Module struct {
	Dir      string
	Path     string
	Version  string
	Requires Requires
	Work     *Work
	Replace  *Module
	locker   sync.Locker
	parsed   bool
	sources  *Sources
	services map[string]*Service
	types    *Types
}

func (mod *Module) Parse(ctx context.Context) (err error) {
	err = mod.parse(ctx, nil)
	return
}

func (mod *Module) parse(ctx context.Context, host *Module) (err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: parse mod failed").
			WithCause(ctx.Err())
		return
	}
	mod.locker.Lock()
	defer mod.locker.Unlock()
	if mod.parsed {
		return
	}
	if mod.Replace != nil {
		err = mod.Replace.parse(ctx, host)
		if err != nil {
			return
		}
		mod.parsed = true
		return
	}

	modFilepath := filepath.ToSlash(filepath.Join(mod.Dir, "go.mod"))
	if !files.ExistFile(modFilepath) {
		err = errors.Warning("sources: parse mod failed").
			WithCause(errors.Warning("sources: mod file was not found").
				WithMeta("file", modFilepath))
		return
	}
	modData, readModErr := os.ReadFile(modFilepath)
	if readModErr != nil {
		err = errors.Warning("sources: parse mod failed").
			WithCause(errors.Warning("sources: read mod file failed").
				WithCause(readModErr).
				WithMeta("file", modFilepath))
		return
	}
	mf, parseModErr := modfile.Parse(modFilepath, modData, nil)
	if parseModErr != nil {
		err = errors.Warning("sources: parse mod failed").
			WithCause(errors.Warning("sources: parse mod file failed").WithCause(parseModErr).WithMeta("file", modFilepath))
		return
	}
	mod.Path = mf.Module.Mod.Path
	mod.Version = mf.Module.Mod.Version
	mod.Requires = make([]*Module, 0, 1)
	if mf.Require != nil && len(mf.Require) > 0 {
		for _, require := range mf.Require {
			if mod.Work != nil {
				use, used := mod.Work.Use(require.Mod.Path)
				if used {
					mod.Requires = append(mod.Requires, use)
					continue
				}
			}
			requireDir := filepath.ToSlash(filepath.Join(PKG(), fmt.Sprintf("%s@%s", require.Mod.Path, require.Mod.Version)))

			mod.Requires = append(mod.Requires, &Module{
				Dir:      requireDir,
				Path:     require.Mod.Path,
				Version:  require.Mod.Version,
				Requires: nil,
				Work:     nil,
				Replace:  nil,
				locker:   &sync.Mutex{},
				parsed:   false,
				services: nil,
				types:    nil,
			})
		}
	}
	if mf.Replace != nil && len(mf.Replace) > 0 {
		for _, replace := range mf.Replace {
			replaceDir := ""
			if replace.New.Version != "" {
				replaceDir = filepath.Join(PKG(), fmt.Sprintf("%s@%s", replace.New.Path, replace.New.Version))
			} else {
				if filepath.IsAbs(replace.New.Path) {
					replaceDir = replace.New.Path
				} else {
					replaceDir = filepath.Join(mod.Dir, replace.New.Path)
				}
			}
			replaceDir = filepath.ToSlash(replaceDir)
			if !files.ExistFile(replaceDir) {
				err = errors.Warning("sources: parse mod failed").WithMeta("mod", mod.Path).
					WithCause(errors.Warning("sources: replace dir was not found").WithMeta("replace", replaceDir))
				return
			}
			replaceFile := filepath.ToSlash(filepath.Join(replaceDir, "go.mod"))
			if !files.ExistFile(replaceFile) {
				err = errors.Warning("sources: parse mod failed").WithMeta("mod", mod.Path).
					WithCause(errors.Warning("sources: replace mod file was not found").
						WithMeta("replace", replaceFile))
				return
			}
			replaceData, readReplaceErr := os.ReadFile(replaceFile)
			if readReplaceErr != nil {
				err = errors.Warning("sources: parse mod failed").WithMeta("mod", mod.Path).
					WithCause(errors.Warning("sources: read replace mod file failed").WithCause(readReplaceErr).WithMeta("replace", replaceFile))
				return
			}
			rmf, parseReplaceModErr := modfile.Parse(replaceFile, replaceData, nil)
			if parseReplaceModErr != nil {
				err = errors.Warning("sources: parse mod failed").WithMeta("mod", mod.Path).
					WithCause(errors.Warning("sources: parse replace mod file failed").WithCause(parseReplaceModErr).WithMeta("replace", replaceFile))
				return
			}
			for _, require := range mod.Requires {
				if require.Path == replace.Old.Path && require.Version == replace.Old.Version {
					require.Replace = &Module{
						Dir:      replaceDir,
						Path:     rmf.Module.Mod.Path,
						Version:  rmf.Module.Mod.Version,
						Requires: nil,
						Work:     nil,
						Replace:  nil,
						locker:   &sync.Mutex{},
						parsed:   false,
						services: nil,
						types:    nil,
					}
				}
			}
		}
	}
	work := mod.Work
	if host != nil && len(mod.Requires) > 0 {
		if host.Replace != nil {
			host = host.Replace
		}
		if host.Work != nil && work == nil {
			work = host.Work
		}
		if host.Requires != nil {
			for i, require := range mod.Requires {
				if require.Work != nil || require.Replace != nil {
					continue
				}
				for _, hr := range host.Requires {
					if require.Path == hr.Path {
						mod.Requires[i] = hr
						break
					}
				}
			}
		}
	}
	if work != nil && len(work.Replaces) > 0 && len(mod.Requires) > 0 {
		for i, require := range mod.Requires {
			if require.Work != nil || require.Replace != nil {
				continue
			}
			for _, replace := range work.Replaces {
				if require.Path == replace.Path {
					mod.Requires[i] = replace
					break
				}
			}
		}
	}
	if mod.Requires.Len() > 0 {
		sort.Sort(sort.Reverse(mod.Requires))
	}

	if host != nil {
		mod.types = host.types
	} else {
		mod.types = &Types{
			values: sync.Map{},
			group:  singleflight.Group{},
		}
	}

	mod.sources = newSource(mod.Path, mod.Dir)
	mod.parsed = true
	return
}

func (mod *Module) Services() (services Services, err error) {
	mod.locker.Lock()
	defer mod.locker.Unlock()
	if mod.services != nil {
		services = make([]*Service, 0, 1)
		for _, service := range mod.services {
			services = append(services, service)
		}
		sort.Sort(services)
		return
	}
	servicesDir := filepath.ToSlash(filepath.Join(mod.Dir, "modules"))
	entries, readServicesDirErr := os.ReadDir(servicesDir)
	if readServicesDirErr != nil {
		err = errors.Warning("read services dir failed").WithCause(readServicesDirErr).WithMeta("dir", servicesDir)
		return
	}
	if entries == nil || len(entries) == 0 {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.ToSlash(filepath.Join(mod.Path, "modules", entry.Name()))
		docFilename := filepath.ToSlash(filepath.Join(mod.Dir, "modules", entry.Name(), "doc.go"))
		if !files.ExistFile(docFilename) {
			continue
		}
		service, loaded, loadErr := tryLoadService(mod, path)
		if loadErr != nil {
			err = errors.Warning("load service failed").WithCause(loadErr).WithMeta("file", docFilename)
			return
		}
		if !loaded {
			continue
		}
		if mod.services == nil {
			mod.services = make(map[string]*Service)
		}
		_, exist := mod.services[service.Name]
		if exist {
			err = errors.Warning("load service failed").WithCause(errors.Warning("sources: services was duplicated")).WithMeta("service", service.Name)
			return
		}
		mod.services[service.Name] = service
	}
	services = make([]*Service, 0, 1)
	for _, service := range mod.services {
		services = append(services, service)
	}
	sort.Sort(services)
	return
}

func (mod *Module) findModuleByPath(ctx context.Context, path string) (v *Module, has bool, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: find module by path failed").
			WithCause(ctx.Err())
		return
	}
	if mod.Requires != nil {
		for _, require := range mod.Requires {
			if path == require.Path || strings.HasPrefix(path, require.Path+"/") {
				parseErr := require.parse(ctx, mod)
				if parseErr != nil {
					err = errors.Warning("sources: find module by path failed").
						WithCause(parseErr)
					return
				}
				if require.Replace != nil {
					require = require.Replace
				}
				v, has, err = require.findModuleByPath(ctx, path)
				if has || err != nil {
					return
				}
			}
		}
	}
	if path == mod.Path || strings.HasPrefix(path, mod.Path+"/") {
		if mod.Replace != nil {
			v = mod.Replace
		} else {
			v = mod
		}
		has = true
		return
	}
	return
}

func (mod *Module) ParseType(ctx context.Context, path string, name string) (typ *Type, err error) {
	// module
	typeModule, hasTypeModule, findTypeModuleErr := mod.findModuleByPath(ctx, path)
	if findTypeModuleErr != nil {
		err = errors.Warning("sources: mod parse type failed").
			WithMeta("path", path).WithMeta("name", name).
			WithCause(findTypeModuleErr)
		return
	}
	if !hasTypeModule {
		err = errors.Warning("sources: mod parse type failed").
			WithMeta("path", path).WithMeta("name", name).
			WithCause(errors.Warning("sources: module of type was not found"))
		return
	}
	// spec
	spec, specImports, genDoc, findSpecErr := typeModule.sources.FindTypeSpec(path, name)
	if findSpecErr != nil {
		err = errors.Warning("sources: mod parse type failed").
			WithMeta("path", path).WithMeta("name", name).
			WithCause(findSpecErr)
		return
	}

	typ, err = typeModule.types.parseType(ctx, spec, &TypeScope{
		Path:       path,
		Mod:        typeModule,
		Imports:    specImports,
		GenericDoc: genDoc,
	})
	return
}

func (mod *Module) GetType(path string, name string) (typ *Type, has bool) {
	key := formatTypeKey(path, name)
	value, exist := mod.types.values.Load(key)
	if !exist {
		return
	}
	typ, has = value.(*Type)
	return
}

func (mod *Module) Types() (types map[string]*Type) {
	types = make(map[string]*Type)
	mod.types.values.Range(func(key, value any) bool {
		typ, ok := value.(*Type)
		if !ok {
			return true
		}
		flats := typ.Flats()
		for k, t := range flats {
			_, exist := types[k]
			if !exist {
				types[k] = t
			}
		}
		return true
	})
	return
}

func (mod *Module) String() (s string) {
	buf := bytes.NewBuffer([]byte{})
	_, _ = buf.WriteString(fmt.Sprintf("path: %s\n", mod.Path))
	_, _ = buf.WriteString(fmt.Sprintf("version: %s\n", mod.Version))
	for _, require := range mod.Requires {
		_, _ = buf.WriteString(fmt.Sprintf("requre: %s@%s", require.Path, require.Version))
		if require.Replace != nil {
			_, _ = buf.WriteString(fmt.Sprintf("=> %s", require.Replace.Path))
			if require.Replace.Version != "" {
				_, _ = buf.WriteString(fmt.Sprintf("@%s", require.Replace.Version))
			}
		}
		_, _ = buf.WriteString("\n")
	}
	services, servicesErr := mod.Services()
	if servicesErr != nil {
		_, _ = buf.WriteString("service: load failed\n")
		_, _ = buf.WriteString(fmt.Sprintf("%+v", servicesErr))

	} else {
		for _, service := range services {
			_, _ = buf.WriteString(fmt.Sprintf("service: %s", service.Name))
			if len(service.Components) > 0 {
				_, _ = buf.WriteString("component: ")
				for i, component := range service.Components {
					if i == 0 {
						_, _ = buf.WriteString(fmt.Sprintf("%s", component.Indent))
					} else {
						_, _ = buf.WriteString(fmt.Sprintf(", %s", component.Indent))
					}
				}
			}
			_, _ = buf.WriteString("\n")
			if len(service.Functions) > 0 {
				for _, function := range service.Functions {
					_, _ = buf.WriteString(fmt.Sprintf("fn: %s\n", function.Name()))
				}
			}
		}
	}
	s = buf.String()
	return
}

type Requires []*Module

func (requires Requires) Len() int {
	return len(requires)
}

func (requires Requires) Less(i, j int) (ok bool) {
	on := strings.Split(requires[i].Path, "/")
	tn := strings.Split(requires[j].Path, "/")
	n := len(on)
	if len(on) > len(tn) {
		n = len(tn)
	}
	x := 0
	for x = 0; x < n; x++ {
		if on[x] != tn[x] {
			break
		}
	}
	if x < n {
		ok = on[x] > tn[x]
	} else {
		ok = len(on) < len(tn)
	}
	return
}

func (requires Requires) Swap(i, j int) {
	requires[i], requires[j] = requires[j], requires[i]
	return
}
