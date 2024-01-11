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
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"github.com/aacfactory/gcg"
	"os"
	"path/filepath"
	"strings"
)

func NewServiceFile(service *Service, annotations FnAnnotationCodeWriters) (file CodeFileWriter) {
	file = &ServiceFile{
		service:     service,
		annotations: annotations,
	}
	return
}

type ServiceFile struct {
	service     *Service
	annotations FnAnnotationCodeWriters
}

func (s *ServiceFile) Name() (name string) {
	name = filepath.ToSlash(filepath.Join(s.service.Dir, "fns.go"))
	return
}

func (s *ServiceFile) Write(ctx context.Context) (err error) {
	if ctx.Err() != nil {
		err = errors.Warning("modules: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).
			WithCause(ctx.Err())
		return
	}

	file := gcg.NewFileWithoutNote(s.service.Path[strings.LastIndex(s.service.Path, "/")+1:])
	// comments
	file.FileComments("NOTE: this file has been automatically generated, DON'T EDIT IT!!!\n")

	// imports
	packages, importsErr := s.importsCode(ctx)
	if importsErr != nil {
		err = errors.Warning("modules: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(importsErr)
		return
	}
	if packages != nil && len(packages) > 0 {
		for _, importer := range packages {
			file.AddImport(importer)
		}
	}

	// names
	names, namesErr := s.constNamesCode(ctx)
	if namesErr != nil {
		err = errors.Warning("modules: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(namesErr)
		return
	}
	file.AddCode(names)

	// fn handler and proxy
	proxies, proxiesErr := s.functionsCode(ctx)
	if proxiesErr != nil {
		err = errors.Warning("modules: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(proxiesErr)
		return
	}
	file.AddCode(proxies)

	// componentCode
	component, componentErr := s.componentCode(ctx)
	if componentErr != nil {
		err = errors.Warning("modules: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(componentErr)
		return
	}
	file.AddCode(component)

	// service
	service, serviceErr := s.serviceCode(ctx)
	if serviceErr != nil {
		err = errors.Warning("modules: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(serviceErr)
		return
	}
	file.AddCode(service)

	buf := bytes.NewBuffer([]byte{})

	renderErr := file.Render(buf)
	if renderErr != nil {
		err = errors.Warning("modules: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(renderErr)
		return
	}
	writer, openErr := os.OpenFile(s.Name(), os.O_CREATE|os.O_TRUNC|os.O_RDWR|os.O_SYNC, 0644)
	if openErr != nil {
		err = errors.Warning("modules: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(openErr)
		return
	}
	n := 0
	bodyLen := buf.Len()
	body := buf.Bytes()
	for n < bodyLen {
		nn, writeErr := writer.Write(body[n:])
		if writeErr != nil {
			err = errors.Warning("modules: code file write failed").
				WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
				WithCause(writeErr)
			return
		}
		n += nn
	}
	syncErr := writer.Sync()
	if syncErr != nil {
		err = errors.Warning("modules: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(syncErr)
		return
	}
	closeErr := writer.Close()
	if closeErr != nil {
		err = errors.Warning("modules: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(closeErr)
		return
	}
	return
}

func (s *ServiceFile) importsCode(ctx context.Context) (packages []*gcg.Package, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("modules: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	packages = make([]*gcg.Package, 0, 1)
	for _, i := range s.service.Imports {
		if i.Alias != "" {
			packages = append(packages, gcg.NewPackageWithAlias(i.Path, i.Alias))
		} else {
			packages = append(packages, gcg.NewPackage(i.Path))
		}
	}
	return
}

func (s *ServiceFile) constNamesCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("modules: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Vars()
	stmt.Add(gcg.Var("_endpointName", gcg.Token(fmt.Sprintf(" = []byte(\"%s\")", s.service.Name))))
	for _, function := range s.service.Functions {
		stmt.Add(gcg.Var(function.VarIdent, gcg.Token(fmt.Sprintf(" = []byte(\"%s\")", function.Name()))))
	}
	code = stmt.Build()
	return
}

func (s *ServiceFile) componentCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("modules: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Statements()
	stmt.Add(gcg.Token("// +-------------------------------------------------------------------------------------------------------------------+").Line().Line())

	fn := gcg.Func()
	fn.Name("Component[C services.Component]")
	fn.AddParam("ctx", contextCode())
	fn.AddParam("name", gcg.Token("string"))
	fn.AddResult("component", gcg.Token("C"))
	fn.AddResult("has", gcg.Token("bool"))
	body := gcg.Statements()
	body.Tab().Token("component, has = services.LoadComponent[C](ctx, _endpointName, name)").Line()
	body.Tab().Return()
	fn.Body(body)

	stmt.Add(fn.Build()).Line()
	code = stmt
	return
}

func (s *ServiceFile) serviceCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("modules: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Statements()
	// instance
	stmt.Add(gcg.Token("// +-------------------------------------------------------------------------------------------------------------------+").Line().Line())
	instanceCode, instanceCodeErr := s.serviceInstanceCode(ctx)
	if instanceCodeErr != nil {
		err = instanceCodeErr
		return
	}
	stmt.Add(instanceCode).Line()

	// type
	stmt.Add(gcg.Token("// +-------------------------------------------------------------------------------------------------------------------+").Line().Line())
	typeCode, typeCodeErr := s.serviceTypeCode(ctx)
	if typeCodeErr != nil {
		err = typeCodeErr
		return
	}
	stmt.Add(typeCode).Line()
	// construct
	constructFnCode, constructCodeErr := s.serviceConstructCode(ctx)
	if constructCodeErr != nil {
		err = constructCodeErr
		return
	}
	if constructFnCode != nil {
		stmt.Add(constructFnCode).Line()
	}
	// doc
	docCode, docCodeErr := s.serviceDocumentCode(ctx)
	if docCodeErr != nil {
		err = docCodeErr
		return
	}
	if docCode != nil {
		stmt.Add(docCode).Line()
	}

	code = stmt
	return
}

func (s *ServiceFile) serviceInstanceCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("modules: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	instance := gcg.Func()
	instance.Name("Service")
	instance.AddResult("v", gcg.Token("services.Service"))
	body := gcg.Statements()
	body.Tab().Token("v = &_service{").Line()
	body.Tab().Tab().Token("Abstract: services.NewAbstract(").Line()
	body.Tab().Tab().Tab().Token("string(_endpointName),").Line()
	if s.service.Internal {
		body.Tab().Tab().Tab().Token("true,").Line()
	} else {
		body.Tab().Tab().Tab().Token("false,").Line()
	}
	if s.service.Components != nil && s.service.Components.Len() > 0 {
		path := fmt.Sprintf("%s/components", s.service.Path)
		for _, component := range s.service.Components {
			componentCode := gcg.QualifiedIdent(gcg.NewPackage(path), component.Indent)
			body.Tab().Tab().Token("&").Add(componentCode).Token("{}").Symbol(",").Line()
		}
	}
	body.Tab().Tab().Symbol(")").Symbol(",").Line()
	body.Tab().Symbol("}").Line()
	body.Tab().Return()

	instance.Body(body)
	code = instance.Build()
	return
}

func (s *ServiceFile) serviceTypeCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("modules: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	abstractFieldCode := gcg.StructField("")
	abstractFieldCode.Type(gcg.Token("services.Abstract", gcg.NewPackage("github.com/aacfactory/fns/services")))
	serviceStructCode := gcg.Struct()
	serviceStructCode.AddField(abstractFieldCode)
	code = gcg.Type("_service", serviceStructCode.Build())
	return
}

func (s *ServiceFile) serviceConstructCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("modules: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	construct := gcg.Func()
	construct.Name("Construct")
	construct.Receiver("svc", gcg.Star().Ident("_service"))
	construct.AddParam("options", gcg.QualifiedIdent(gcg.NewPackage("github.com/aacfactory/fns/services"), "Options"))
	construct.AddResult("err", gcg.Ident("error"))

	body := gcg.Statements()
	body.Tab().Token("if err = svc.Abstract.Construct(options); err != nil {").Line()
	body.Tab().Tab().Token("return").Line()
	body.Tab().Token("}").Line()

	for _, function := range s.service.Functions {
		body.Tab().Token("svc.AddFunction(")
		body.Token(fmt.Sprintf("commons.NewFn(string(%s)", function.VarIdent))
		body.Token(fmt.Sprintf(", %v", function.Readonly()))
		body.Token(fmt.Sprintf(", %v", function.Internal()))
		body.Token(fmt.Sprintf(", %v", function.Authorization()))
		body.Token(fmt.Sprintf(", %v", function.Permission()))
		body.Token(fmt.Sprintf(", %v", function.Metric()))
		body.Token(fmt.Sprintf(", %v", function.Barrier()))
		body.Token(fmt.Sprintf(", %s", function.HandlerIdent))
		middlewares := function.Middlewares()
		for _, middleware := range middlewares {
			if idx := strings.LastIndexByte(middleware, '.'); idx > 0 {
				pkg := middleware[0:idx]
				name := middleware[idx+1:]
				middleImport, hasMiddleImport := s.service.Imports.Path(pkg)
				if !hasMiddleImport {
					middlewareImports := sources.Imports{}
					middlewareImports.Add(&sources.Import{
						Path:  pkg,
						Alias: "",
					})
					s.service.Imports = sources.MergeImports([]sources.Imports{s.service.Imports, middlewareImports})
					middleImport, _ = s.service.Imports.Path(pkg)
				}
				body.Token(fmt.Sprintf(", &%s.%s{}", middleImport.Ident(), name))
			} else {
				body.Token(fmt.Sprintf(", &%s{}", middleware))
			}
		}
		body.Token("))")
		body.Line()
	}
	body.Tab().Return()
	construct.Body(body)
	code = construct.Build()
	return
}

func (s *ServiceFile) serviceDocumentCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("modules: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	if s.service.Title == "" {
		s.service.Title = s.service.Name
	}
	fnCodes := make([]gcg.Code, 0, 1)
	for _, function := range s.service.Functions {
		if function.Internal() {
			continue
		}
		fnCode := gcg.Statements()
		fnCode.Token("// ").Token(function.Name()).Line()
		fnCode.Token("document.AddFn(").Line()
		fnCode.Tab().Token("documents.NewFn(").Token("\"").Token(function.Name()).Token("\"").Token(")").Dot().Line()
		fnCode.Tab().Tab().Token("SetInfo(").Token("\"").Token(strings.ReplaceAll(function.Title(), "\n", "\\n")).Token("\", ").Token("\"").Token(strings.ReplaceAll(function.Description(), "\n", "\\n")).Token("\"").Token(")").Dot().Line()
		fnCode.Tab().Tab().
			Token(fmt.Sprintf("SetReadonly(%v)", function.Readonly())).Dot().
			Token(fmt.Sprintf("SetInternal(%v)", function.Internal())).Dot().
			Token(fmt.Sprintf("SetDeprecated(%v)", function.Deprecated())).Dot().Line()
		fnCode.Tab().Tab().
			Token(fmt.Sprintf("SetAuthorization(%v)", function.Authorization())).Dot().
			Token(fmt.Sprintf("SetPermission(%v)", function.Permission())).Dot().Line()
		if function.Param == nil {
			fnCode.Tab().Tab().Token("SetParam(documents.Nil())").Dot().Line()
		} else {
			paramCode, paramCodeErr := mapTypeToFunctionElementCode(ctx, function.Param.Type)
			if paramCodeErr != nil {
				err = errors.Warning("modules: make service document code failed").
					WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
					WithMeta("function", function.Name()).
					WithCause(paramCodeErr)
				return
			}
			fnCode.Tab().Tab().Token("SetParam(").Add(paramCode).Token(")").Dot().Line()
		}
		if function.Result == nil {
			fnCode.Tab().Tab().Token("SetResult(documents.Nil())").Dot().Line()
		} else {
			resultCode, resultCodeErr := mapTypeToFunctionElementCode(ctx, function.Result.Type)
			if resultCodeErr != nil {
				err = errors.Warning("modules: make service document code failed").
					WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
					WithMeta("function", function.Name()).
					WithCause(resultCodeErr)
				return
			}
			fnCode.Tab().Tab().Token("SetResult(").Add(resultCode).Token(")").Dot().Line()
		}
		fnCode.Tab().Token(fmt.Sprintf("SetErrors(\"%s\"),", strings.ReplaceAll(function.Errors(), "\n", "\\n"))).Line()
		fnCode.Token(")")
		fnCodes = append(fnCodes, fnCode)
	}
	if len(fnCodes) == 0 {
		return
	}
	docFnCode := gcg.Func()
	docFnCode.Receiver("svc", gcg.Star().Ident("_service"))
	docFnCode.Name("Document")
	docFnCode.AddResult("document", gcg.QualifiedIdent(gcg.NewPackage("github.com/aacfactory/fns/services/documents"), "Endpoint"))
	body := gcg.Statements()
	body.Token(fmt.Sprintf("document = documents.New(svc.Name(), \"%s\", \"%s\")", strings.ReplaceAll(s.service.Title, "\n", "\\n"), strings.ReplaceAll(s.service.Description, "\n", "\\n")))
	if s.service.Internal {
		body.Token("document.SetInternal()").Line()
	}
	for _, fnCode := range fnCodes {
		body.Line().Add(fnCode).Line()
	}
	body.Tab().Return()
	docFnCode.Body(body)
	code = docFnCode.Build()
	return
}

func (s *ServiceFile) functionsCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("modules: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Statements()

	for _, function := range s.service.Functions {
		stmt.Add(gcg.Token("// +-------------------------------------------------------------------------------------------------------------------+").Line().Line())
		// proxy
		proxy, proxyErr := s.functionProxyCode(ctx, function)
		if proxyErr != nil {
			err = proxyErr
			return
		}
		stmt.Add(proxy).Line()
		// proxy async
		proxyAsync, proxyAsyncErr := s.functionProxyAsyncCode(ctx, function)
		if proxyAsyncErr != nil {
			err = proxyAsyncErr
			return
		}
		stmt.Add(proxyAsync).Line()
		// handler
		handler, handlerErr := s.functionHandlerCode(ctx, function)
		if handlerErr != nil {
			err = handlerErr
			return
		}
		stmt.Add(handler).Line()
	}

	code = stmt
	return
}

func (s *ServiceFile) functionProxyAsyncCode(ctx context.Context, function *Function) (code gcg.Code, err error) {
	proxyIdent := function.ProxyAsyncIdent
	proxy := gcg.Func()
	proxy.Name(proxyIdent)
	proxy.AddParam("ctx", contextCode())
	if function.Param != nil {
		var param gcg.Code = nil
		if s.service.Path == function.Param.Type.Path {
			param = gcg.Ident(function.Param.Type.Name)
		} else {
			pkg, hasPKG := s.service.Imports.Path(function.Param.Type.Path)
			if !hasPKG {
				err = errors.Warning("modules: make function proxy code failed").
					WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
					WithMeta("function", function.Name()).
					WithCause(errors.Warning("import of param was not found").WithMeta("path", function.Param.Type.Path))
				return
			}
			if pkg.Alias == "" {
				param = gcg.QualifiedIdent(gcg.NewPackage(pkg.Path), function.Param.Type.Name)
			} else {
				param = gcg.QualifiedIdent(gcg.NewPackageWithAlias(pkg.Path, pkg.Alias), function.Param.Type.Name)
			}
		}
		proxy.AddParam("param", param)
	}
	var result gcg.Code = nil
	if function.Result != nil {
		if s.service.Path == function.Result.Type.Path {
			result = gcg.Ident(function.Result.Type.Name)
		} else {
			pkg, hasPKG := s.service.Imports.Path(function.Result.Type.Path)
			if !hasPKG {
				err = errors.Warning("modules: make function proxy code failed").
					WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
					WithMeta("function", function.Name()).
					WithCause(errors.Warning("import of result was not found").WithMeta("path", function.Result.Type.Path))
				return
			}
			if pkg.Alias == "" {
				result = gcg.QualifiedIdent(gcg.NewPackage(pkg.Path), function.Result.Type.Name)
			} else {
				result = gcg.QualifiedIdent(gcg.NewPackageWithAlias(pkg.Path, pkg.Alias), function.Result.Type.Name)
			}
		}
		proxy.AddResult("result", result)
	}
	proxy.AddResult("future", gcg.QualifiedIdent(gcg.NewPackage("github.com/aacfactory/fns/commons/futures"), "Future"))
	proxy.AddResult("err", gcg.Ident("error"))
	// body >>>
	body := gcg.Statements()
	if function.Param != nil {
		// validate
		if validTitle, valid := function.Validation(); valid {
			body.Tab().Token("// validate param").Line()
			if validTitle == "" {
				body.Tab().Token("if err = validators.Validate(param); err != nil {", gcg.NewPackage("github.com/aacfactory/fns/services/validators")).Line()
				body.Tab().Tab().Token("return").Line()
				body.Tab().Token("}").Line()
			} else {
				body.Tab().Token(fmt.Sprintf("if err = validators.ValidateWithErrorTitle(param, \"%s\"); err != nil {", validTitle)).Line()
				body.Tab().Tab().Token("return").Line()
				body.Tab().Token("}").Line()
			}
		}
	}

	// handle
	body.Tab().Token("// handle").Line()
	body.Tab().Token("eps := runtime.Endpoints(ctx)").Line()
	if function.Param != nil {
		body.Tab().Token(fmt.Sprintf("req := services.NewRequest(ctx, _endpointName, %s, param)", function.VarIdent)).Line()
	} else {
		body.Tab().Token(fmt.Sprintf("req := services.NewRequest(ctx, _endpointName, %s, nil)", function.VarIdent)).Line()
	}
	body.Tab().Token("future, err = eps.RequestAsync(req)").Line()
	// return
	body.Tab().Token("return")
	// body <<<
	proxy.Body(body)
	code = proxy.Build()
	return
}

func (s *ServiceFile) functionProxyCode(ctx context.Context, function *Function) (code gcg.Code, err error) {
	proxyIdent := function.ProxyIdent
	proxy := gcg.Func()
	proxy.Name(proxyIdent)
	proxy.AddParam("ctx", contextCode())
	if function.Param != nil {
		var param gcg.Code = nil
		if s.service.Path == function.Param.Type.Path {
			param = gcg.Ident(function.Param.Type.Name)
		} else {
			pkg, hasPKG := s.service.Imports.Path(function.Param.Type.Path)
			if !hasPKG {
				err = errors.Warning("modules: make function proxy code failed").
					WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
					WithMeta("function", function.Name()).
					WithCause(errors.Warning("import of param was not found").WithMeta("path", function.Param.Type.Path))
				return
			}
			if pkg.Alias == "" {
				param = gcg.QualifiedIdent(gcg.NewPackage(pkg.Path), function.Param.Type.Name)
			} else {
				param = gcg.QualifiedIdent(gcg.NewPackageWithAlias(pkg.Path, pkg.Alias), function.Param.Type.Name)
			}
		}
		proxy.AddParam("param", param)
	}
	var result gcg.Code = nil
	if function.Result != nil {
		if s.service.Path == function.Result.Type.Path {
			result = gcg.Ident(function.Result.Type.Name)
		} else {
			pkg, hasPKG := s.service.Imports.Path(function.Result.Type.Path)
			if !hasPKG {
				err = errors.Warning("modules: make function proxy code failed").
					WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
					WithMeta("function", function.Name()).
					WithCause(errors.Warning("import of result was not found").WithMeta("path", function.Result.Type.Path))
				return
			}
			if pkg.Alias == "" {
				result = gcg.QualifiedIdent(gcg.NewPackage(pkg.Path), function.Result.Type.Name)
			} else {
				result = gcg.QualifiedIdent(gcg.NewPackageWithAlias(pkg.Path, pkg.Alias), function.Result.Type.Name)
			}
		}
		proxy.AddResult("result", result)
	}
	proxy.AddResult("err", gcg.Ident("error"))
	// body >>>
	body := gcg.Statements()
	if function.Param != nil {
		// validate
		if validTitle, valid := function.Validation(); valid {
			body.Tab().Token("// validate param").Line()
			if validTitle == "" {
				body.Tab().Token("if err = validators.Validate(param); err != nil {", gcg.NewPackage("github.com/aacfactory/fns/services/validators")).Line()
				body.Tab().Tab().Token("return").Line()
				body.Tab().Token("}").Line()
			} else {
				body.Tab().Token(fmt.Sprintf("if err = validators.ValidateWithErrorTitle(param, \"%s\"); err != nil {", validTitle)).Line()
				body.Tab().Tab().Token("return").Line()
				body.Tab().Token("}").Line()
			}
		}
		// cache
		cacheCmd, _, hasCache := function.Cache()
		if hasCache && function.Result != nil {
			if cacheCmd == "get" || cacheCmd == "get-set" {
				body.Tab().Token("// cache get").Line()
				body.Tab().Tab().Token("cached := ").Add(result).Token("{}").Line()
				body.Tab().Tab().Token("cacheExist, cacheGetErr := caches.Load(ctx, param, &cached)").Line()
				body.Tab().Token("if cacheGetErr != nil {").Line()
				body.Tab().Tab().Token("log := logs.Load(ctx)", gcg.NewPackage("github.com/aacfactory/fns/logs")).Line()
				body.Tab().Tab().Token("if log.WarnEnabled() {").Line()
				body.Tab().Tab().Tab().Token("log.Warn().Cause(cacheGetErr).With(\"fns\", \"caches\").Message(\"fns: get cache failed\")").Line()
				body.Tab().Tab().Token("}").Line()
				body.Tab().Token("}").Line()
				body.Tab().Token("if cacheExist {").Line()
				body.Tab().Tab().Token("result = cached").Line()
				body.Tab().Tab().Token("return").Line()
				body.Tab().Token("}").Line()
			}
		}
	}

	// handle
	body.Tab().Token("// handle").Line()
	body.Tab().Token("eps := runtime.Endpoints(ctx)").Line()
	if function.Param != nil {
		if function.Result == nil {
			body.Tab().Token(fmt.Sprintf("_, handleErr := eps.Request(ctx, _endpointName, %s, param)", function.VarIdent)).Line()
		} else {
			body.Tab().Token(fmt.Sprintf("response, handleErr := eps.Request(ctx, _endpointName, %s, param)", function.VarIdent)).Line()
		}
	} else {
		if function.Result == nil {
			body.Tab().Token(fmt.Sprintf("_, handleErr := eps.Request(ctx, _endpointName, %s, nil)", function.VarIdent)).Line()
		} else {
			body.Tab().Token(fmt.Sprintf("response, handleErr := eps.Request(ctx, _endpointName, %s, nil)", function.VarIdent)).Line()
		}
	}

	body.Tab().Token("if handleErr != nil {").Line()
	body.Tab().Tab().Token("err = handleErr").Line()
	body.Tab().Tab().Token("return").Line()
	body.Tab().Token("}").Line()
	if function.Result != nil {
		body.Tab().Token("result, err = services.ValueOfResponse[").Add(result).Token("](response)").Line()
	}

	// return
	body.Tab().Token("return")
	// body <<<
	proxy.Body(body)
	code = proxy.Build()
	return
}

func (s *ServiceFile) functionHandlerCode(ctx context.Context, function *Function) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("modules: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	handlerIdent := function.HandlerIdent
	handler := gcg.Func()
	handler.Name(handlerIdent)
	handler.AddParam("ctx", gcg.QualifiedIdent(gcg.NewPackage("github.com/aacfactory/fns/services"), "Request"))
	handler.AddResult("v", gcg.Ident("any"))
	handler.AddResult("err", gcg.Ident("error"))

	// body >>>
	body := gcg.Statements()
	var result gcg.Code = nil
	if function.Result != nil {
		if s.service.Path == function.Result.Type.Path {
			result = gcg.Ident(function.Result.Type.Name)
		} else {
			pkg, hasPKG := s.service.Imports.Path(function.Result.Type.Path)
			if !hasPKG {
				err = errors.Warning("modules: make function proxy code failed").
					WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
					WithMeta("function", function.Name()).
					WithCause(errors.Warning("import of result was not found").WithMeta("path", function.Result.Type.Path))
				return
			}
			if pkg.Alias == "" {
				result = gcg.QualifiedIdent(gcg.NewPackage(pkg.Path), function.Result.Type.Name)
			} else {
				result = gcg.QualifiedIdent(gcg.NewPackageWithAlias(pkg.Path, pkg.Alias), function.Result.Type.Name)
			}
		}
	}
	if function.Param != nil {
		// param
		var param gcg.Code = nil
		if s.service.Path == function.Param.Type.Path {
			param = gcg.Ident(function.Param.Type.Name)
		} else {
			pkg, hasPKG := s.service.Imports.Path(function.Param.Type.Path)
			if !hasPKG {
				err = errors.Warning("modules: make function handle function code failed").
					WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
					WithMeta("function", function.Name()).
					WithCause(errors.Warning("import of param was not found").WithMeta("path", function.Param.Type.Path))
				return
			}
			if pkg.Alias == "" {
				param = gcg.QualifiedIdent(gcg.NewPackage(pkg.Path), function.Param.Type.Name)
			} else {
				param = gcg.QualifiedIdent(gcg.NewPackageWithAlias(pkg.Path, pkg.Alias), function.Param.Type.Name)
			}
		}
		body.Tab().Token("// param").Line()
		body.Tab().Token("param, paramErr := services.ValueOfParam[").Add(param).Token("](ctx.Param())").Line()
		body.Tab().Token("if paramErr != nil {").Line()
		body.Tab().Tab().Token("err = errors.BadRequest(\"scan params failed\").WithCause(paramErr)").Line()
		body.Tab().Tab().Token("return").Line()
		body.Tab().Token("}").Line()
		// validate
		if validTitle, valid := function.Validation(); valid {
			body.Tab().Token("// validate param").Line()
			if validTitle == "" {
				body.Tab().Token("if err = validators.Validate(param); err != nil {", gcg.NewPackage("github.com/aacfactory/fns/services/validators")).Line()
				body.Tab().Tab().Token("return").Line()
				body.Tab().Token("}").Line()
			} else {
				body.Tab().Token(fmt.Sprintf("if err = validators.ValidateWithErrorTitle(param, \"%s\"); err != nil {", validTitle)).Line()
				body.Tab().Tab().Token("return").Line()
				body.Tab().Token("}").Line()
			}
		}
		// cache
		cacheCmd, _, hasCache := function.Cache()
		if hasCache && function.Result != nil {
			if cacheCmd == "get" || cacheCmd == "get-set" {
				body.Tab().Token("// cache get").Line()
				body.Tab().Tab().Token("cached := ").Add(result).Token("{}").Line()
				body.Tab().Tab().Token("cacheExist, cacheGetErr := caches.Load(ctx, param, &cached)").Line()
				body.Tab().Token("if cacheGetErr != nil {").Line()
				body.Tab().Tab().Token("log := logs.Load(ctx)", gcg.NewPackage("github.com/aacfactory/fns/logs")).Line()
				body.Tab().Tab().Token("if log.WarnEnabled() {").Line()
				body.Tab().Tab().Tab().Token("log.Warn().Cause(cacheGetErr).With(\"fns\", \"caches\").Message(\"fns: get cache failed\")").Line()
				body.Tab().Tab().Token("}").Line()
				body.Tab().Token("}").Line()
				body.Tab().Token("if cacheExist {").Line()
				body.Tab().Tab().Token("v = cached").Line()
				body.Tab().Tab().Token("return").Line()
				body.Tab().Token("}").Line()
			}
		}
	}
	// annotation writers before
	matchedAnnotations := make([]sources.Annotation, 0, 1)
	matchedAnnotationWriters := make([]FnAnnotationCodeWriter, 0, 1)
	for _, annotation := range function.Annotations {
		annotationWriter, hasAnnotationWriter := s.annotations.Get(annotation.Name)
		if hasAnnotationWriter {
			matchedAnnotations = append(matchedAnnotations, annotation)
			matchedAnnotationWriters = append(matchedAnnotationWriters, annotationWriter)
		}
	}
	for i, annotationWriter := range matchedAnnotationWriters {
		annotationCode, annotationCodeErr := annotationWriter.HandleBefore(ctx, matchedAnnotations[i].Params, function.Param != nil, function.Result != nil)
		if annotationCodeErr != nil {
			err = errors.Warning("modules: make function proxy code failed").
				WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
				WithMeta("function", function.Name()).
				WithCause(annotationCodeErr).WithMeta("annotation", annotationWriter.Annotation())
			return
		}
		if annotationCode != nil {
			body.Tab().Token("// generated by " + annotationWriter.Annotation()).Line()
			body.Add(annotationCode).Line()
		}
	}
	// handle
	body.Tab().Token("// handle").Line()
	if function.Param == nil && function.Result == nil {
		body.Tab().Token(fmt.Sprintf("err = %s(ctx)", function.Ident)).Line()
	} else if function.Param == nil && function.Result != nil {
		body.Tab().Token(fmt.Sprintf("v, err = %s(ctx)", function.Ident)).Line()
	} else if function.Param != nil && function.Result == nil {
		body.Tab().Token(fmt.Sprintf("err = %s(ctx, param)", function.Ident)).Line()
	} else {
		body.Tab().Token(fmt.Sprintf("v, err = %s(ctx, param)", function.Ident)).Line()
	}
	// cache
	cacheCmd, cacheTTL, hasCache := function.Cache()
	if hasCache && function.Param != nil {
		if function.Result != nil {
			if cacheCmd == "set" || cacheCmd == "get-set" {
				body.Tab().Token("// cache set").Line()
				body.Tab().Token(fmt.Sprintf("if cacheSetErr := caches.Set(ctx, param, v, %s*time.Second); cacheSetErr != nil {", cacheTTL), gcg.NewPackage("github.com/aacfactory/fns/services/caches")).Line()
				body.Tab().Tab().Token("log := logs.Load(ctx)", gcg.NewPackage("github.com/aacfactory/fns/logs")).Line()
				body.Tab().Tab().Token("if log.WarnEnabled() {").Line()
				body.Tab().Tab().Tab().Token("log.Warn().Cause(cacheSetErr).With(\"fns\", \"caches\").Message(\"fns: set cache failed\")").Line()
				body.Tab().Tab().Token("}").Line()
				body.Tab().Token("}").Line()
			}
		}
		if cacheCmd == "remove" {
			body.Tab().Token("// cache remove").Line()
			body.Tab().Token("if cacheRemoveErr := caches.Remove(ctx, param); cacheRemoveErr != nil {", gcg.NewPackage("github.com/aacfactory/fns/services/caches")).Line()
			body.Tab().Tab().Token("log := logs.Load(ctx)", gcg.NewPackage("github.com/aacfactory/fns/logs")).Line()
			body.Tab().Tab().Token("if log.WarnEnabled() {").Line()
			body.Tab().Tab().Tab().Token("log.Warn().Cause(cacheRemoveErr).With(\"fns\", \"caches\").Message(\"fns: remove cache failed\")").Line()
			body.Tab().Tab().Token("}").Line()
			body.Tab().Token("}").Line()
		}
	}
	// cache control
	if function.Readonly() && !function.Internal() && function.Result != nil {
		maxAge, public, mr, pr, hasCC, ccErr := function.CacheControl()
		if ccErr != nil {
			err = errors.Warning("modules: make function handler code failed").
				WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
				WithMeta("function", function.Name()).
				WithCause(ccErr)
			return
		}
		if hasCC {
			body.Tab().Token("// cache control").Line()
			body.Tab().Token("cachecontrol.Make(ctx", gcg.NewPackage("github.com/aacfactory/fns/transports/middlewares/cachecontrol"))
			if maxAge > 0 {
				body.Token(fmt.Sprintf(", cachecontrol.MaxAge(%d)", maxAge))
			}
			if public {
				body.Token(", cachecontrol.Public()")
			}
			if mr {
				body.Token(", cachecontrol.MustRevalidate()")
			}
			if pr {
				body.Token(", cachecontrol.ProxyRevalidate()")
			}
			body.Token(")").Line()
		}
	}
	// annotation writers after
	for i, annotationWriter := range matchedAnnotationWriters {
		annotationCode, annotationCodeErr := annotationWriter.HandleAfter(ctx, matchedAnnotations[i].Params, function.Param != nil, function.Result != nil)
		if annotationCodeErr != nil {
			err = errors.Warning("modules: make function handler code failed").
				WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
				WithMeta("function", function.Name()).
				WithCause(annotationCodeErr).WithMeta("annotation", annotationWriter.Annotation())
			return
		}
		if annotationCode != nil {
			body.Tab().Token("// generated by " + annotationWriter.Annotation()).Line()
			body.Add(annotationCode).Line()
		}
	}
	// return
	body.Tab().Token("return").Line()
	handler.Body(body)
	// body <<<

	code = handler.Build()
	return
}
