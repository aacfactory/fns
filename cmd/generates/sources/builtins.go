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
	"fmt"
)

func RegisterBuiltinType(typ *Type) {
	key := fmt.Sprintf("%s.%s", typ.Path, typ.Name)
	builtinTypes[key] = typ
}

var builtinTypes = make(map[string]*Type)

func tryGetBuiltinType(path string, name string) (typ *Type, has bool) {
	typ, has = builtinTypes[fmt.Sprintf("%s.%s", path, name)]
	return
}

func registerBuiltinTypes() {
	// password
	RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "github.com/aacfactory/fns/commons/passwords",
		Name:        "Password",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// time
	RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "time",
		Name:        "Time",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "time",
		Name:        "Duration",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// encoding/json
	RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "encoding/json",
		Name:        "RawMessage",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// github.com/aacfactory/json
	RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "github.com/aacfactory/json",
		Name:        "RawMessage",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "github.com/aacfactory/json",
		Name:        "Date",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "github.com/aacfactory/json",
		Name:        "Time",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	RegisterBuiltinType(&Type{
		Kind:        AnyKind,
		Path:        "github.com/aacfactory/json",
		Name:        "Object",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	RegisterBuiltinType(&Type{
		Kind:        ArrayKind,
		Path:        "github.com/aacfactory/json",
		Name:        "Array",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    []*Type{AnyType},
	})
	// github.com/aacfactory/services
	RegisterBuiltinType(&Type{
		Kind:        BuiltinKind,
		Path:        "github.com/aacfactory/services",
		Name:        "Empty",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// todo full elements github.com/aacfactory/errors
	RegisterBuiltinType(&Type{
		Kind: BuiltinKind,
		Path: "github.com/aacfactory/errors",
		Name: "CodeError",
		Annotations: Annotations{
			NewAnnotation("title", "错误"),
			NewAnnotation("description", "具备代码与跟踪的错误"),
		},
		Paradigms: nil,
		Tags:      nil,
		Elements:  nil,
	})
	// todo move into github.com/aacfactory/fns-contrib/databases/sql
	RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "github.com/aacfactory/fns-contrib/databases/sql",
		Name:        "Date",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// todo move into github.com/aacfactory/fns-contrib/databases/sql
	RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "github.com/aacfactory/fns-contrib/databases/sql",
		Name:        "Time",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// todo move into github.com/aacfactory/fns-contrib/databases/sql/dal
	RegisterBuiltinType(&Type{
		Kind: StructKind,
		Path: "github.com/aacfactory/fns-contrib/databases/sql/dal",
		Name: "Pager",
		Annotations: Annotations{
			NewAnnotation("title", "页"),
			NewAnnotation("description", "分页查询结果"),
		},
		Paradigms: []*TypeParadigm{{
			Name:  "E",
			Types: []*Type{AnyType},
		}},
		Tags: nil,
		Elements: []*Type{
			{
				Kind: StructFieldKind,
				Path: "",
				Name: "No",
				Annotations: Annotations{
					NewAnnotation("title", "页码"),
					NewAnnotation("description", "当前页码"),
				},
				Paradigms: nil,
				Tags:      map[string]string{"json": "no"},
				Elements: []*Type{{
					Kind:        BasicKind,
					Path:        "",
					Name:        "int64",
					Annotations: nil,
					Paradigms:   nil,
					Tags:        nil,
					Elements:    nil,
				}},
			},
			{
				Kind: StructFieldKind,
				Path: "",
				Name: "Num",
				Annotations: Annotations{
					NewAnnotation("title", "总页数"),
					NewAnnotation("description", "总页数"),
				},
				Paradigms: nil,
				Tags:      map[string]string{"json": "num"},
				Elements: []*Type{{
					Kind:        BasicKind,
					Path:        "",
					Name:        "int64",
					Annotations: nil,
					Paradigms:   nil,
					Tags:        nil,
					Elements:    nil,
				}},
			},
			{
				Kind: StructFieldKind,
				Path: "",
				Name: "Total",
				Annotations: Annotations{
					NewAnnotation("title", "总条目数"),
					NewAnnotation("description", "总条目数"),
				},
				Paradigms: nil,
				Tags:      map[string]string{"json": "total"},
				Elements: []*Type{{
					Kind:        BasicKind,
					Path:        "",
					Name:        "int64",
					Annotations: nil,
					Paradigms:   nil,
					Tags:        nil,
					Elements:    nil,
				}},
			},
			{
				Kind: StructFieldKind,
				Path: "",
				Name: "Items",
				Annotations: Annotations{
					NewAnnotation("title", "页条目"),
					NewAnnotation("description", "页条目"),
				},
				Paradigms: nil,
				Tags:      map[string]string{"json": "items"},
				Elements: []*Type{{
					Kind:        ArrayKind,
					Path:        "",
					Name:        "",
					Annotations: nil,
					Paradigms:   nil,
					Tags:        nil,
					Elements: []*Type{{
						Kind:        ParadigmElementKind,
						Path:        "",
						Name:        "E",
						Annotations: nil,
						Paradigms:   nil,
						Tags:        nil,
						Elements:    []*Type{AnyType},
					}},
				}},
			},
		},
	})
}
