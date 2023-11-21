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
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"go/ast"
	"reflect"
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
	Annotations     Annotations
	Param           *FunctionField
	Result          *FunctionField
}

func (f *Function) HostServiceName() (name string) {
	name = f.hostServiceName
	return
}

func (f *Function) Name() (name string) {
	anno, _ := f.Annotations.Get("fn")
	name = anno.Params[0]
	return
}

func (f *Function) Readonly() (ok bool) {
	_, ok = f.Annotations.Get("readonly")
	return
}

func (f *Function) Internal() (ok bool) {
	_, ok = f.Annotations.Get("internal")
	return
}

func (f *Function) Title() (title string) {
	anno, has := f.Annotations.Get("title")
	if has {
		if len(anno.Params) > 0 {
			title = anno.Params[0]
		} else {
			title = f.Name()
		}
	} else {
		title = f.Name()
	}
	return
}

func (f *Function) Description() (description string) {
	anno, has := f.Annotations.Get("description")
	if has {
		if len(anno.Params) > 0 {
			description = anno.Params[0]
		}
	}
	return
}

func (f *Function) Errors() (errs string) {
	anno, has := f.Annotations.Get("errors")
	if !has {
		return
	}
	if len(anno.Params) == 0 {
		return
	}
	errs = anno.Params[0]
	return
}

func (f *Function) Validation() (title string, ok bool) {
	if f.Param == nil {
		return
	}
	anno, has := f.Annotations.Get("validation")
	if !has {
		return
	}
	if len(anno.Params) == 0 {
		title = "invalid"
		ok = true
		return
	}
	title = anno.Params[0]
	ok = true
	return
}

func (f *Function) Authorization() (ok bool) {
	_, ok = f.Annotations.Get("authorization")
	return
}

func (f *Function) Permission() (ok bool) {
	_, ok = f.Annotations.Get("permission")
	return
}

func (f *Function) Deprecated() (ok bool) {
	_, ok = f.Annotations.Get("deprecated")
	return
}

func (f *Function) Barrier() (ok bool) {
	_, ok = f.Annotations.Get("barrier")
	return
}

func (f *Function) Cache() (params []string, has bool) {
	anno, exist := f.Annotations.Get("cache")
	if !exist {
		return
	}
	if len(anno.Params) == 0 {
		return
	}
	params = anno.Params
	has = true
	return
}

func (f *Function) Annotation(name string) (params []string, has bool) {
	anno, exist := f.Annotations.Get(name)
	if exist {
		params = anno.Params
		has = true
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