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
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"go/ast"
	"reflect"
	"strconv"
	"strings"
)

type FunctionField struct {
	mod  *sources.Module
	Name string
	Type *sources.Type
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
	mod             *sources.Module
	hostServiceName string
	path            string
	filename        string
	file            *ast.File
	imports         sources.Imports
	decl            *ast.FuncDecl
	Ident           string
	VarIdent        string
	ProxyIdent      string
	ProxyAsyncIdent string
	HandlerIdent    string
	Annotations     sources.Annotations
	Param           *FunctionField
	Result          *FunctionField
}

func (f *Function) HostServiceName() (name string) {
	name = f.hostServiceName
	return
}

func (f *Function) Name() (name string) {
	name, _ = f.Annotations.Value("fn")
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
	has := false
	title, has = f.Annotations.Value("title")
	if !has {
		title = f.Name()
	}
	return
}

func (f *Function) Description() (description string) {
	description, _ = f.Annotations.Value("description")
	return
}

func (f *Function) Errors() (errs string) {
	errs, _ = f.Annotations.Value("errors")
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

func (f *Function) Metric() (ok bool) {
	_, ok = f.Annotations.Get("metric")
	return
}

func (f *Function) Barrier() (ok bool) {
	_, ok = f.Annotations.Get("barrier")
	return
}

func (f *Function) Cache() (cmd string, ttl string, has bool) {
	anno, exist := f.Annotations.Get("cache")
	if !exist {
		return
	}
	if len(anno.Params) == 0 {
		return
	}
	cmd = strings.TrimSpace(anno.Params[0])
	switch cmd {
	case "get":
		has = true
		break
	case "set":
		if len(anno.Params) == 1 {
			ttl = "10"
			has = true
			break
		}
		ttl = anno.Params[1]
		sec, ttlErr := strconv.Atoi(ttl)
		if ttlErr != nil {
			ttl = "10"
			has = true
			break
		}
		if sec < 1 {
			ttl = "10"
			has = true
			break
		}
		has = true
		break
	case "remove":
		has = true
		break
	case "get-set":
		if len(anno.Params) == 1 {
			ttl = "10"
			has = true
			break
		}
		ttl = anno.Params[1]
		sec, ttlErr := strconv.Atoi(ttl)
		if ttlErr != nil {
			ttl = "10"
			has = true
			break
		}
		if sec < 1 {
			ttl = "10"
			has = true
			break
		}
		has = true
		break
	default:
		break
	}
	return
}

func (f *Function) CacheControl() (maxAge int, public bool, mustRevalidate bool, proxyRevalidate bool, has bool, err error) {
	anno, exist := f.Annotations.Get("cache-control")
	if !exist {
		return
	}
	has = true
	if len(anno.Params) == 0 {
		return
	}
	for _, param := range anno.Params {
		maxAgeValue, hasMaxValue := strings.CutPrefix(param, "max-age=")
		if hasMaxValue {
			maxAge, err = strconv.Atoi(strings.TrimSpace(maxAgeValue))
			if err != nil {
				err = errors.Warning("fns: parse @cache-control max-age failed").WithMeta("max-age", maxAgeValue)
				return
			}
		}
		publicValue, hasPublic := strings.CutPrefix(param, "public=")
		if hasPublic {
			public, err = strconv.ParseBool(strings.TrimSpace(publicValue))
			if err != nil {
				err = errors.Warning("fns: parse @cache-control public failed").WithMeta("public", publicValue)
				return
			}
		}
		mustRevalidateValue, hasMustRevalidate := strings.CutPrefix(param, "must-revalidate=")
		if hasMustRevalidate {
			mustRevalidate, err = strconv.ParseBool(strings.TrimSpace(mustRevalidateValue))
			if err != nil {
				err = errors.Warning("fns: parse @cache-control must-revalidate failed").WithMeta("must-revalidate", mustRevalidateValue)
				return
			}
		}
		proxyRevalidateValue, hasProxyRevalidate := strings.CutPrefix(param, "proxy-revalidate=")
		if hasProxyRevalidate {
			proxyRevalidate, err = strconv.ParseBool(strings.TrimSpace(proxyRevalidateValue))
			if err != nil {
				err = errors.Warning("fns: parse @cache-control proxy-revalidate failed").WithMeta("proxy-revalidate", proxyRevalidateValue)
				return
			}
		}
	}
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

func (f *Function) ForeachAnnotations(fn func(name string, params []string)) {
	for _, annotation := range f.Annotations {
		fn(annotation.Name, annotation.Params)
	}
}

func (f *Function) FieldImports() (v sources.Imports) {
	v = sources.Imports{}
	paths := make([]string, 0, 1)
	if f.Param != nil {
		paths = append(paths, f.Param.Paths()...)
	}
	if f.Result != nil {
		paths = append(paths, f.Result.Paths()...)
	}
	for _, path := range paths {
		v.Add(&sources.Import{
			Path:  path,
			Alias: "",
		})
	}
	return
}

func (f *Function) Parse(ctx context.Context) (err error) {
	if ctx.Err() != nil {
		err = errors.Warning("modules: parse function failed").WithCause(ctx.Err()).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}
	if f.decl.Type.TypeParams != nil && f.decl.Type.TypeParams.List != nil && len(f.decl.Type.TypeParams.List) > 0 {
		err = errors.Warning("modules: parse function failed").WithCause(errors.Warning("function can not use paradigm")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}

	// params
	params := f.decl.Type.Params
	if params == nil || params.List == nil || len(params.List) == 0 || len(params.List) > 2 {
		err = errors.Warning("modules: parse function failed").WithCause(errors.Warning("params length must be one or two")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}

	if !f.mod.Types().IsContextType(params.List[0].Type, f.imports) {
		err = errors.Warning("modules: parse function failed").WithCause(errors.Warning("first param must be context.Context")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}
	if len(params.List) == 2 {
		param, parseParamErr := f.parseField(ctx, params.List[1])
		if parseParamErr != nil {
			err = errors.Warning("modules: parse function failed").WithCause(parseParamErr).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
			return
		}
		f.Param = param
	}
	// results
	results := f.decl.Type.Results
	if results == nil || results.List == nil || len(results.List) == 0 || len(results.List) > 2 {
		err = errors.Warning("modules: parse function failed").WithCause(errors.Warning("results length must be one or two")).
			WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
		return
	}
	if len(results.List) == 1 {
		if !f.mod.Types().IsCodeErrorType(results.List[0].Type, f.imports) {
			err = errors.Warning("modules: parse function failed").WithCause(errors.Warning("the last results must be error or github.com/aacfactory/errors.CodeError")).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
			return
		}
	} else {
		if !f.mod.Types().IsCodeErrorType(results.List[1].Type, f.imports) {
			err = errors.Warning("modules: parse function failed").WithCause(errors.Warning("the last results must be error or github.com/aacfactory/errors.CodeError")).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
			return
		}
		result, parseResultErr := f.parseField(ctx, results.List[0])
		if parseResultErr != nil {
			err = errors.Warning("modules: parse function failed").WithCause(parseResultErr).
				WithMeta("service", f.hostServiceName).WithMeta("function", f.Ident).WithMeta("file", f.filename)
			return
		}
		f.Result = result
	}
	return
}

func (f *Function) parseField(ctx context.Context, field *ast.Field) (v *FunctionField, err error) {
	if len(field.Names) != 1 {
		err = errors.Warning("modules: field must has only one name")
		return
	}
	name := field.Names[0].Name
	typ, parseTypeErr := f.parseFieldType(ctx, field.Type)
	if parseTypeErr != nil {
		err = errors.Warning("modules: parse field failed").WithMeta("field", name).WithCause(parseTypeErr)
		return
	}
	v = &FunctionField{
		mod:  f.mod,
		Name: name,
		Type: typ,
	}
	return
}

func (f *Function) parseFieldType(ctx context.Context, e ast.Expr) (typ *sources.Type, err error) {
	switch e.(type) {
	case *ast.Ident:
		typ, err = f.mod.Types().ParseExpr(ctx, e, &sources.TypeScope{
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
			err = errors.Warning("modules: field type only support value object")
			return
		}
		typ.Path = f.path
		typ.Name = e.(*ast.Ident).Name
	case *ast.SelectorExpr:
		typ, err = f.mod.Types().ParseExpr(ctx, e, &sources.TypeScope{
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
			err = errors.Warning("modules: field type only support value object")
			return
		}
		break
	default:
		err = errors.Warning("modules: field type only support no paradigms value object or typed slice").WithMeta("expr", reflect.TypeOf(e).String())
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
