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

func NewServiceFile(service *sources.Service, annotations FnAnnotationCodeWriters) (file CodeFileWriter) {
	file = &ServiceFile{
		service:     service,
		annotations: annotations,
	}
	return
}

type ServiceFile struct {
	service     *sources.Service
	annotations FnAnnotationCodeWriters
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

	// componentCode
	component, componentErr := s.componentCode(ctx)
	if componentErr != nil {
		err = errors.Warning("sources: code file write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(componentErr)
		return
	}
	file.AddCode(component)

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
	stmt.Add(gcg.Var("_endpointName", gcg.Token(fmt.Sprintf(" = []byte(\"%s\")", s.service.Name))))
	for _, function := range s.service.Functions {
		stmt.Add(gcg.Var(function.ConstIdent, gcg.Token(fmt.Sprintf(" = []byte(\"%s\")", function.Name()))))
	}
	code = stmt.Build()
	return
}

func (s *ServiceFile) componentCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
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
		err = errors.Warning("sources: service write failed").
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
		err = errors.Warning("sources: service write failed").
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
		err = errors.Warning("sources: service write failed").
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
		err = errors.Warning("sources: service write failed").
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
		body.Token(fmt.Sprintf("commons.NewFn(string(%s)", function.ConstIdent))
		body.Token(fmt.Sprintf(", %v", function.Readonly()))
		body.Token(fmt.Sprintf(", %v", function.Internal()))
		body.Token(fmt.Sprintf(", %v", function.Authorization()))
		body.Token(fmt.Sprintf(", %v", function.Permission()))
		body.Token(fmt.Sprintf(", %v", function.Metric()))
		body.Token(fmt.Sprintf(", %v", function.Barrier()))
		body.Token(fmt.Sprintf(", %s))", function.HandlerIdent))
		body.Line()
	}
	body.Tab().Return()
	construct.Body(body)
	code = construct.Build()
	return
}

func (s *ServiceFile) serviceDocumentCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
			WithMeta("kind", "service").WithMeta("service", s.service.Name).WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}
	if s.service.Internal {
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
	for _, fnCode := range fnCodes {
		body.Line().Add(fnCode).Line()
	}
	body.Tab().Return()
	docFnCode.Body(body)
	code = docFnCode.Build()
	return
}
