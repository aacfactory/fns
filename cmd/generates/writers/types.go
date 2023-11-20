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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"github.com/aacfactory/gcg"
	"strings"
)

func mapTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	switch typ.Kind {
	case sources.BasicKind:
		code, err = mapBasicTypeToFunctionElementCode(ctx, typ)
		break
	case sources.BuiltinKind:
		code, err = mapBuiltinTypeToFunctionElementCode(ctx, typ)
		break
	case sources.IdentKind:
		code, err = mapIdentTypeToFunctionElementCode(ctx, typ)
		break
	case sources.InterfaceKind:
		code, err = mapInterfaceTypeToFunctionElementCode(ctx, typ)
		break
	case sources.StructKind:
		code, err = mapStructTypeToFunctionElementCode(ctx, typ)
		break
	case sources.StructFieldKind:
		code, err = mapStructFieldTypeToFunctionElementCode(ctx, typ)
		break
	case sources.PointerKind:
		code, err = mapPointerTypeToFunctionElementCode(ctx, typ)
		break
	case sources.ArrayKind:
		code, err = mapArrayTypeToFunctionElementCode(ctx, typ)
		break
	case sources.MapKind:
		code, err = mapMapTypeToFunctionElementCode(ctx, typ)
		break
	case sources.AnyKind:
		code, err = mapAnyTypeToFunctionElementCode(ctx, typ)
		break
	case sources.ParadigmKind:
		code, err = mapParadigmTypeToFunctionElementCode(ctx, typ)
		break
	case sources.ParadigmElementKind:
		code, err = mapParadigmElementTypeToFunctionElementCode(ctx, typ)
		break
	case sources.ReferenceKind:
		code, err = mapReferenceTypeToFunctionElementCode(ctx, typ)
		break
	default:
		err = errors.Warning("sources: mapping type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(errors.Warning("unsupported kind"))
		break
	}
	return
}

func mapBasicTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping basic type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Statements()
	switch typ.Name {
	case "string":
		stmt.Token(fmt.Sprintf("documents.String()"))
		break
	case "bool":
		stmt.Token(fmt.Sprintf("documents.Bool()"))
		break
	case "int8", "int16", "int32":
		stmt.Token(fmt.Sprintf("documents.Int32()"))
		break
	case "int", "int64":
		stmt.Token(fmt.Sprintf("documents.Int64()"))
		break
	case "uint8", "byte":
		stmt.Token(fmt.Sprintf("documents.Uint8()"))
		break
	case "uint16", "uint32":
		stmt.Token(fmt.Sprintf("documents.Uint32()"))
		break
	case "uint", "uint64":
		stmt.Token(fmt.Sprintf("documents.Uint64()"))
		break
	case "float32":
		stmt.Token(fmt.Sprintf("documents.Float32()"))
		break
	case "float64":
		stmt.Token(fmt.Sprintf("documents.Float64()"))
		break
	case "complex64":
		stmt.Token(fmt.Sprintf("documents.Complex64()"))
		break
	case "complex128":
		stmt.Token(fmt.Sprintf("documents.Complex128()"))
		break
	default:
		if typ.Path == "time" && typ.Name == "Time" {
			stmt.Token(fmt.Sprintf("documents.DateTime()"))
			break
		}
		if typ.Path == "time" && typ.Name == "Duration" {
			stmt.Token(fmt.Sprintf("documents.Duration()"))
			break
		}
		if typ.Path == "github.com/aacfactory/fns/commons/passwords" && typ.Name == "Password" {
			stmt.Token(fmt.Sprintf("documents.Password()"))
			break
		}
		if typ.Path == "github.com/aacfactory/json" && typ.Name == "Date" {
			stmt.Token(fmt.Sprintf("documents.Date()"))
			break
		}
		if typ.Path == "github.com/aacfactory/json" && typ.Name == "Time" {
			stmt.Token(fmt.Sprintf("documents.Time()"))
			break
		}
		// todo move it to fns/commons/times.Date
		if typ.Path == "github.com/aacfactory/fns-contrib/databases/sql" && typ.Name == "Date" {
			stmt.Token(fmt.Sprintf("documents.Date()"))
			break
		}
		// todo move it to fns/commons/times.Time
		if typ.Path == "github.com/aacfactory/fns-contrib/databases/sql" && typ.Name == "Time" {
			stmt.Token(fmt.Sprintf("documents.Time()"))
			break
		}
		if typ.Path == "encoding/json" && typ.Name == "RawMessage" {
			stmt.Token(fmt.Sprintf("documents.JsonRaw()"))
			break
		}
		if typ.Path == "github.com/aacfactory/json" && typ.Name == "RawMessage" {
			stmt.Token(fmt.Sprintf("documents.JsonRaw()"))
			break
		}
		err = errors.Warning("sources: unsupported basic type").WithMeta("name", typ.Name)
		return
	}
	code = stmt
	return
}

func mapBuiltinTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping builtin type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	code = gcg.Statements().Token("documents.Ref(").Token(fmt.Sprintf("\"%s\",\"%s\"", typ.Path, typ.Name)).Symbol(")")
	return
}

func mapIdentTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping ident type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	targetCode, targetCodeErr := mapTypeToFunctionElementCode(ctx, typ.Elements[0])
	if targetCodeErr != nil {
		err = errors.Warning("sources: mapping ident type to function element code failed").
			WithMeta("name", typ.Name).WithMeta("path", typ.Path).
			WithCause(targetCodeErr)
		return
	}
	code = gcg.Statements().Token("documents.Ident(").Line().
		Token(fmt.Sprintf("\"%s\",\"%s\"", typ.Path, typ.Name)).Symbol(",").Line().
		Add(targetCode).Symbol(",").Line().
		Symbol(")")
	return
}

func mapInterfaceTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping interface type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	stmt := gcg.Statements()
	stmt = stmt.Token("documents.Struct(").Token(fmt.Sprintf("\"%s\",\"%s\"", typ.Path, typ.Name)).Symbol(")")
	title, hasTitle := typ.Annotations.Get("title")
	if hasTitle {
		stmt = stmt.Dot().Line().Token("SetTitle(").Token(fmt.Sprintf("\"%s\"", title)).Symbol(")")
	}
	description, hasDescription := typ.Annotations.Get("description")
	if hasDescription {
		stmt = stmt.Dot().Line().Token("SetDescription(").Token(fmt.Sprintf("\"%s\"", description)).Symbol(")")
	}
	_, hasDeprecated := typ.Annotations.Get("deprecated")
	if hasDeprecated {
		stmt = stmt.Dot().Line().Token("AsDeprecated()")
	}
	code = stmt
	return
}

func mapPointerTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping pointer type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	code, err = mapTypeToFunctionElementCode(ctx, typ.Elements[0])
	return
}

func mapStructTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping struct type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	if typ.ParadigmsPacked != nil {
		typ = typ.ParadigmsPacked
	}
	stmt := gcg.Statements()
	stmt = stmt.Token("documents.Struct(").Token(fmt.Sprintf("\"%s\",\"%s\"", typ.Path, typ.Name)).Symbol(")")
	title, hasTitle := typ.Annotations.Get("title")
	if hasTitle {
		stmt = stmt.Dot().Line().Token("SetTitle(").Token(fmt.Sprintf("\"%s\"", title)).Symbol(")")
	}
	description, hasDescription := typ.Annotations.Get("description")
	if hasDescription {
		stmt = stmt.Dot().Line().Token("SetDescription(").Token(fmt.Sprintf("\"%s\"", description)).Symbol(")")
	}
	_, hasDeprecated := typ.Annotations.Get("deprecated")
	if hasDeprecated {
		stmt = stmt.Dot().Line().Token("AsDeprecated()")
	}
	for _, field := range typ.Elements {
		name, hasName := field.Tags["json"]
		if !hasName {
			name = field.Name
		}
		if name == "-" {
			continue
		}
		fieldCode, fieldCodeErr := mapTypeToFunctionElementCode(ctx, field)
		if fieldCodeErr != nil {
			err = errors.Warning("sources: mapping struct type to function element code failed").
				WithMeta("name", typ.Name).WithMeta("path", typ.Path).
				WithMeta("field", typ.Name).
				WithCause(fieldCodeErr)
			return
		}
		stmt = stmt.Dot().Line().
			Token("AddProperty(").Line().
			Token(fmt.Sprintf("\"%s\"", name)).Symbol(",").Line().
			Add(fieldCode).Symbol(",").Line().
			Symbol(")")
	}
	code = stmt
	return
}

func mapStructFieldTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping struct field type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	elementCode, elementCodeErr := mapTypeToFunctionElementCode(ctx, typ.Elements[0])
	if elementCodeErr != nil {
		err = errors.Warning("sources: mapping struct field type to function element code failed").
			WithMeta("name", typ.Name).WithMeta("path", typ.Path).
			WithMeta("field", typ.Name).
			WithCause(elementCodeErr)
		return
	}
	stmt := elementCode.(*gcg.Statement)
	fieldTitle, hasFieldTitle := typ.Annotations.Get("title")
	if hasFieldTitle {
		stmt = stmt.Dot().Line().Token("SetTitle(").Token(fmt.Sprintf("\"%s\"", fieldTitle)).Symbol(")")
	}
	fieldDescription, hasFieldDescription := typ.Annotations.Get("description")
	if hasFieldDescription {
		stmt = stmt.Dot().Line().Token("SetDescription(").Token(fmt.Sprintf("\"%s\"", fieldDescription)).Symbol(")")
	}
	_, hasFieldDeprecated := typ.Annotations.Get("deprecated")
	if hasFieldDeprecated {
		stmt = stmt.Dot().Line().Token("AsDeprecated()")
	}
	// password
	_, hasFieldPassword := typ.Annotations.Get("password")
	if hasFieldPassword {
		stmt = stmt.Dot().Line().Token("AsPassword()")
	}
	// enum
	fieldEnum, hasFieldEnum := typ.Annotations["enum"]
	if hasFieldEnum && fieldEnum != "" {
		fieldEnums := strings.Split(fieldEnum, ",")
		fieldEnumsCodeToken := ""
		for _, enumValue := range fieldEnums {
			fieldEnumsCodeToken = fieldEnumsCodeToken + `, "` + strings.TrimSpace(enumValue) + `"`
		}
		if fieldEnumsCodeToken != "" {
			fieldEnumsCodeToken = fieldEnumsCodeToken[2:]
			stmt = stmt.Dot().Line().Token("AddEnum").Symbol("(").Token(fieldEnumsCodeToken).Symbol(")")
		}
	}
	// validation
	fieldValidate, hasFieldValidate := typ.Tags["validate"]
	if hasFieldValidate && fieldValidate != "" {
		fieldRequired := strings.Contains(fieldValidate, "required")
		if fieldRequired {
			stmt = stmt.Dot().Line().Token("AsRequired()")
		}
		fieldValidateMessage, hasFieldValidateMessage := typ.Tags["validate-message"]
		if !hasFieldValidateMessage {
			fieldValidateMessage = typ.Tags["message"]
		}
		fieldValidateMessageI18ns := make([]string, 0, 1)
		validateMessageI18n, hasValidateMessageI18n := typ.Annotations.Get("validate-message-i18n")
		if hasValidateMessageI18n && validateMessageI18n != "" {
			reader := bufio.NewReader(bytes.NewReader([]byte(validateMessageI18n)))
			for {
				line, _, readErr := reader.ReadLine()
				if readErr != nil {
					break
				}
				idx := bytes.IndexByte(line, ':')
				if idx > 0 && idx < len(line) {
					fieldValidateMessageI18ns = append(fieldValidateMessageI18ns, strings.TrimSpace(string(line[0:idx])))
					fieldValidateMessageI18ns = append(fieldValidateMessageI18ns, strings.TrimSpace(string(line[idx+1:])))
				}
			}
		}
		fieldValidateMessageI18nsCodeToken := ""
		for _, fieldValidateMessageI18n := range fieldValidateMessageI18ns {
			fieldValidateMessageI18nsCodeToken = fieldValidateMessageI18nsCodeToken + `, "` + fieldValidateMessageI18n + `"`
		}
		if fieldValidateMessageI18nsCodeToken != "" {
			fieldValidateMessageI18nsCodeToken = fieldValidateMessageI18nsCodeToken[2:]
		}
		stmt = stmt.Dot().Line().Token("SetValidation(").
			Token("documents.NewElementValidation(").
			Token(fmt.Sprintf("\"%s\"", fieldValidateMessage)).
			Symbol(", ").
			Token(fieldValidateMessageI18nsCodeToken).
			Symbol(")").
			Symbol(")")
	}
	code = stmt
	return
}

func mapArrayTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping array type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	element := typ.Elements[0]
	name, isBasic := element.Basic()
	if isBasic && (name == "byte" || name == "uint8") {
		stmt := gcg.Statements()
		stmt = stmt.Token(fmt.Sprintf("documents.Bytes()"))
		code = stmt
		return
	}
	elementCode, elementCodeErr := mapTypeToFunctionElementCode(ctx, element)
	if elementCodeErr != nil {
		err = errors.Warning("sources: mapping array type to function element code failed").
			WithMeta("name", typ.Name).WithMeta("path", typ.Path).
			WithCause(elementCodeErr)
		return
	}
	stmt := gcg.Statements()
	stmt = stmt.Token("documents.Array(").Add(elementCode).Symbol(")")
	if typ.Path != "" && typ.Name != "" {
		stmt = stmt.Dot().Line().Token(fmt.Sprintf("SetPath(\"%s\")", typ.Path))
		stmt = stmt.Dot().Line().Token(fmt.Sprintf("SetName(\"%s\")", typ.Name))
	}
	title, hasTitle := typ.Annotations.Get("title")
	if hasTitle {
		stmt = stmt.Dot().Line().Token("SetTitle(").Token(fmt.Sprintf("\"%s\"", title)).Symbol(")")
	}
	description, hasDescription := typ.Annotations.Get("description")
	if hasDescription {
		stmt = stmt.Dot().Line().Token("SetDescription(").Token(fmt.Sprintf("\"%s\"", description)).Symbol(")")
	}
	_, hasDeprecated := typ.Annotations.Get("deprecated")
	if hasDeprecated {
		stmt = stmt.Dot().Line().Token("AsDeprecated()")
	}
	code = stmt
	return
}

func mapMapTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping map type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	element := typ.Elements[1]
	elementCode, elementCodeErr := mapTypeToFunctionElementCode(ctx, element)
	if elementCodeErr != nil {
		err = errors.Warning("sources: mapping map type to function element code failed").
			WithMeta("name", typ.Name).WithMeta("path", typ.Path).
			WithCause(elementCodeErr)
		return
	}
	stmt := gcg.Statements()
	stmt = stmt.Token("documents.Map(").Add(elementCode).Symbol(")")
	if typ.Path != "" && typ.Name != "" {
		stmt = stmt.Dot().Line().Token(fmt.Sprintf("SetPath(\"%s\")", typ.Path))
		stmt = stmt.Dot().Line().Token(fmt.Sprintf("SetName(\"%s\")", typ.Name))
	}
	title, hasTitle := typ.Annotations.Get("title")
	if hasTitle {
		stmt = stmt.Dot().Line().Token("SetTitle(").Token(fmt.Sprintf("\"%s\"", title)).Symbol(")")
	}
	description, hasDescription := typ.Annotations.Get("description")
	if hasDescription {
		stmt = stmt.Dot().Line().Token("SetDescription(").Token(fmt.Sprintf("\"%s\"", description)).Symbol(")")
	}
	_, hasDeprecated := typ.Annotations.Get("deprecated")
	if hasDeprecated {
		stmt = stmt.Dot().Line().Token("AsDeprecated()")
	}
	code = stmt
	return
}

func mapAnyTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping any type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	code = gcg.Statements().Token("documents.Any()")
	return
}

func mapParadigmTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping paradigm type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	code, err = mapTypeToFunctionElementCode(ctx, typ.ParadigmsPacked)
	if err != nil {
		err = errors.Warning("sources: mapping paradigm type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(err)
		return
	}
	return
}

func mapParadigmElementTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping paradigm element type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	code, err = mapTypeToFunctionElementCode(ctx, typ.ParadigmsPacked)
	if err != nil {
		err = errors.Warning("sources: mapping paradigm element type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(err)
		return
	}
	return
}

func mapReferenceTypeToFunctionElementCode(ctx context.Context, typ *sources.Type) (code gcg.Code, err error) {
	if ctx.Err() != nil {
		err = errors.Warning("sources: mapping reference type to function document element code failed").
			WithMeta("path", typ.Path).WithMeta("name", typ.Name).WithMeta("kind", typ.Kind.String()).
			WithCause(ctx.Err())
		return
	}
	code = gcg.Statements().Token(fmt.Sprintf("documents.Ref(\"%s\", \"%s\")", typ.Path, typ.Name))
	return
}
