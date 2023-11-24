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
	"fmt"
	"github.com/aacfactory/cases"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/files"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Load(mod *sources.Module, dir string) (services Services, err error) {
	dir = filepath.ToSlash(filepath.Join(mod.Dir, dir))
	entries, readServicesDirErr := os.ReadDir(dir)
	if readServicesDirErr != nil {
		err = errors.Warning("read services dir failed").WithCause(readServicesDirErr).WithMeta("dir", dir)
		return
	}
	if entries == nil || len(entries) == 0 {
		return
	}
	group := make(map[string]*Service)
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
		_, exist := group[service.Name]
		if exist {
			err = errors.Warning("load service failed").WithCause(errors.Warning("modules: services was duplicated")).WithMeta("service", service.Name)
			return
		}
		group[service.Name] = service
	}
	services = make([]*Service, 0, 1)
	for _, service := range group {
		services = append(services, service)
	}
	sort.Sort(services)
	return
}

func tryLoadService(mod *sources.Module, path string) (service *Service, has bool, err error) {
	f, filename, readErr := mod.Sources().ReadFile(path, "doc.go")
	if readErr != nil {
		err = errors.Warning("modules: parse service failed").WithCause(readErr).WithMeta("path", path).WithMeta("file", "doc.go")
		return
	}
	_, pkg := filepath.Split(path)
	if pkg != f.Name.Name {
		err = errors.Warning("modules: parse service failed").WithCause(errors.Warning("pkg must be same as dir name")).WithMeta("path", path).WithMeta("file", "doc.go")
		return
	}

	doc := f.Doc.Text()
	if doc == "" {
		return
	}
	annotations, parseAnnotationsErr := sources.ParseAnnotations(doc)
	if parseAnnotationsErr != nil {
		err = errors.Warning("modules: parse service failed").WithCause(parseAnnotationsErr).WithMeta("path", path).WithMeta("file", "doc.go")
		return
	}

	name, hasName := annotations.Get("service")
	if !hasName {
		return
	}
	has = true
	title := ""
	description := ""
	internal := false
	titleAnno, hasTitle := annotations.Get("title")
	if hasTitle && len(titleAnno.Params) > 0 {
		title = titleAnno.Params[0]
	}
	descriptionAnno, hasDescription := annotations.Get("description")
	if hasDescription && len(descriptionAnno.Params) > 0 {
		description = descriptionAnno.Params[0]
	}
	internalAnno, hasInternal := annotations.Get("internal")
	if hasInternal && len(descriptionAnno.Params) > 0 {
		if len(internalAnno.Params) > 0 {
			internal = true
		}
		internal = internalAnno.Params[0] == "true"
	}

	service = &Service{
		mod:         mod,
		Dir:         filepath.Dir(filename),
		Path:        path,
		PathIdent:   f.Name.Name,
		Name:        strings.ToLower(name.Params[0]),
		Internal:    internal,
		Title:       title,
		Description: description,
		Imports:     sources.Imports{},
		Functions:   make([]*Function, 0, 1),
		Components:  make([]*Component, 0, 1),
	}
	loadFunctionsErr := service.loadFunctions()
	if loadFunctionsErr != nil {
		err = errors.Warning("modules: parse service failed").WithCause(loadFunctionsErr).WithMeta("path", path).WithMeta("file", "doc.go")
		return
	}
	loadComponentsErr := service.loadComponents()
	if loadComponentsErr != nil {
		err = errors.Warning("modules: parse service failed").WithCause(loadComponentsErr).WithMeta("path", path).WithMeta("file", "doc.go")
		return
	}
	sort.Sort(service.Functions)
	sort.Sort(service.Components)

	service.mergeImports()
	return
}

type Service struct {
	mod         *sources.Module
	Dir         string
	Path        string
	PathIdent   string
	Name        string
	Internal    bool
	Title       string
	Description string
	Imports     sources.Imports
	Functions   Functions
	Components  Components
}

func (service *Service) loadFunctions() (err error) {
	err = service.mod.Sources().ReadDir(service.Path, func(file *ast.File, filename string) (err error) {
		if file.Decls == nil || len(file.Decls) == 0 {
			return
		}
		fileImports := sources.NewImportsFromAstFileImports(file.Imports)
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if funcDecl.Recv != nil {
				continue
			}
			if funcDecl.Doc == nil {
				continue
			}
			doc := funcDecl.Doc.Text()
			if !strings.Contains(doc, "@fn") {
				continue
			}
			ident := funcDecl.Name.Name
			if ast.IsExported(ident) {
				err = errors.Warning("modules: parse func name failed").
					WithMeta("file", filename).
					WithMeta("func", ident).
					WithCause(errors.Warning("modules: func name must not be exported"))
				return
			}
			nameAtoms, parseNameErr := cases.LowerCamel().Parse(ident)
			if parseNameErr != nil {
				err = errors.Warning("modules: parse func name failed").
					WithMeta("file", filename).
					WithMeta("func", ident).
					WithCause(parseNameErr)
				return
			}
			proxyIdent := cases.Camel().Format(nameAtoms)
			constIdent := fmt.Sprintf("_%sFnName", ident)
			handlerIdent := fmt.Sprintf("_%s", ident)
			annotations, parseAnnotationsErr := sources.ParseAnnotations(doc)
			if parseAnnotationsErr != nil {
				err = errors.Warning("modules: parse func annotations failed").
					WithMeta("file", filename).
					WithMeta("func", ident).
					WithCause(parseAnnotationsErr)
				return
			}
			service.Functions = append(service.Functions, &Function{
				mod:             service.mod,
				hostServiceName: service.Name,
				path:            service.Path,
				filename:        filename,
				file:            file,
				imports:         fileImports,
				decl:            funcDecl,
				Ident:           ident,
				VarIdent:        constIdent,
				ProxyIdent:      proxyIdent,
				HandlerIdent:    handlerIdent,
				Annotations:     annotations,
				Param:           nil,
				Result:          nil,
			})
		}
		return
	})
	return
}

func (service *Service) loadComponents() (err error) {
	componentsPath := fmt.Sprintf("%s/components", service.Path)
	dir, dirErr := service.mod.Sources().DestinationPath(componentsPath)
	if dirErr != nil {
		err = errors.Warning("modules: read service components dir failed").WithCause(dirErr).WithMeta("service", service.Path)
		return
	}
	if !files.ExistFile(dir) {
		return
	}
	readErr := service.mod.Sources().ReadDir(componentsPath, func(file *ast.File, filename string) (err error) {
		if file.Decls == nil || len(file.Decls) == 0 {
			return
		}
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			specs := genDecl.Specs
			if specs == nil || len(specs) == 0 {
				continue
			}
			for _, spec := range specs {
				ts, tsOk := spec.(*ast.TypeSpec)
				if !tsOk {
					continue
				}
				doc := ""
				if ts.Doc == nil || ts.Doc.Text() == "" {
					if len(specs) == 1 && genDecl.Doc != nil && genDecl.Doc.Text() != "" {
						doc = genDecl.Doc.Text()
					}
				} else {
					doc = ts.Doc.Text()
				}
				if !strings.Contains(doc, "@component") {
					continue
				}
				ident := ts.Name.Name
				if !ast.IsExported(ident) {
					err = errors.Warning("modules: parse component name failed").
						WithMeta("file", filename).
						WithMeta("component", ident).
						WithCause(errors.Warning("modules: component name must be exported"))
					return
				}
				service.Components = append(service.Components, &Component{
					Indent: ident,
				})
			}
		}
		return
	})
	if readErr != nil {
		err = errors.Warning("modules: read service components dir failed").WithCause(readErr).WithMeta("service", service.Path)
		return
	}
	return
}

func (service *Service) mergeImports() {
	importer := sources.Imports{}
	importer.Add(&sources.Import{
		Path:  "github.com/aacfactory/fns/context",
		Alias: "",
	})
	importer.Add(&sources.Import{
		Path:  "github.com/aacfactory/errors",
		Alias: "",
	})
	importer.Add(&sources.Import{
		Path:  "github.com/aacfactory/fns/services",
		Alias: "",
	})
	importer.Add(&sources.Import{
		Path:  "github.com/aacfactory/fns/services/documents",
		Alias: "",
	})
	imports := make([]sources.Imports, 0, 1)
	imports = append(imports, importer)
	for _, function := range service.Functions {
		imports = append(imports, function.imports)
	}
	service.Imports = sources.MergeImports(imports)
	return
}

type Services []*Service

func (services Services) Len() int {
	return len(services)
}

func (services Services) Less(i, j int) bool {
	return services[i].Name < services[j].Name
}

func (services Services) Swap(i, j int) {
	services[i], services[j] = services[j], services[i]
	return
}
