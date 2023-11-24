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

package sources

func registerBuiltinTypes(mode *Module) {
	// github.com/aacfactory/fns/commons/passwords.password
	mode.RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "github.com/aacfactory/fns/commons/passwords",
		Name:        "Password",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// time.Time
	mode.RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "time",
		Name:        "Time",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// time.Duration
	mode.RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "time",
		Name:        "Duration",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// encoding/json.RawMessage
	mode.RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "encoding/json",
		Name:        "RawMessage",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// github.com/aacfactory/json.RawMessage
	mode.RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "github.com/aacfactory/json",
		Name:        "RawMessage",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// github.com/aacfactory/json.Date
	mode.RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "github.com/aacfactory/json",
		Name:        "Date",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// github.com/aacfactory/json.Time
	mode.RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "github.com/aacfactory/json",
		Name:        "Time",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// github.com/aacfactory/json.Object
	mode.RegisterBuiltinType(&Type{
		Kind:        AnyKind,
		Path:        "github.com/aacfactory/json",
		Name:        "Object",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// github.com/aacfactory/json.Array
	mode.RegisterBuiltinType(&Type{
		Kind:        ArrayKind,
		Path:        "github.com/aacfactory/json",
		Name:        "Array",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    []*Type{AnyType},
	})
	// github.com/aacfactory/services.Empty
	mode.RegisterBuiltinType(&Type{
		Kind:        BuiltinKind,
		Path:        "github.com/aacfactory/services",
		Name:        "Empty",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// github.com/aacfactory/errors.CodeErr
	mode.RegisterBuiltinType(&Type{
		Kind: StructKind,
		Path: "github.com/aacfactory/errors",
		Name: "CodeError",
		Annotations: Annotations{
			NewAnnotation("title", "Code error"),
			NewAnnotation("description", "Errors with code and tracking"),
		},
		Paradigms: nil,
		Tags:      nil,
		Elements: []*Type{
			{
				Kind: StructFieldKind,
				Name: "Id",
				Tags: map[string]string{"json": "id"},
				Elements: []*Type{{
					Kind: BasicKind,
					Name: "string",
				}},
			},
			{
				Kind: StructFieldKind,
				Name: "Code",
				Tags: map[string]string{"json": "code"},
				Elements: []*Type{{
					Kind: BasicKind,
					Name: "int",
				}},
			},
			{
				Kind: StructFieldKind,
				Name: "Name",
				Tags: map[string]string{"json": "name"},
				Elements: []*Type{{
					Kind: BasicKind,
					Name: "string",
				}},
			},
			{
				Kind: StructFieldKind,
				Name: "Message",
				Tags: map[string]string{"json": "message"},
				Elements: []*Type{{
					Kind: BasicKind,
					Name: "string",
				}},
			},
			{
				Kind: StructFieldKind,
				Name: "Meta",
				Tags: map[string]string{"json": "meta"},
				Elements: []*Type{{
					Kind:        MapKind,
					Path:        "",
					Name:        "",
					Annotations: nil,
					Paradigms:   nil,
					Tags:        nil,
					Elements: []*Type{
						{
							Kind: BasicKind,
							Name: "string",
						},
						{
							Kind: BasicKind,
							Name: "string",
						},
					},
				}},
			},
			{
				Kind: StructFieldKind,
				Name: "Stacktrace",
				Tags: map[string]string{"json": "stacktrace"},
				Elements: []*Type{
					{
						Kind: StructKind,
						Path: "github.com/aacfactory/errors",
						Name: "Stacktrace",
						Elements: []*Type{
							{
								Kind: StructFieldKind,
								Name: "Fn",
								Tags: map[string]string{"json": "fn"},
								Elements: []*Type{{
									Kind: BasicKind,
									Name: "string",
								}},
							},
							{
								Kind: StructFieldKind,
								Name: "File",
								Tags: map[string]string{"json": "file"},
								Elements: []*Type{{
									Kind: BasicKind,
									Name: "string",
								}},
							},
							{
								Kind: StructFieldKind,
								Name: "Line",
								Tags: map[string]string{"json": "line"},
								Elements: []*Type{{
									Kind: BasicKind,
									Name: "int",
								}},
							},
						},
					},
				},
			},
			{
				Kind: StructFieldKind,
				Name: "Cause",
				Tags: map[string]string{"json": "cause"},
				Elements: []*Type{{
					Kind: ReferenceKind,
					Path: "github.com/aacfactory/errors",
					Name: "CodeError",
				}},
			},
		},
	})
	// github.com/aacfactory/fns/commons/times.Date
	mode.RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "github.com/aacfactory/fns/commons/times",
		Name:        "Date",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
	// github.com/aacfactory/fns/commons/times.Time
	mode.RegisterBuiltinType(&Type{
		Kind:        BasicKind,
		Path:        "github.com/aacfactory/fns/commons/times",
		Name:        "Time",
		Annotations: nil,
		Paradigms:   nil,
		Tags:        nil,
		Elements:    nil,
	})
}
