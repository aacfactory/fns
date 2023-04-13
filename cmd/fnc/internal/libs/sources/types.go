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
	"golang.org/x/sync/singleflight"
	"reflect"
	"sync"
)

const (
	BasicKind   = TypeKind(iota + 1) // 基本类型，oas时不需要ref
	BuiltinKind                      // 内置类型，oas时需要ref，但不需要建component
	IdentKind
	InterfaceKind
	StructKind
	StructFieldKind
	PointerKind
	ArrayKind
	MapKind
	AnyKind
	ParadigmKind
	ParadigmElementKind
	ReferenceKind
)

type TypeKind int

func (kind TypeKind) String() string {
	switch kind {
	case BasicKind:
		return "basic"
	case BuiltinKind:
		return "builtin"
	case IdentKind:
		return "ident"
	case InterfaceKind:
		return "interface"
	case StructKind:
		return "struct"
	case StructFieldKind:
		return "struct_field"
	case PointerKind:
		return "pointer"
	case ArrayKind:
		return "array"
	case MapKind:
		return "map"
	case AnyKind:
		return "any"
	case ParadigmKind:
		return "paradigm"
	case ParadigmElementKind:
		return "paradigm_element"
	case ReferenceKind:
		return "reference"
	}
	return "unknown"
}

type TypeParadigm struct {
	Name  string
	Types []*Type
}

func (tp *TypeParadigm) String() (v string) {
	types := ""
	if tp.Types != nil && len(tp.Types) > 0 {
		for _, typ := range tp.Types {
			types = types + "| " + typ.String()
		}
		if types != "" {
			types = types[2:]
		}
	}
	v = fmt.Sprintf("[%s %s]", tp.Name, types)
	return
}

var AnyType = &Type{
	Kind:        AnyKind,
	Path:        "",
	Name:        "",
	Annotations: nil,
	Paradigms:   nil,
	Tags:        nil,
	Elements:    nil,
}

type Type struct {
	Kind            TypeKind
	Path            string
	Name            string
	Annotations     Annotations
	Paradigms       []*TypeParadigm
	Tags            map[string]string
	Elements        []*Type
	ParadigmsPacked *Type
}

func (typ *Type) Flats() (v map[string]*Type) {
	v = make(map[string]*Type)
	switch typ.Kind {
	case BuiltinKind, AnyKind:
		if _, has := v[typ.Key()]; !has {
			v[typ.Key()] = typ
		}
		break
	case IdentKind, PointerKind, ArrayKind:
		if _, has := v[typ.Key()]; !has {
			v[typ.Key()] = typ
			vv := typ.Elements[0].Flats()
			for k, t := range vv {
				if _, has = vv[k]; !has {
					v[k] = t
				}
			}
		}
		break
	case InterfaceKind:
		if _, has := v[typ.Key()]; !has {
			v[typ.Key()] = typ
		}
		break
	case StructKind:
		if _, has := v[typ.Key()]; !has {
			v[typ.Key()] = typ
			if typ.Elements != nil && len(typ.Elements) > 0 {
				for _, element := range typ.Elements {
					vv := element.Elements[0].Flats()
					for k, t := range vv {
						if _, has = vv[k]; !has {
							v[k] = t
						}
					}
				}
			}
		}
		break
	case MapKind:
		if _, has := v[typ.Key()]; !has {
			v[typ.Key()] = typ
			vv := typ.Elements[0].Flats()
			for k, t := range vv {
				if _, has = vv[k]; !has {
					v[k] = t
				}
			}
			vv = typ.Elements[1].Flats()
			for k, t := range vv {
				if _, has = vv[k]; !has {
					v[k] = t
				}
			}
		}
		break
	case ParadigmKind:
		if _, has := v[typ.Key()]; !has {
			v[typ.Key()] = typ
			vv := typ.Elements[0].Flats()
			for k, t := range vv {
				if _, has = vv[k]; !has {
					v[k] = t
				}
			}
			for _, paradigm := range typ.Paradigms {
				for _, pt := range paradigm.Types {
					vv = pt.Flats()
					for k, t := range vv {
						if _, has = vv[k]; !has {
							v[k] = t
						}
					}
				}
			}
		}

		if typ.ParadigmsPacked != nil {
			vv := typ.ParadigmsPacked.Flats()
			for k, t := range vv {
				if _, has := vv[k]; !has {
					v[k] = t
				}
			}
		}
		break
	default:
		break
	}
	return
}

func (typ *Type) String() (v string) {
	if typ.Path != "" && typ.Name != "" {
		v = typ.Key()
		return
	}
	switch typ.Kind {
	case BasicKind:
		v = typ.Name
		break
	case BuiltinKind, IdentKind, InterfaceKind, StructKind, StructFieldKind, PointerKind, ReferenceKind:
		v = typ.Key()
		break
	case ArrayKind:
		v = fmt.Sprintf("[]%s", typ.Elements[0].Key())
		break
	case MapKind:
		v = fmt.Sprintf("map[%s]%s", typ.Elements[0].String(), typ.Elements[1].String())
		break
	case AnyKind:
		v = "any"
		break
	case ParadigmElementKind:
		elements := ""
		for _, element := range typ.Elements {
			elements = elements + "| " + element.String()
		}
		if elements != "" {
			elements = elements[2:]
		}
		v = fmt.Sprintf("[%s %s]", typ.Name, elements)
		break
	default:
		v = typ.Key()
		break
	}
	return
}

func (typ *Type) Key() (key string) {
	key = formatTypeKey(typ.Path, typ.Name)
	return
}

func formatTypeKey(path string, name string) (key string) {
	key = fmt.Sprintf("%s.%s", path, name)
	return
}

func (typ *Type) GetTopPaths() (paths []string) {
	paths = make([]string, 0, 1)
	switch typ.Kind {
	case StructKind:
		paths = append(paths, typ.Path)
	case PointerKind, ArrayKind:
		paths = append(paths, typ.Elements[0].GetTopPaths()...)
		break
	case MapKind:
		paths = append(paths, typ.Elements[0].GetTopPaths()...)
		paths = append(paths, typ.Elements[1].GetTopPaths()...)
		break
	}
	return
}

func (typ *Type) Basic() (name string, ok bool) {
	if typ.Kind == BasicKind {
		name = typ.Name
		ok = true
		return
	}
	if typ.Kind == IdentKind {
		name, ok = typ.Elements[0].Basic()
		return
	}
	return
}

func (typ *Type) Copied() (v *Type) {
	v = &Type{
		Kind:            typ.Kind,
		Path:            typ.Path,
		Name:            typ.Name,
		Annotations:     typ.Annotations,
		Paradigms:       typ.Paradigms,
		Tags:            typ.Tags,
		Elements:        nil,
		ParadigmsPacked: typ.ParadigmsPacked,
	}
	if typ.Elements != nil && len(typ.Elements) > 0 {
		v.Elements = make([]*Type, 0, 1)
		for _, element := range typ.Elements {
			v.Elements = append(v.Elements, element.Copied())
		}
	}
	return
}

func (typ *Type) packParadigms(ctx context.Context) (err error) {
	if typ.ParadigmsPacked != nil {
		return
	}
	switch typ.Kind {
	case IdentKind, PointerKind, ArrayKind:
		if typ.Elements == nil || len(typ.Elements) == 0 {
			err = errors.Warning("element is nil")
			break
		}
		err = typ.Elements[0].packParadigms(ctx)
		break
	case MapKind:
		if typ.Elements == nil || len(typ.Elements) != 2 {
			err = errors.Warning("element is nil or length is not 2")
			break
		}
		err = typ.Elements[0].packParadigms(ctx)
		if err != nil {
			break
		}
		err = typ.Elements[1].packParadigms(ctx)
		if err != nil {
			break
		}
		break
	case StructKind:
		if typ.Paradigms == nil || len(typ.Paradigms) == 0 {
			break
		}
		for _, field := range typ.Elements {
			err = field.packParadigms(ctx)
			if err != nil {
				break
			}
		}
		break
	case StructFieldKind:
		err = typ.Elements[0].packParadigms(ctx)
		if err != nil {
			err = errors.Warning("sources: pack struct field paradigms failed").WithMeta("name", typ.Name).WithCause(err)
			break
		}
		break
	case ParadigmKind:
		var topParadigms []*TypeParadigm
		packing := ctx.Value("packing")
		if packing != nil {
			isPacking := false
			topParadigms, isPacking = packing.([]*TypeParadigm)
			if !isPacking {
				err = errors.Warning("sources: context packing value must be []*TypeParadigm")
				break
			}
		}

		pt := typ.Elements[0]
		paradigms := make([]*TypeParadigm, 0, 1)
		paradigmKeys := ""
		stop := false
		for i, paradigm := range typ.Paradigms {
			if paradigm.Types[0].Kind == ParadigmElementKind {
				if topParadigms == nil || len(topParadigms) == 0 {
					// 其所在类型也是个泛型，不用盒化
					stop = true
					break
				}
				matched := false
				for _, topParadigm := range topParadigms {
					if topParadigm.Name == paradigm.Name {
						paradigm = topParadigm
						matched = true
						break
					}
				}
				if !matched {
					err = errors.Warning("sources: can not found paradigm instance from top paradigm instances")
					break
				}
			}
			paradigms = append(paradigms, &TypeParadigm{
				Name:  pt.Paradigms[i].Name,
				Types: paradigm.Types,
			})
			paradigmKeys = paradigmKeys + "+" + paradigm.Types[0].Key()
		}
		if err != nil {
			break
		}
		if stop {
			break
		}
		packed := pt.Copied()
		err = packed.packParadigms(context.WithValue(ctx, "packing", paradigms))
		if err != nil {
			break
		}
		packed.Name = packed.Name + paradigmKeys
		typ.ParadigmsPacked = packed
		break
	case ParadigmElementKind:
		if typ.ParadigmsPacked != nil {
			break
		}
		var topParadigms []*TypeParadigm
		packing := ctx.Value("packing")
		if packing != nil {
			isPacking := false
			topParadigms, isPacking = packing.([]*TypeParadigm)
			if !isPacking {
				err = errors.Warning("sources: context packing value must be []*TypeParadigm")
				break
			}
		}
		if topParadigms == nil || len(topParadigms) == 0 {
			err = errors.Warning("sources: there is no packing in context")
			break
		}
		packed := false
		for _, paradigm := range topParadigms {
			if paradigm.Name == typ.Name {
				typ.ParadigmsPacked = paradigm.Types[0]
				packed = true
				break
			}
		}
		if !packed {
			err = errors.Warning("sources: pack missed")
		}
		break
	default:
		break
	}
	if err != nil {
		err = errors.Warning("sources: type pack paradigms failed").
			WithMeta("path", typ.Path).
			WithMeta("name", typ.Name).
			WithMeta("kind", typ.Kind.String()).
			WithCause(err)
		return
	}

	return
}

type TypeScope struct {
	Path       string
	Mod        *Module
	Imports    Imports
	GenericDoc string
}

type Types struct {
	values sync.Map
	group  singleflight.Group
}

func (types *Types) parseType(ctx context.Context, spec *ast.TypeSpec, scope *TypeScope) (typ *Type, err error) {
	path := scope.Path
	name := spec.Name.Name

	key := formatTypeKey(path, name)

	processing := ctx.Value(key)
	if processing != nil {
		typ = &Type{
			Kind:        ReferenceKind,
			Path:        path,
			Name:        name,
			Annotations: nil,
			Paradigms:   nil,
			Tags:        nil,
			Elements:    nil,
		}
		return
	}

	result, doErr, _ := types.group.Do(key, func() (v interface{}, err error) {
		stored, loaded := types.values.Load(key)
		if loaded {
			v = stored.(*Type)
			return
		}
		ctx = context.WithValue(ctx, key, "processing")
		var result *Type
		switch spec.Type.(type) {
		case *ast.Ident:
			identType, parseIdentTypeErr := types.parseExpr(ctx, spec.Type, scope)
			if parseIdentTypeErr != nil {
				err = errors.Warning("sources: parse ident type spec failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseIdentTypeErr)
				break
			}
			// annotations
			doc := ""
			if spec.Doc != nil && spec.Doc.Text() != "" {
				doc = spec.Doc.Text()
			} else {
				doc = scope.GenericDoc
			}
			annotations, parseAnnotationsErr := ParseAnnotations(doc)
			if parseAnnotationsErr != nil {
				err = errors.Warning("sources: parse ident type failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseAnnotationsErr)
				return
			}
			result = &Type{
				Kind:        IdentKind,
				Path:        path,
				Name:        name,
				Annotations: annotations,
				Paradigms:   nil,
				Tags:        nil,
				Elements:    []*Type{identType},
			}
			break
		case *ast.StructType:
			result, err = types.parseStructType(ctx, spec, scope)
			break
		case *ast.InterfaceType:
			// annotations
			doc := ""
			if spec.Doc != nil && spec.Doc.Text() != "" {
				doc = spec.Doc.Text()
			} else {
				doc = scope.GenericDoc
			}
			annotations, parseAnnotationsErr := ParseAnnotations(doc)
			if parseAnnotationsErr != nil {
				err = errors.Warning("sources: parse interface type failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseAnnotationsErr)
				return
			}
			result = &Type{
				Kind:        InterfaceKind,
				Path:        path,
				Name:        name,
				Annotations: annotations,
				Paradigms:   nil,
				Tags:        nil,
				Elements:    nil,
			}
			break
		case *ast.ArrayType:
			arrayType := spec.Type.(*ast.ArrayType)
			arrayElementType, parseArrayElementTypeErr := types.parseExpr(ctx, arrayType.Elt, scope)
			if parseArrayElementTypeErr != nil {
				err = errors.Warning("sources: parse array type spec failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseArrayElementTypeErr)
				break
			}
			// annotations
			doc := ""
			if spec.Doc != nil && spec.Doc.Text() != "" {
				doc = spec.Doc.Text()
			} else {
				doc = scope.GenericDoc
			}
			annotations, parseAnnotationsErr := ParseAnnotations(doc)
			if parseAnnotationsErr != nil {
				err = errors.Warning("sources: parse array type failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseAnnotationsErr)
				return
			}
			result = &Type{
				Kind:        ArrayKind,
				Path:        path,
				Name:        name,
				Annotations: annotations,
				Paradigms:   nil,
				Tags:        nil,
				Elements:    []*Type{arrayElementType},
			}
			break
		case *ast.MapType:
			mapType := spec.Type.(*ast.MapType)
			keyElement, parseKeyErr := types.parseExpr(ctx, mapType.Key, scope)
			if parseKeyErr != nil {
				err = errors.Warning("sources: parse map type spec failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseKeyErr)
				break
			}
			if _, basic := keyElement.Basic(); !basic {
				err = errors.Warning("sources: parse map type spec failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(errors.Warning("sources: key kind of map kind field must be basic"))
				break
			}
			valueElement, parseValueErr := types.parseExpr(ctx, mapType.Value, scope)
			if parseValueErr != nil {
				err = errors.Warning("sources: parse map type spec failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseValueErr)
				break
			}
			// annotations
			doc := ""
			if spec.Doc != nil && spec.Doc.Text() != "" {
				doc = spec.Doc.Text()
			} else {
				doc = scope.GenericDoc
			}
			annotations, parseAnnotationsErr := ParseAnnotations(doc)
			if parseAnnotationsErr != nil {
				err = errors.Warning("sources: parse map type failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseAnnotationsErr)
				return
			}
			result = &Type{
				Kind:        MapKind,
				Path:        path,
				Name:        name,
				Annotations: annotations,
				Paradigms:   nil,
				Tags:        nil,
				Elements:    []*Type{keyElement, valueElement},
			}
			break
		case *ast.IndexExpr, *ast.IndexListExpr:
			result, err = types.parseExpr(ctx, spec.Type, scope)
			if err != nil {
				break
			}
			result.Path = path
			result.Name = name
			if result.ParadigmsPacked != nil {
				result.ParadigmsPacked.Path = path
				result.ParadigmsPacked.Name = name
			}
			// annotations
			doc := ""
			if spec.Doc != nil && spec.Doc.Text() != "" {
				doc = spec.Doc.Text()
			} else {
				doc = scope.GenericDoc
			}
			annotations, parseAnnotationsErr := ParseAnnotations(doc)
			if parseAnnotationsErr != nil {
				err = errors.Warning("sources: parse map type failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseAnnotationsErr)
				return
			}
			result.Annotations = annotations
			break
		default:
			err = errors.Warning("sources: unsupported type spec").WithMeta("path", path).WithMeta("name", name).WithMeta("type", reflect.TypeOf(spec.Type).String())
			break
		}
		if err != nil {
			return
		}
		types.values.Store(key, result)
		v = result
		return
	})
	if doErr != nil {
		err = doErr
		return
	}
	typ = result.(*Type)
	return
}

func (types *Types) parseTypeParadigms(ctx context.Context, params *ast.FieldList, scope *TypeScope) (paradigms []*TypeParadigm, err error) {
	paradigms = make([]*TypeParadigm, 0, 1)
	for _, param := range params.List {
		paradigm, paradigmErr := types.parseTypeParadigm(ctx, param, scope)
		if paradigmErr != nil {
			err = paradigmErr
			return
		}
		paradigms = append(paradigms, paradigm)
	}
	return
}

func (types *Types) parseTypeParadigm(ctx context.Context, param *ast.Field, scope *TypeScope) (paradigm *TypeParadigm, err error) {
	if param.Names != nil && len(param.Names) > 1 {
		err = errors.Warning("sources: parse paradigm failed").WithCause(errors.Warning("too many names"))
		return
	}
	name := ""
	if param.Names != nil {
		name = param.Names[0].Name
	}
	paradigm = &TypeParadigm{
		Name:  name,
		Types: make([]*Type, 0, 1),
	}
	if param.Type == nil {
		return
	}

	switch param.Type.(type) {
	case *ast.BinaryExpr:
		exprs := types.parseTypeParadigmBinaryExpr(param.Type.(*ast.BinaryExpr))
		for _, expr := range exprs {
			typ, parseTypeErr := types.parseExpr(ctx, expr, scope)
			if parseTypeErr != nil {
				err = errors.Warning("sources: parse paradigm failed").WithMeta("name", name).WithCause(parseTypeErr)
				return
			}
			paradigm.Types = append(paradigm.Types, typ)
		}
		break
	default:
		typ, parseTypeErr := types.parseExpr(ctx, param.Type, scope)
		if parseTypeErr != nil {
			err = errors.Warning("sources: parse paradigm failed").WithMeta("name", name).WithCause(parseTypeErr)
			return
		}
		paradigm.Types = append(paradigm.Types, typ)
		break
	}
	return
}

func (types *Types) parseTypeParadigmBinaryExpr(bin *ast.BinaryExpr) (exprs []ast.Expr) {
	exprs = make([]ast.Expr, 0, 1)
	xBin, isXBin := bin.X.(*ast.BinaryExpr)
	if isXBin {
		exprs = append(exprs, types.parseTypeParadigmBinaryExpr(xBin)...)
	} else {
		exprs = append(exprs, bin.X)
	}
	yBin, isYBin := bin.Y.(*ast.BinaryExpr)
	if isYBin {
		exprs = append(exprs, types.parseTypeParadigmBinaryExpr(yBin)...)
	} else {
		exprs = append(exprs, bin.Y)
	}
	return
}

func (types *Types) parseExpr(ctx context.Context, x ast.Expr, scope *TypeScope) (typ *Type, err error) {
	switch x.(type) {
	case *ast.Ident:
		expr := x.(*ast.Ident)
		if expr.Obj == nil {
			if expr.Name == "any" {
				typ = AnyType
				break
			}
			isBasic := expr.Name == "string" ||
				expr.Name == "bool" ||
				expr.Name == "int" || expr.Name == "int8" || expr.Name == "int16" || expr.Name == "int32" || expr.Name == "int64" ||
				expr.Name == "uint" || expr.Name == "uint8" || expr.Name == "uint16" || expr.Name == "uint32" || expr.Name == "uint64" ||
				expr.Name == "float32" || expr.Name == "float64" ||
				expr.Name == "complex64" || expr.Name == "complex128" ||
				expr.Name == "byte"
			if isBasic {
				typ = &Type{
					Kind:        BasicKind,
					Path:        "",
					Name:        expr.Name,
					Annotations: Annotations{},
					Paradigms:   make([]*TypeParadigm, 0, 1),
					Elements:    make([]*Type, 0, 1),
				}
				break
			} else {
				typ, err = scope.Mod.ParseType(ctx, scope.Path, expr.Name)
				//err = errors.Warning("sources: kind of ident expr object must be type and decl must not be nil")
				break
			}
		}
		if expr.Obj.Kind != ast.Typ || expr.Obj.Decl == nil {
			err = errors.Warning("sources: kind of ident expr object must be type and decl must not be nil")
			break
		}
		switch expr.Obj.Decl.(type) {
		case *ast.Field:
			// paradigms
			field := expr.Obj.Decl.(*ast.Field)
			paradigm, parseParadigmsErr := types.parseTypeParadigm(ctx, field, scope)
			if parseParadigmsErr != nil {
				err = parseParadigmsErr
				break
			}
			typ = &Type{
				Kind:        ParadigmElementKind,
				Path:        "",
				Name:        paradigm.Name,
				Annotations: nil,
				Paradigms:   nil,
				Tags:        nil,
				Elements:    paradigm.Types,
			}
			break
		case *ast.TypeSpec:
			// type
			spec := expr.Obj.Decl.(*ast.TypeSpec)
			typ, err = scope.Mod.ParseType(ctx, scope.Path, spec.Name.Name)
			break
		default:
			err = errors.Warning("sources: unsupported ident expr object decl").WithMeta("decl", reflect.TypeOf(expr.Obj.Decl).String())
			break
		}
		break
	case *ast.InterfaceType:
		typ = AnyType
		break
	case *ast.SelectorExpr:
		expr := x.(*ast.SelectorExpr)
		ident, isIdent := expr.X.(*ast.Ident)
		if !isIdent {
			err = errors.Warning("sources: x type of selector field must be ident").WithMeta("selector_x", reflect.TypeOf(expr.X).String())
			break
		}
		// path
		importer, hasImporter := scope.Imports.Find(ident.Name)
		if !hasImporter {
			err = errors.Warning("sources: import of selector field was not found").WithMeta("import", ident.Name)
			break
		}
		// name
		name := expr.Sel.Name
		builtin, isBuiltin := tryGetBuiltinType(importer.Path, name)
		if isBuiltin {
			typ = builtin
			break
		}
		// find in mod
		typ, err = scope.Mod.ParseType(ctx, importer.Path, expr.Sel.Name)
		break
	case *ast.StarExpr:
		expr := x.(*ast.StarExpr)
		starElement, parseStarErr := types.parseExpr(ctx, expr.X, scope)
		if parseStarErr != nil {
			err = parseStarErr
			break
		}
		typ = &Type{
			Kind:        PointerKind,
			Path:        "",
			Name:        "",
			Annotations: nil,
			Paradigms:   nil,
			Tags:        nil,
			Elements:    []*Type{starElement},
		}
		break
	case *ast.ArrayType:
		expr := x.(*ast.ArrayType)
		arrayElement, parseArrayErr := types.parseExpr(ctx, expr.Elt, scope)
		if parseArrayErr != nil {
			err = parseArrayErr
			break
		}
		typ = &Type{
			Kind:        ArrayKind,
			Path:        "",
			Name:        "",
			Annotations: nil,
			Paradigms:   nil,
			Tags:        nil,
			Elements:    []*Type{arrayElement},
		}
		break
	case *ast.MapType:
		expr := x.(*ast.MapType)
		keyElement, parseKeyErr := types.parseExpr(ctx, expr.Key, scope)
		if parseKeyErr != nil {
			err = parseKeyErr
			break
		}
		if _, basic := keyElement.Basic(); !basic {
			err = errors.Warning("sources: key kind of map kind field must be basic")
			break
		}
		valueElement, parseValueErr := types.parseExpr(ctx, expr.Value, scope)
		if parseValueErr != nil {
			err = parseValueErr
			break
		}
		typ = &Type{
			Kind:        MapKind,
			Path:        "",
			Name:        "",
			Annotations: nil,
			Paradigms:   nil,
			Tags:        nil,
			Elements:    []*Type{keyElement, valueElement},
		}
		break
	case *ast.IndexExpr:
		expr := x.(*ast.IndexExpr)
		paradigmType, parseParadigmTypeErr := types.parseExpr(ctx, expr.Index, scope)
		if parseParadigmTypeErr != nil {
			err = parseParadigmTypeErr
			break
		}
		paradigms := []*TypeParadigm{{
			Name:  "",
			Types: []*Type{paradigmType},
		}}
		xType, parseXErr := types.parseExpr(ctx, expr.X, scope)
		if parseXErr != nil {
			err = parseXErr
			break
		}
		if xType.Paradigms == nil || len(xType.Paradigms) != len(paradigms) {
			err = errors.Warning("sources: parse index expr failed").WithCause(errors.Warning("sources: invalid paradigms in x type"))
			return
		}
		for i, paradigm := range xType.Paradigms {
			paradigms[i].Name = paradigm.Name
		}
		typ = &Type{
			Kind:            ParadigmKind,
			Path:            "",
			Name:            "",
			Annotations:     nil,
			Paradigms:       paradigms,
			Tags:            nil,
			Elements:        []*Type{xType},
			ParadigmsPacked: nil,
		}
		packErr := typ.packParadigms(ctx)
		if packErr != nil {
			err = packErr
			break
		}
		break
	case *ast.IndexListExpr:
		expr := x.(*ast.IndexListExpr)
		paradigmTypes := make([]*Type, 0, 1)
		for _, index := range expr.Indices {
			paradigmType, parseParadigmTypeErr := types.parseExpr(ctx, index, scope)
			if parseParadigmTypeErr != nil {
				err = parseParadigmTypeErr
				break
			}
			paradigmTypes = append(paradigmTypes, paradigmType)
		}
		paradigms := make([]*TypeParadigm, 0, 1)
		for _, paradigmType := range paradigmTypes {
			paradigms = append(paradigms, &TypeParadigm{
				Name:  "",
				Types: []*Type{paradigmType},
			})
		}
		xType, parseXErr := types.parseExpr(ctx, expr.X, scope)
		if parseXErr != nil {
			err = parseXErr
			break
		}
		if xType.Paradigms == nil || len(xType.Paradigms) != len(paradigms) {
			err = errors.Warning("sources: parse index list expr failed").WithCause(errors.Warning("sources: invalid paradigms in x type"))
			return
		}
		for i, paradigm := range xType.Paradigms {
			paradigms[i].Name = paradigm.Name
		}
		typ = &Type{
			Kind:            ParadigmKind,
			Path:            "",
			Name:            "",
			Annotations:     nil,
			Paradigms:       paradigms,
			Tags:            nil,
			Elements:        []*Type{xType},
			ParadigmsPacked: nil,
		}
		packErr := typ.packParadigms(ctx)
		if packErr != nil {
			err = packErr
			break
		}
		break
	default:
		err = errors.Warning("sources: unsupported field type").WithMeta("type", reflect.TypeOf(x).String())
		return
	}
	return
}

func (types *Types) parseStructType(ctx context.Context, spec *ast.TypeSpec, scope *TypeScope) (typ *Type, err error) {
	path := scope.Path
	name := spec.Name.Name
	st, typeOk := spec.Type.(*ast.StructType)
	if !typeOk {
		err = errors.Warning("sources: parse struct type failed").
			WithMeta("path", path).WithMeta("name", name).
			WithCause(errors.Warning("type of spec is not ast.StructType").WithMeta("type", reflect.TypeOf(spec.Type).String()))
		return
	}
	typ = &Type{
		Kind:        StructKind,
		Path:        path,
		Name:        name,
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	}
	// annotations
	doc := ""
	if spec.Doc != nil && spec.Doc.Text() != "" {
		doc = spec.Doc.Text()
	} else {
		doc = scope.GenericDoc
	}
	annotations, parseAnnotationsErr := ParseAnnotations(doc)
	if parseAnnotationsErr != nil {
		err = errors.Warning("sources: parse struct type failed").
			WithMeta("path", path).WithMeta("name", name).
			WithCause(parseAnnotationsErr)
		return
	}
	typ.Annotations = annotations
	// paradigms
	if spec.TypeParams != nil && spec.TypeParams.NumFields() > 0 {
		paradigms, parseParadigmsErr := types.parseTypeParadigms(ctx, spec.TypeParams, scope)
		if parseParadigmsErr != nil {
			err = errors.Warning("sources: parse struct type failed").
				WithMeta("path", path).WithMeta("name", name).
				WithCause(parseParadigmsErr)
			return
		}
		typ.Paradigms = paradigms
	}
	// elements
	if st.Fields != nil && st.Fields.NumFields() > 0 {
		typ.Elements = make([]*Type, 0, 1)
		for i, field := range st.Fields.List {
			if field.Names != nil && len(field.Names) > 1 {
				err = errors.Warning("sources: parse struct type failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(errors.Warning("sources: too many names of one field")).WithMeta("field_no", fmt.Sprintf("%d", i))
				return
			}
			if field.Names == nil || len(field.Names) == 0 {
				// compose
				if field.Type != nil {
					fieldElementType, parseFieldElementTypeErr := types.parseExpr(ctx, field.Type, scope)
					if parseFieldElementTypeErr != nil {
						err = errors.Warning("sources: parse struct type failed").
							WithMeta("path", path).WithMeta("name", name).
							WithCause(parseFieldElementTypeErr).WithMeta("field_no", fmt.Sprintf("%d", i))
						return
					}
					typ.Elements = append(typ.Elements, &Type{
						Kind:        StructFieldKind,
						Path:        "",
						Name:        "",
						Annotations: nil,
						Paradigms:   nil,
						Tags:        nil,
						Elements:    []*Type{fieldElementType},
					})
				} else {
					err = errors.Warning("sources: parse struct type failed").
						WithMeta("path", path).WithMeta("name", name).
						WithCause(errors.Warning("sources: unsupported field")).WithMeta("field_no", fmt.Sprintf("%d", i))
					return
				}
				return
			}
			if !ast.IsExported(field.Names[0].Name) {
				continue
			}
			ft := &Type{
				Kind:        StructFieldKind,
				Path:        "",
				Name:        "",
				Annotations: nil,
				Paradigms:   nil,
				Tags:        nil,
				Elements:    nil,
			}
			// name
			ft.Name = field.Names[0].Name
			// tag
			if field.Tag != nil && field.Tag.Value != "" {
				ft.Tags = parseFieldTag(field.Tag.Value)
			}
			// annotations
			if field.Doc != nil && field.Doc.Text() != "" {
				fieldAnnotations, parseFieldAnnotationsErr := ParseAnnotations(field.Doc.Text())
				if parseFieldAnnotationsErr != nil {
					err = errors.Warning("sources: parse struct type failed").
						WithMeta("path", path).WithMeta("name", name).
						WithCause(parseFieldAnnotationsErr).
						WithMeta("field_no", fmt.Sprintf("%d", i)).
						WithMeta("field", ft.Name)
					return
				}
				ft.Annotations = fieldAnnotations
			}
			// element
			fieldElementType, parseFieldElementTypeErr := types.parseExpr(ctx, field.Type, scope)
			if parseFieldElementTypeErr != nil {
				err = errors.Warning("sources: parse struct type failed").
					WithMeta("path", path).WithMeta("name", name).
					WithCause(parseFieldElementTypeErr).
					WithMeta("field_no", fmt.Sprintf("%d", i)).
					WithMeta("field", ft.Name)
				return
			}
			ft.Elements = []*Type{fieldElementType}
			typ.Elements = append(typ.Elements, ft)
		}
	}
	return
}

func (types *Types) isContextType(expr ast.Expr, imports Imports) (ok bool) {
	e, isSelector := expr.(*ast.SelectorExpr)
	if !isSelector {
		return
	}
	if e.X == nil {
		return
	}
	ident, isIdent := e.X.(*ast.Ident)
	if !isIdent {
		return
	}
	pkg := ident.Name
	if pkg == "" {
		return
	}
	if e.Sel == nil {
		return
	}
	ok = e.Sel.Name == "Context"
	if !ok {
		return
	}
	importer, has := imports.Find(pkg)
	if !has {
		return
	}
	ok = importer.Path == "context"
	return
}

func (types *Types) isCodeErrorType(expr ast.Expr, imports Imports) (ok bool) {
	e, isSelector := expr.(*ast.SelectorExpr)
	if !isSelector {
		return
	}
	if e.X == nil {
		return
	}
	ident, isIdent := e.X.(*ast.Ident)
	if !isIdent {
		return
	}
	pkg := ident.Name
	if pkg == "" {
		return
	}
	if e.Sel == nil {
		return
	}
	ok = e.Sel.Name == "CodeError"
	if !ok {
		return
	}
	importer, has := imports.Find(pkg)
	if !has {
		return
	}
	ok = importer.Path == "github.com/aacfactory/errors"
	return
}
