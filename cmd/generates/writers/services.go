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

package writers

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

func NewServiceFile(service *sources.Service) (file CodeFileWriter) {
	file = &ServiceFile{
		service: service,
	}
	return
}

type ServiceFile struct {
	service *sources.Service
}

func (s *ServiceFile) Name() (name string) {
	name = filepath.ToSlash(filepath.Join(s.service.Dir, "fns.go"))
	return
}

func (s *ServiceFile) Write(ctx context.Context) (err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
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
		err = errors.Warning("sources: code file write failed").
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
		err = errors.Warning("sources: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(namesErr)
		return
	}
	file.AddCode(names)

	// fn handler and proxy
	proxies, proxiesErr := s.functionsCode(ctx)
	if proxiesErr != nil {
		err = errors.Warning("sources: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(proxiesErr)
		return
	}
	file.AddCode(proxies)

	// service
	service, serviceErr := s.serviceCode(ctx)
	if serviceErr != nil {
		err = errors.Warning("sources: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(serviceErr)
		return
	}
	file.AddCode(service)

	buf := bytes.NewBuffer([]byte{})

	renderErr := file.Render(buf)
	if renderErr != nil {
		err = errors.Warning("sources: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(renderErr)
		return
	}
	writer, openErr := os.OpenFile(s.Name(), os.O_CREATE|os.O_TRUNC|os.O_RDWR|os.O_SYNC, 0644)
	if openErr != nil {
		err = errors.Warning("sources: code file write failed").
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
			err = errors.Warning("sources: code file write failed").
				WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
				WithCause(writeErr)
			return
		}
		n += nn
	}
	syncErr := writer.Sync()
	if syncErr != nil {
		err = errors.Warning("sources: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(syncErr)
		return
	}
	closeErr := writer.Close()
	if closeErr != nil {
		err = errors.Warning("sources: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(closeErr)
		return
	}
	return
}

func (s *ServiceFile) importsCode(ctx context.Context) (packages []*gcg.Package, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
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
		err = errors.Warning("sources: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Vars()
	stmt.Add(gcg.Var("_endpointName", gcg.Token(fmt.Sprintf("[]byte(\"%s\")", s.service.Name))))
	for _, function := range s.service.Functions {
		stmt.Add(gcg.Var(function.ConstIdent, gcg.Token(fmt.Sprintf("[]byte(\"%s\")", function.Name()))))
	}
	code = stmt.Build()
	return
}

func (s *ServiceFile) functionsCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Statements()

	for _, function := range s.service.Functions {
		stmt.Add(gcg.Token("// +-------------------------------------------------------------------------------------------------------------------+").Line())
		// proxy
		proxy, proxyErr := s.functionProxyCode(ctx, function)
		if proxyErr != nil {
			err = proxyErr
			return
		}
		stmt.Add(proxy).Line()
		// handler

	}
	return
}

func (s *ServiceFile) functionProxyCode(ctx context.Context, function *sources.Function) (code gcg.Code, err error) {
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
				err = errors.Warning("sources: make function proxy code failed").
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
	if function.Result != nil {
		var result gcg.Code = nil
		if s.service.Path == function.Result.Type.Path {
			result = gcg.Ident(function.Param.Type.Name)
		} else {
			pkg, hasPKG := s.service.Imports.Path(function.Result.Type.Path)
			if !hasPKG {
				err = errors.Warning("sources: make function proxy code failed").
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
	// validate
	if validTitle, valid := function.Validation(); valid {
		body.Tab().Token("// validate param").Line()
		if validTitle == "" {
			body.Tab().Token("if err = validators.Validate(param); err != nil {").Line()
			body.Tab().Tab().Token("return").Line()
			body.Tab().Token("}").Line()
		} else {
			body.Tab().Token(fmt.Sprintf("if err = validators.ValidateWithErrorTitle(param, \"%s\"); err != nil {", validTitle)).Line()
			body.Tab().Tab().Token("return").Line()
			body.Tab().Token("}").Line()
		}
	}
	// cache
	function.Cache()
	// body <<<
	code = proxy.Build()
	return
}
func (s *ServiceFile) proxyFunctionsCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Statements()
	for _, function := range s.service.Functions {
		constIdent := function.ConstIdent
		proxyIdent := function.ProxyIdent
		proxy := gcg.Func()
		proxy.Name(proxyIdent)
		proxy.AddParam("ctx", gcg.QualifiedIdent(gcg.NewPackage("context"), "Context"))
		if function.Param != nil {
			var param gcg.Code = nil
			if s.service.Path == function.Param.Type.Path {
				param = gcg.Ident(function.Param.Type.Name)
			} else {
				pkg, hasPKG := s.service.Imports.Path(function.Param.Type.Path)
				if !hasPKG {
					err = errors.Warning("sources: make function proxy code failed").
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
			proxy.AddParam("argument", param)
		}
		if function.Result != nil {
			var result gcg.Code = nil
			if s.service.Path == function.Result.Type.Path {
				result = gcg.Ident(function.Param.Type.Name)
			} else {
				pkg, hasPKG := s.service.Imports.Path(function.Result.Type.Path)
				if !hasPKG {
					err = errors.Warning("sources: make function proxy code failed").
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
		proxy.AddResult("err", gcg.QualifiedIdent(gcg.NewPackage("github.com/aacfactory/errors"), "CodeError"))
		// body
		body := gcg.Statements()
		body.Tab().Ident("endpoint").Symbol(",").Space().Ident("hasEndpoint").Space().ColonEqual().Space().Token("service.GetEndpoint(ctx, _name)").Line()
		body.Tab().Token("if !hasEndpoint {").Line()
		body.Tab().Tab().Token(fmt.Sprintf("err = errors.Warning(\"%s: endpoint was not found\").WithMeta(\"name\", _name)", s.service.Name)).Line()
		body.Tab().Tab().Return().Line()
		body.Tab().Token("}").Line()
		if function.Param == nil {
			body.Tab().Token("argument := service.Empty{}").Line()
		}
		bodyArgumentCode := gcg.Statements().Token("service.NewArgument(argument)")
		bodyRequestCode := gcg.Statements().Token(fmt.Sprintf("service.NewRequest(ctx, _name, %s, ", constIdent)).Add(bodyArgumentCode).Symbol(",").Token("service.WithInternalRequest()").Symbol(")")
		if function.Result == nil {
			body.Tab().Token("_, err = endpoint.RequestSync(ctx, ").Add(bodyRequestCode).Symbol(")").Line()
		} else {
			body.Tab().Token("fr, requestErr := endpoint.RequestSync(ctx, ").Add(bodyRequestCode).Symbol(")").Line()
			body.Tab().Token("if requestErr != nil {").Line()
			body.Tab().Tab().Token("err = requestErr").Line()
			body.Tab().Tab().Return().Line()
			body.Tab().Token("}").Line()
			body.Tab().Token("if !fr.Exist() {").Line()
			body.Tab().Tab().Return().Line()
			body.Tab().Token("}").Line()
			body.Tab().Token("scanErr := fr.Scan(&result)").Line()
			body.Tab().Token("if scanErr != nil {").Line()
			body.Tab().Tab().Token(fmt.Sprintf("err = errors.Warning(\"%s: scan future result failed\")", s.service.Name)).Dot().Line()
			body.Tab().Tab().Tab().Token(fmt.Sprintf("WithMeta(\"service\", _name).WithMeta(\"fn\", %s)", constIdent)).Dot().Line()
			body.Tab().Tab().Tab().Token("WithCause(scanErr)").Line()
			body.Tab().Tab().Return().Line()
			body.Tab().Token("}").Line()
		}
		body.Tab().Return()
		proxy.Body(body)
		stmt = stmt.Add(proxy.Build()).Line()
	}
	code = stmt
	return
}

func (s *ServiceFile) serviceCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Statements()
	instanceCode, instanceCodeErr := s.serviceInstanceCode(ctx)
	if instanceCodeErr != nil {
		err = instanceCodeErr
		return
	}
	stmt.Add(instanceCode).Line()

	typeCode, typeCodeErr := s.serviceTypeCode(ctx)
	if typeCodeErr != nil {
		err = typeCodeErr
		return
	}
	stmt.Add(typeCode).Line()

	docFnCode, docFnCodeErr := s.serviceDocumentCode(ctx)
	if docFnCodeErr != nil {
		err = docFnCodeErr
		return
	}
	if docFnCode != nil {
		stmt.Add(docFnCode).Line()
	}

	code = stmt
	return
}

func (s *ServiceFile) functionTypeCode(ctx context.Context) (code gcg.Code, err error) {

	return
}

func (s *ServiceFile) serviceInstanceCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	instance := gcg.Func()
	instance.Name("Service")
	instance.AddResult("v", gcg.Token("services.Service"))
	body := gcg.Statements()
	body.Tab().Token("v = &service{").Line()
	body.Tab().Tab().Token("Abstract: services.NewAbstract(").Line()
	body.Tab().Tab().Tab().Token("_name,").Line()
	if s.service.Internal {
		body.Tab().Tab().Tab().Token("true,").Line()
	} else {
		body.Tab().Tab().Tab().Token("false,").Line()
	}
	if s.service.Components != nil && s.service.Components.Len() > 0 {
		//body.Tab().Tab().Tab().Token("[]service.Component{").Line()
		path := fmt.Sprintf("%s/components", s.service.Path)
		for _, component := range s.service.Components {
			componentCode := gcg.QualifiedIdent(gcg.NewPackage(path), component.Indent)
			body.Tab().Tab().Token("&").Add(componentCode).Token("{}").Symbol(",").Line()
		}
		//body.Tab().Tab().Tab().Token("}...,").Line()
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
		err = errors.Warning("sources: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	abstractFieldCode := gcg.StructField("")
	abstractFieldCode.Type(gcg.Token("service.Abstract", gcg.NewPackage("github.com/aacfactory/fns/service")))
	serviceStructCode := gcg.Struct()
	serviceStructCode.AddField(abstractFieldCode)
	code = gcg.Type("_service", serviceStructCode.Build())
	return
}

func (s *ServiceFile) serviceDocumentCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	if s.service.Title == "" {
		return
	}
	fnCodes := make([]gcg.Code, 0, 1)
	for _, function := range s.service.Functions {
		if function.Title() == "" {
			continue
		}
		fnCode := gcg.Statements()
		fnCode.Token("// ").Token(function.Name()).Line()
		fnCode.Token("document.AddFn(").Line()
		fnCode.Tab().Token("documents.NewFn(").Token("\"").Token(function.Name()).Token("\"").Token(")").Dot().Line()
		fnCode.Tab().Tab().Token("SetInfo(").Token("\"").Token(function.Title()).Token("\", ").Token("\"").Token(function.Description()).Token("\"").Token(")").Dot().Line()
		fnCode.Tab().Tab().
			Token(fmt.Sprintf("SetInternal(%v)", function.Internal())).Dot().
			Token(fmt.Sprintf("SetReadonly(%v)", function.Readonly())).Dot().
			Token(fmt.Sprintf("SetDeprecated(%v)", function.Deprecated())).Dot().Line()
		fnCode.Tab().Tab().
			Token(fmt.Sprintf("SetAuthorization(%v)", function.Authorization())).Dot().
			Token(fmt.Sprintf("SetPermission(%v)", function.Permission())).Dot().Line()
		if function.Param == nil {
			fnCode.Tab().Tab().Token("SetParam(documents.Nil())").Dot().Line()
		} else {
			paramCode, paramCodeErr := mapTypeToFunctionElementCode(ctx, function.Param.Type)
			if paramCodeErr != nil {
				err = errors.Warning("sources: make service document code failed").
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
				err = errors.Warning("sources: make service document code failed").
					WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
					WithMeta("function", function.Name()).
					WithCause(resultCodeErr)
				return
			}
			fnCode.Tab().Tab().Token("SetResult(").Add(resultCode).Token(")").Dot().Line()
		}
		fnCode.Tab().Token(fmt.Sprintf("SetErrors(\"%s\"),", function.Errors()))
		fnCode.Token(")").Line()
		fnCodes = append(fnCodes, fnCode)
	}
	if len(fnCodes) == 0 {
		return
	}
	docFnCode := gcg.Func()
	docFnCode.Receiver("svc", gcg.Star().Ident("service"))
	docFnCode.Name("Document")
	docFnCode.AddResult("document", gcg.QualifiedIdent(gcg.NewPackage("github.com/aacfactory/fns/services/documents"), "Endpoint"))
	body := gcg.Statements()
	body.Token(fmt.Sprintf("document = documents.New(svc.Name(), \"%s\", svc.Internal(), svc.Version())", s.service.Description))
	for _, fnCode := range fnCodes {
		body.Line().Token("document.AddFn(").Line().Add(fnCode).Line().Token(")")
	}
	body.Return().Line()
	docFnCode.Body(body)
	code = docFnCode.Build()
	return
}
