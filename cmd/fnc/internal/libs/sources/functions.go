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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"go/ast"
	"io"
	"reflect"
	"strings"
	"time"
)

type FunctionField struct {
	Name string
	Type *Type
}

func (sf *FunctionField) String() (v string) {
	v = fmt.Sprintf("%s %s", sf.Name, sf.Type.String())
	return
}

func (sf *FunctionField) Paths() (paths []string) {
	paths = sf.Type.GetTopPaths()
	return
}

type FunctionError struct {
	Name         string
	Descriptions map[string]string
}

type Function struct {
	mod             *Module
	hostServiceName string
	path            string
	filename        string
	file            *ast.File
	imports         Imports
	decl            *ast.FuncDecl
	Ident           string
	ConstIdent      string
	ProxyIdent      string
	Annotations     map[string]string
	Param           *FunctionField
	Result          *FunctionField
}

func (f *Function) HostServiceName() (name string) {
	name = f.hostServiceName
	return
}

func (f *Function) Name() (name string) {
	name = f.Annotations["fn"]
	return
}

func (f *Function) Internal() (ok bool) {
	_, ok = f.Annotations["internal"]
	return
}

func (f *Function) Title() (title string) {
	title = f.Annotations["title"]
	title = strings.TrimSpace(title)
	if title == "" {
		title = f.Name()
	}
	return
}

func (f *Function) Description() (description string) {
	description = f.Annotations["description"]
	return
}

func (f *Function) Errors() (errs []FunctionError) {
	errs = make([]FunctionError, 0, 1)
	p, has := f.Annotations["errors"]
	if !has {
		return
	}
	reader := bufio.NewReader(bytes.NewReader([]byte(p)))
	current := FunctionError{
		Name:         "",
		Descriptions: make(map[string]string),
	}
	for {
		line, _, readErr := reader.ReadLine()
		if readErr == io.EOF {
			break
		}
		px := bytes.IndexByte(line, '+')
		if px >= 0 {
			if current.Name != "" {
				errs = append(errs, current)
			}
			current = FunctionError{
				Name:         string(bytes.TrimSpace(line[px+1:])),
				Descriptions: make(map[string]string),
			}
			continue
		}
		dx := bytes.IndexByte(line, '-')
		if dx >= 0 {
			description := bytes.TrimSpace(line[dx+1:])
			idx := bytes.IndexByte(description, ':')
			if idx < 0 {
				continue
			}
			key := bytes.TrimSpace(description[0:idx])
			val := bytes.TrimSpace(description[idx+1:])
			current.Descriptions[string(key)] = string(val)
			continue
		}
	}
	if current.Name != "" {
		errs = append(errs, current)
	}
	return
}

func (f *Function) Validation() (title string, ok bool) {
	title, ok = f.Annotations["validation"]
	if !ok {
		return
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "invalid"
	}
	return
}

func (f *Function) Authorization() (ok bool) {
	_, ok = f.Annotations["authorization"]
	return
}

func (f *Function) Permission() (ok bool) {
	_, ok = f.Annotations["permission"]
	return
}

func (f *Function) Deprecated() (ok bool) {
	_, ok = f.Annotations["deprecated"]
	return
}

// todo barrier {scope: local | global}
func (f *Function) Barrier() (ok bool) {
	_, ok = f.Annotations["barrier"]
	return
}

func (f *Function) Timeout() (timeout time.Duration, has bool, err error) {
	s := ""
	s, has = f.Annotations["timeout"]
	if has {
		timeout, err = time.ParseDuration(s)
	}
	return
}

func (f *Function) SQL() (name string, has bool) {
	name, has = f.Annotations["sql"]
	return
}

func (f *Function) Transactional() (has bool) {
	_, has = f.Annotations["transactional"]
	return
}

func (f *Function) Cache() (ttl time.Duration, has bool, err error) {
	s := ""
	s, has = f.Annotations["cache"]
	if has {
		ttl, err = time.ParseDuration(s)
	}
	return
}

func (f *Function) FieldImports() (v Imports) {
	v = Imports{}
	paths := make([]string, 0, 1)
	if f.Param != nil {
		paths = append(paths, f.Param.Paths()...)
	}
	if f.Result != nil {
		paths = append(paths, f.Result.Paths()...)
	}
	for _, path := range paths {
		v.Add(&Import{
			Path:  path,
			Alias: "",
		})
	}
	return
}

func (f *Function) Parse(ctx context.Context) (err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: parse function failed").WithCause(ctx.Err()).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}
	if f.decl.Type.TypeParams != nil && f.decl.Type.TypeParams.List != nil && len(f.decl.Type.TypeParams.List) > 0 {
		err = errors.Warning("sources: parse function failed").WithCause(errors.Warning("function can not use paradigm")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}

	// params
	params := f.decl.Type.Params
	if params == nil || params.List == nil || len(params.List) == 0 || len(params.List) > 2 {
		err = errors.Warning("sources: parse function failed").WithCause(errors.Warning("params length must be one or two")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}
	if !f.mod.types.isContextType(params.List[0].Type, f.imports) {
		err = errors.Warning("sources: parse function failed").WithCause(errors.Warning("first param must be context.Context")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}
	if len(params.List) == 2 {
		param, parseParamErr := f.parseField(ctx, params.List[1])
		if parseParamErr != nil {
			err = errors.Warning("sources: parse function failed").WithCause(parseParamErr).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
			return
		}
		f.Param = param
	}
	// results
	results := f.decl.Type.Results
	if results == nil || results.List == nil || len(results.List) == 0 || len(results.List) > 2 {
		err = errors.Warning("sources: parse function failed").WithCause(errors.Warning("results length must be one or two")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}
	if len(results.List) == 1 {
		if !f.mod.types.isCodeErrorType(results.List[0].Type, f.imports) {
			err = errors.Warning("sources: parse function failed").WithCause(errors.Warning("the last results must github.com/aacfactory/errors.CodeError")).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
			return
		}
	} else {
		if !f.mod.types.isCodeErrorType(results.List[1].Type, f.imports) {
			err = errors.Warning("sources: parse function failed").WithCause(errors.Warning("the last results must github.com/aacfactory/errors.CodeError")).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
			return
		}
		result, parseResultErr := f.parseField(ctx, results.List[0])
		if parseResultErr != nil {
			err = errors.Warning("sources: parse function failed").WithCause(parseResultErr).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
			return
		}
		f.Result = result
	}
	return
}

func (f *Function) parseField(ctx context.Context, field *ast.Field) (v *FunctionField, err error) {
	if len(field.Names) != 1 {
		err = errors.Warning("sources: field must has only one name")
		return
	}
	name := field.Names[0].Name
	typ, parseTypeErr := f.parseFieldType(ctx, field.Type)
	if parseTypeErr != nil {
		err = errors.Warning("sources: parse field failed").WithMeta("field", name).WithCause(parseTypeErr)
		return
	}
	v = &FunctionField{
		Name: name,
		Type: typ,
	}
	return
}

func (f *Function) parseFieldType(ctx context.Context, e ast.Expr) (typ *Type, err error) {
	switch e.(type) {
	case *ast.Ident:
		typ, err = f.mod.types.parseExpr(ctx, e, &TypeScope{
			Path:       f.path,
			Mod:        f.mod,
			Imports:    f.imports,
			GenericDoc: "",
		})
		if err != nil {
			return
		}
		_, isBasic := typ.Basic()
		if isBasic {
			err = errors.Warning("sources: field type only support value object")
			return
		}
		typ.Path = f.path
		typ.Name = e.(*ast.Ident).Name
	case *ast.SelectorExpr:
		typ, err = f.mod.types.parseExpr(ctx, e, &TypeScope{
			Path:       f.path,
			Mod:        f.mod,
			Imports:    f.imports,
			GenericDoc: "",
		})
		if err != nil {
			return
		}
		_, isBasic := typ.Basic()
		if isBasic {
			err = errors.Warning("sources: field type only support value object")
			return
		}
		break
	default:
		err = errors.Warning("sources: field type only support no paradigms value object or array").WithMeta("expr", reflect.TypeOf(e).String())
		return
	}
	return
}

func (f *Function) Handle(ctx context.Context) (result interface{}, err error) {
	err = f.Parse(ctx)
	if err != nil {
		return
	}
	result = fmt.Sprintf("%s/%s: parse succeed", f.HostServiceName(), f.Name())
	return
}

type Functions []*Function

func (fns Functions) Len() int {
	return len(fns)
}

func (fns Functions) Less(i, j int) bool {
	return fns[i].Ident < fns[j].Ident
}

func (fns Functions) Swap(i, j int) {
	fns[i], fns[j] = fns[j], fns[i]
	return
}
