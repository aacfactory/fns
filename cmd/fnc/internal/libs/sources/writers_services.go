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
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/gcg"
	"os"
	"path/filepath"
	"strings"
)

func NewServiceFile(service *Service) (file CodeFileWriter) {
	file = &ServiceFile{
		service: service,
	}
	return
}

type ServiceFile struct {
	service *Service
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
	file.FileComments("NOTE: this file has been automatically generated, DON'T EDIT IT!!!\n")

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

	names, namesErr := s.constFunctionNamesCode(ctx)
	if namesErr != nil {
		err = errors.Warning("sources: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(namesErr)
		return
	}
	file.AddCode(names)

	proxies, proxiesErr := s.proxyFunctionsCode(ctx)
	if proxiesErr != nil {
		err = errors.Warning("sources: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(proxiesErr)
		return
	}
	file.AddCode(proxies)

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

func (s *ServiceFile) constFunctionNamesCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Constants()
	stmt.Add("_name", s.service.Name)
	for _, function := range s.service.Functions {
		stmt.Add(function.ConstIdent, function.Name())
	}
	code = stmt.Build()
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

	handleFnCode, handleFnCodeErr := s.serviceHandleCode(ctx)
	if handleFnCodeErr != nil {
		err = handleFnCodeErr
		return
	}
	stmt.Add(handleFnCode).Line()

	docFnCode, docFnCodeErr := s.serviceDocumentCode(ctx)
	if docFnCodeErr != nil {
		err = docFnCodeErr
		return
	}
	stmt.Add(docFnCode).Line()

	code = stmt
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
	instance.AddResult("v", gcg.Token("service.Service"))
	body := gcg.Statements()
	body.Tab().Token("v = &_service{").Line()
	body.Tab().Tab().Token("Abstract: service.NewAbstract(").Line()
	body.Tab().Tab().Tab().Token("_name,").Line()
	if s.service.Internal {
		body.Tab().Tab().Tab().Token("true,").Line()
	} else {
		body.Tab().Tab().Tab().Token("false,").Line()
	}
	if s.service.Components != nil && s.service.Components.Len() > 0 {
		body.Tab().Tab().Tab().Token("[]service.Component{").Line()
		path := fmt.Sprintf("%s/components", s.service.Path)
		for _, component := range s.service.Components {
			componentCode := gcg.QualifiedIdent(gcg.NewPackage(path), component.Indent)
			body.Tab().Tab().Token("&").Add(componentCode).Token("{}").Symbol(",").Line()
		}
		body.Tab().Tab().Tab().Token("}...,").Line()
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

func (s *ServiceFile) serviceHandleCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	handleFnCode := gcg.Func()
	handleFnCode.Receiver("svc", gcg.Star().Ident("_service"))
	handleFnCode.Name("Handle")
	handleFnCode.AddParam("ctx", gcg.QualifiedIdent(gcg.NewPackage("context"), "Context"))
	handleFnCode.AddParam("fn", gcg.String())
	handleFnCode.AddParam("argument", gcg.QualifiedIdent(gcg.NewPackage("github.com/aacfactory/fns/service"), "Param"))
	handleFnCode.AddResult("v", gcg.Token("interface{}"))
	handleFnCode.AddResult("err", gcg.QualifiedIdent(gcg.NewPackage("github.com/aacfactory/errors"), "CodeError"))

	body := gcg.Statements()
	if s.service.Functions != nil && s.service.Functions.Len() > 0 {
		fnSwitchCode := gcg.Switch()
		fnSwitchCode.Expression(gcg.Ident("fn"))
		for _, function := range s.service.Functions {
			functionCode := gcg.Statements()
			// internal
			if function.Internal() {
				functionCode.Token("// check internal").Line()
				functionCode.Token("if !service.CanAccessInternal(ctx) {").Line()
				functionCode.Tab().Token(fmt.Sprintf("err = errors.Warning(\"%s: %s cannot be accessed externally\")", s.service.Name, function.Name())).Line()
				functionCode.Tab().Break().Line()
				functionCode.Symbol("}").Line()
			}
			// authorization
			if function.Authorization() {
				functionCode.Token("// verify authorizations").Line()
				functionCode.Token("err = authorizations.ParseContext(ctx)", gcg.NewPackage("github.com/aacfactory/fns/service/builtin/authorizations")).Line()
				functionCode.Token("if err != nil {").Line()
				functionCode.Tab().Break().Line()
				functionCode.Token("}").Line()
			}
			// permission
			if function.Permission() {
				functionCode.Token("// verify permissions").Line()
				functionCode.Token("err = permissions.EnforceContext(ctx, _name, fn)", gcg.NewPackage("github.com/aacfactory/fns/service/builtin/permissions")).Line()
				functionCode.Token("if err != nil {").Line()
				functionCode.Tab().Break().Line()
				functionCode.Token("}").Line()
			}
			// param
			if function.Param != nil {
				var param gcg.Code = nil
				if s.service.Path == function.Param.Type.Path {
					param = gcg.Ident(function.Param.Type.Name)
				} else {
					pkg, hasPKG := s.service.Imports.Path(function.Param.Type.Path)
					if !hasPKG {
						err = errors.Warning("sources: make service handle function code failed").
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
				functionCode.Token("// param").Line()
				functionCode.Token("param := ").Add(param).Token("{}").Line()
				functionCode.Token("paramErr := argument.As(&param)").Line()
				functionCode.Token("if paramErr != nil {").Line()
				functionCode.Tab().Token(fmt.Sprintf("err = errors.Warning(\"%s: decode request argument failed\").WithCause(paramErr)", s.service.Name)).Line()
				functionCode.Tab().Break().Line()
				functionCode.Token("}").Line()
				// param validation
				if title, has := function.Validation(); has {
					functionCode.Token(fmt.Sprintf("err = validators.ValidateWithErrorTitle(param, \"%s\")", title), gcg.NewPackage("github.com/aacfactory/fns/service/validators")).Line()
					functionCode.Token("if err != nil {").Line()
					functionCode.Tab().Break().Line()
					functionCode.Token("}").Line()
				}
			}
			// timeout
			timeout, hasTimeout, timeoutErr := function.Timeout()
			if timeoutErr != nil {
				timeoutValue := function.Annotations["timeout"]
				err = errors.Warning("sources: make service handle function code failed").
					WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
					WithMeta("function", function.Name()).
					WithCause(errors.Warning("value of @timeout is invalid").WithMeta("timeout", timeoutValue).WithCause(timeoutErr))
				return
			}
			if hasTimeout {
				functionCode.Token("// make timeout context").Line()
				functionCode.Token("var cancel context.CancelFunc = nil").Line()
				functionCode.Token(fmt.Sprintf("ctx, cancel = context.WithTimeout(ctx, time.Duration(%d))", int64(timeout))).Line()
			}
			// exec
			functionExecCode := gcg.Statements()
			// sql
			if db, has := function.SQL(); has {
				db = strings.TrimSpace(db)
				if db == "" {
					err = errors.Warning("sources: make service handle function code failed").
						WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
						WithMeta("function", function.Name()).
						WithCause(errors.Warning("value of @sql is required"))
					return
				}
				functionExecCode.Token("// use sql database").Line()
				functionExecCode.Token(fmt.Sprintf("ctx = sql.WithOptions(ctx, sql.Database(\"%s\"))", db), gcg.NewPackage("github.com/aacfactory/fns-contrib/databases/sql")).Line()
			}
			// transactional
			if function.Transactional() {
				functionExecCode.Token("// sql begin transaction").Line()
				functionExecCode.Token("beginTransactionErr := sql.BeginTransaction(ctx)", gcg.NewPackage("github.com/aacfactory/fns-contrib/databases/sql")).Line()
				functionExecCode.Token("if beginTransactionErr != nil {").Line()
				functionExecCode.Tab().Token(fmt.Sprintf("err = errors.Warning(\"%s: begin sql transaction failed\").WithCause(beginTransactionErr)", s.service.Name)).Line()
				functionExecCode.Tab().Return().Line()
				functionExecCode.Token("}").Line()
			}
			// handle
			functionExecCode.Token("// execute function").Line()
			if function.Param != nil && function.Result != nil {
				functionExecCode.Token(fmt.Sprintf("v, err = %s(ctx, param)", function.Ident)).Line()
			} else if function.Param == nil && function.Result != nil {
				functionExecCode.Token(fmt.Sprintf("v, err = %s(ctx)", function.Ident)).Line()
			} else if function.Param != nil && function.Result == nil {
				functionExecCode.Token(fmt.Sprintf("err = %s(ctx, param)", function.Ident)).Line()
			} else if function.Param == nil && function.Result == nil {
				functionExecCode.Token(fmt.Sprintf("err = %s(ctx)", function.Ident)).Line()
			}
			if function.Transactional() {
				functionExecCode.Token("// sql commit transaction").Line()
				functionExecCode.Token("if err == nil {").Line()
				functionExecCode.Tab().Token("commitTransactionErr := sql.CommitTransaction(ctx)", gcg.NewPackage("github.com/aacfactory/fns-contrib/databases/sql")).Line()
				functionExecCode.Tab().Token("if commitTransactionErr != nil {").Line()
				functionExecCode.Tab().Tab().Token("_ = sql.RollbackTransaction(ctx)", gcg.NewPackage("github.com/aacfactory/fns-contrib/databases/sql")).Line()
				functionExecCode.Tab().Tab().Token(fmt.Sprintf("err = errors.ServiceError(\"%s: commit sql transaction failed\").WithCause(commitTransactionErr)", s.service.Name)).Line()
				functionExecCode.Tab().Tab().Return().Line()
				functionExecCode.Tab().Token("}").Line()
				functionExecCode.Token("}").Line()
			}
			// cache
			// cache control
			cacheControlTTL, hasCacheControl, parseCacheControlErr := function.Cache()
			if parseCacheControlErr != nil {
				cacheValue := function.Annotations["cache"]
				err = errors.Warning("sources: make service handle function code failed").
					WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
					WithMeta("function", function.Name()).
					WithCause(errors.Warning("value of @cache is invalid").WithMeta("cache", cacheValue).WithCause(parseCacheControlErr))
				return
			}
			if hasCacheControl {
				functionExecCode.Token("// cache control").Line()
				functionExecCode.Token(fmt.Sprintf("service.CacheControl(ctx, v, time.Duration(%d))", cacheControlTTL)).Line()
			}

			// barrier
			if function.Barrier() {
				functionCode.Token("// barrier").Line()
				functionCode.Token(fmt.Sprintf("v, err = svc.Barrier(ctx, %s, argument, func() (v interface{}, err errors.CodeError) {", function.ConstIdent)).Line()
				functionCode.Add(functionExecCode)
				functionCode.Tab().Return().Line()
				functionCode.Token("})").Line()
			} else {
				functionCode.Add(functionExecCode)
			}
			if hasTimeout {
				functionCode.Token("// cancel timeout context").Line()
				functionCode.Token("cancel()").Line()
			}
			functionCode.Break()
			fnSwitchCode.Case(gcg.Ident(function.ConstIdent), functionCode)
		}
		notFoundCode := gcg.Statements()
		notFoundCode.Token(fmt.Sprintf("err = errors.Warning(\"%s: fn was not found\").WithMeta(\"service\", _name).WithMeta(\"fn\", fn)", s.service.Name)).Line()
		notFoundCode.Break()
		fnSwitchCode.Default(notFoundCode)
		body.Add(fnSwitchCode.Build()).Return()
	}
	handleFnCode.Body(body)

	code = handleFnCode.Build()
	return
}

func (s *ServiceFile) serviceDocumentCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	docFnCode := gcg.Func()
	docFnCode.Receiver("svc", gcg.Star().Ident("_service"))
	docFnCode.Name("Document")
	docFnCode.AddResult("document", gcg.Star().QualifiedIdent(gcg.NewPackage("github.com/aacfactory/fns/service/documents"), "Document"))
	body := gcg.Statements()
	if !s.service.Internal {
		fnCodes := make([]gcg.Code, 0, 1)
		for _, function := range s.service.Functions {
			if function.Internal() {
				continue
			}
			fnCode := gcg.Statements()
			fnCode.Token("// ").Token(function.Name()).Line()
			fnCode.Token("document.AddFn(").Line()
			fnCode.Tab().Token(fmt.Sprintf("\"%s\", \"%s\", \"%s\",%v, %v,", function.Name(), function.Title(), function.Description(), function.Authorization(), function.Deprecated())).Line()
			if function.Param != nil {
				paramCode, paramCodeErr := mapTypeToFunctionElementCode(ctx, function.Param.Type)
				if paramCodeErr != nil {
					err = errors.Warning("sources: make service document code failed").
						WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
						WithMeta("function", function.Name()).
						WithCause(paramCodeErr)
					return
				}
				fnCode.Add(paramCode).Symbol(",").Line()
			} else {
				fnCode.Tab().Token("nil").Symbol(",").Line()
			}
			if function.Result != nil {
				resultCode, resultCodeErr := mapTypeToFunctionElementCode(ctx, function.Result.Type)
				if resultCodeErr != nil {
					err = errors.Warning("sources: make service document code failed").
						WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
						WithMeta("function", function.Name()).
						WithCause(resultCodeErr)
					return
				}
				fnCode.Add(resultCode).Symbol(",").Line()
			} else {
				fnCode.Tab().Token("nil").Symbol(",").Line()
			}
			fnErrsCode := gcg.Statements()
			fnErrsCode.Token("[]documents.FnError{")
			functionErrors := function.Errors()
			if functionErrors != nil && len(functionErrors) > 0 {
				fnErrsCode.Line()

				for _, functionError := range functionErrors {
					fnErrCode := gcg.Statements()
					fnErrCode.Symbol("{").Line()
					fnErrCode.Tab().Token(fmt.Sprintf("Name_: \"%s\"", functionError.Name)).Symbol(",").Line()
					fnErrCode.Tab().Token("Descriptions_: map[string]string{")
					if functionError.Descriptions != nil && len(functionError.Descriptions) > 0 {
						fnErrCode.Line()
						for dk, dv := range functionError.Descriptions {
							fnErrCode.Tab().Tab().Token(fmt.Sprintf("\"%s\": \"%s\"", dk, dv)).Symbol(",").Line()
						}
					}
					fnErrCode.Tab().Symbol("}").Symbol(",").Line()
					fnErrCode.Symbol("}")
					fnErrsCode.Tab().Add(fnErrCode).Symbol(",").Line()
				}
			}
			fnErrsCode.Symbol("}")
			fnCode.Add(fnErrsCode).Symbol(",").Line()
			fnCode.Token(")").Line()
			fnCodes = append(fnCodes, fnCode)
		}
		if len(fnCodes) > 0 {
			internal := "false"
			if s.service.Internal {
				internal = "true"
			}
			body.Token(fmt.Sprintf("document = documents.New(_name, \"%s\", %s, svc.AppVersion())", s.service.Description, internal), gcg.NewPackage("github.com/aacfactory/fns/service/documents")).Line()
			for _, fnCode := range fnCodes {
				body.Add(fnCode).Line()
			}
		}
	}
	body.Return().Line()
	docFnCode.Body(body)
	code = docFnCode.Build()
	return
}
