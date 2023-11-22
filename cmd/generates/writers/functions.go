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
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"github.com/aacfactory/gcg"
)

func (s *ServiceFile) functionsCode(ctx context.Context) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
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
	cacheCmd, _, hasCache := function.Cache()
	if hasCache {
		if cacheCmd == "get" || cacheCmd == "get-set" {
			body.Tab().Token("// cache get").Line()
			body.Tab().Token("cached, cacheExist, cacheGetErr := caches.Get(ctx, &param)", gcg.NewPackage("github.com/aacfactory/fns/services/caches")).Line()
			body.Tab().Token("if cacheGetErr != nil {").Line()
			body.Tab().Tab().Token("log := logs.Load(ctx)", gcg.NewPackage("github.com/aacfactory/fns/logs")).Line()
			body.Tab().Tab().Token("if log.WarnEnabled() {").Line()
			body.Tab().Tab().Tab().Token("log.Warn().Cause(cacheGetErr).With(\"fns\", \"caches\").Message(\"fns: get cache failed\")").Line()
			body.Tab().Tab().Token("}").Line()
			body.Tab().Token("}").Line()
			body.Tab().Token("if cacheExist {").Line()
			body.Tab().Tab().Token("response := services.NewResponse(cached)").Line()
			body.Tab().Tab().Token("scanErr := response.Scan(&result)").Line()
			body.Tab().Tab().Token("if scanErr == nil {").Line()
			body.Tab().Tab().Tab().Token("return").Line()
			body.Tab().Tab().Token("}").Line()
			body.Tab().Tab().Token("log := logs.Load(ctx)").Line()
			body.Tab().Tab().Token("if log.WarnEnabled() {").Line()
			body.Tab().Tab().Tab().Token("log.Warn().Cause(scanErr).With(\"fns\", \"caches\").Message(\"fns: scan cached value failed\")").Line()
			body.Tab().Tab().Token("}").Line()
			body.Tab().Token("}").Line()
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
		annotationCode, annotationCodeErr := annotationWriter.ProxyBefore(ctx, matchedAnnotations[i].Params, function.Param != nil, function.Result != nil)
		if annotationCodeErr != nil {
			err = errors.Warning("sources: make function proxy code failed").
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
	body.Tab().Token("eps := runtime.Endpoints(ctx)").Line()
	body.Tab().Token(fmt.Sprintf("response, handleErr := eps.Request(ctx, _endpointName, %s, param)", function.ConstIdent)).Line()
	body.Tab().Token("if handleErr != nil {").Line()
	body.Tab().Tab().Token("err = handleErr").Line()
	body.Tab().Tab().Token("return").Line()
	body.Tab().Token("}").Line()
	body.Tab().Token("scanErr := response.Scan(&result)").Line()
	body.Tab().Token("if scanErr != nil {").Line()
	body.Tab().Tab().Token("err = scanErr").Line()
	body.Tab().Tab().Token("return").Line()
	body.Tab().Token("}").Line()

	// annotation writers after
	for i, annotationWriter := range matchedAnnotationWriters {
		annotationCode, annotationCodeErr := annotationWriter.ProxyAfter(ctx, matchedAnnotations[i].Params, function.Param != nil, function.Result != nil)
		if annotationCodeErr != nil {
			err = errors.Warning("sources: make function proxy code failed").
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
	// body <<<
	proxy.Body(body)
	code = proxy.Build()
	return
}

func (s *ServiceFile) functionHandlerCode(ctx context.Context, function *sources.Function) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: service write failed").
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
	if function.Param != nil {
		// param
		var param gcg.Code = nil
		if s.service.Path == function.Param.Type.Path {
			param = gcg.Ident(function.Param.Type.Name)
		} else {
			pkg, hasPKG := s.service.Imports.Path(function.Param.Type.Path)
			if !hasPKG {
				err = errors.Warning("sources: make function handle function code failed").
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
		body.Tab().Token("param := ").Add(param).Token("{}").Line()
		body.Tab().Token("if paramErr := ctx.Param().Scan(&param); paramErr != nil {").Line()
		body.Tab().Tab().Token("err = errors.BadRequest(\"scan params failed\").WithCause(paramErr)").Line()
		body.Tab().Tab().Token("return").Line()
		body.Tab().Token("}").Line()
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
		cacheCmd, _, hasCache := function.Cache()
		if hasCache {
			if cacheCmd == "get" || cacheCmd == "get-set" {
				body.Tab().Token("// cache get").Line()
				body.Tab().Token("cached, cacheExist, cacheGetErr := caches.Get(ctx, &param)", gcg.NewPackage("github.com/aacfactory/fns/services/caches")).Line()
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
			err = errors.Warning("sources: make function proxy code failed").
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
	if function.Param != nil && function.Result != nil {
		cacheCmd, cacheTTL, hasCache := function.Cache()
		if hasCache {
			if cacheCmd == "set" || cacheCmd == "get-set" {
				body.Tab().Token("// cache set").Line()
				body.Tab().Token(fmt.Sprintf("if cacheSetErr := caches.Set(ctx, param, v, %s); cacheSetErr != nil {", cacheTTL), gcg.NewPackage("github.com/aacfactory/fns/services/caches")).Line()
				body.Tab().Tab().Token("log := logs.Load(ctx)", gcg.NewPackage("github.com/aacfactory/fns/logs")).Line()
				body.Tab().Tab().Token("if log.WarnEnabled() {").Line()
				body.Tab().Tab().Tab().Token("log.Warn().Cause(cacheSetErr).With(\"fns\", \"caches\").Message(\"fns: set cache failed\")").Line()
				body.Tab().Tab().Token("}").Line()
				body.Tab().Token("}").Line()
			} else if cacheCmd == "remove" {
				body.Tab().Token("// cache remove").Line()
				body.Tab().Token("if cacheRemoveErr := caches.Remove(ctx, param); cacheRemoveErr != nil {", gcg.NewPackage("github.com/aacfactory/fns/services/caches")).Line()
				body.Tab().Tab().Token("log := logs.Load(ctx)", gcg.NewPackage("github.com/aacfactory/fns/logs")).Line()
				body.Tab().Tab().Token("if log.WarnEnabled() {").Line()
				body.Tab().Tab().Tab().Token("log.Warn().Cause(cacheRemoveErr).With(\"fns\", \"caches\").Message(\"fns: remove cache failed\")").Line()
				body.Tab().Tab().Token("}").Line()
				body.Tab().Token("}").Line()
			}
		}
	}
	// cache control
	if function.Readonly() && !function.Internal() && function.Result != nil {
		maxAge, public, mr, pr, hasCC, ccErr := function.CacheControl()
		if ccErr != nil {
			err = errors.Warning("sources: make function handler code failed").
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
			err = errors.Warning("sources: make function handler code failed").
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
