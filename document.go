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

package fns

import (
	"fmt"
	"github.com/aacfactory/fns/oas"
	"github.com/aacfactory/json"
	"os"
	"strings"
	"sync"
)

func newDocument(title string, description string, terms string, contact *appContact, license *appLicense, version string, publicAddress string, https bool) (doc *document) {
	scheme := "http"
	if https {
		scheme = "https"
	}
	doc = &document{
		mapOnce:     sync.Once{},
		openAPI:     nil,
		title:       title,
		description: description,
		terms:       terms,
		contact:     contact,
		license:     license,
		host:        publicAddress,
		scheme:      scheme,
		version:     version,
		Services:    make(map[string]*ServiceDocument),
	}
	return
}

type document struct {
	mapOnce     sync.Once
	openAPI     []byte
	title       string
	description string
	terms       string
	contact     *appContact
	license     *appLicense
	host        string
	scheme      string
	version     string
	Services    map[string]*ServiceDocument `json:"services,omitempty"`
}

func (doc *document) addServiceDocument(s *ServiceDocument) {
	if _, has := doc.Services[s.Namespace]; has {
		return
	}
	doc.Services[s.Namespace] = s
}

func (doc *document) mapToOpenApi() (v []byte) {
	doc.mapOnce.Do(func() {
		api := &oas.API{
			Openapi: "3.0.3",
			Info: &oas.Info{
				Title:          doc.title,
				Description:    doc.description,
				TermsOfService: doc.terms,
				Contact:        nil,
				License:        nil,
				Version:        doc.version,
			},
			Servers: []*oas.Server{
				{
					Url: func() string {
						return fmt.Sprintf("%s://%s", doc.scheme, doc.host)
					}(),
					Description: func() string {
						active, _ := os.LookupEnv(activeSystemEnvKey)
						return active
					}(),
				},
			},
			Paths: make(map[string]*oas.Path),
			Components: &oas.Components{
				Schemas:   make(map[string]*oas.Schema),
				Responses: make(map[string]*oas.Response),
			},
			Tags: make([]*oas.Tag, 0, 1),
		}
		// info
		if doc.contact != nil {
			api.Info.SetContact(doc.contact.name, doc.contact.url, doc.contact.email)
		}
		if doc.license != nil {
			api.Info.SetLicense(doc.license.name, doc.license.url)
		}
		// fns schemas
		api.Components.Schemas["fns_CodeError"] = &oas.Schema{
			Key:         "fns_CodeError",
			Title:       "CodeError",
			Description: "Fns Code Error",
			Type:        "object",
			Required:    []string{"id", "code", "name", "message", "stacktrace"},
			Properties: map[string]*oas.Schema{
				"id": {
					Title: "error id",
					Type:  "string",
				},
				"code": {
					Title: "error code",
					Type:  "string",
				},
				"name": {
					Title: "error name",
					Type:  "string",
				},
				"message": {
					Title: "error message",
					Type:  "string",
				},
				"meta": {
					Title:                "error meta",
					Type:                 "object",
					AdditionalProperties: &oas.Schema{Type: "string"},
				},
				"stacktrace": {
					Title: "error stacktrace",
					Type:  "object",
					Properties: map[string]*oas.Schema{
						"fn":   {Type: "string"},
						"file": {Type: "string"},
						"line": {Type: "string"},
					},
				},
				"cause": oas.RefSchema("fns_CodeError"),
			},
		}
		api.Components.Schemas["fns_JsonRawMessage"] = &oas.Schema{
			Key:         "fns_JsonRawMessage",
			Title:       "JsonRawMessage",
			Description: "Json Raw Message",
			Type:        "object",
		}
		api.Components.Schemas["fns_Empty"] = &oas.Schema{
			Key:         "fns_Empty",
			Title:       "Empty",
			Description: "",
			Type:        "object",
		}

		// headers
		authorizationHeaderParams := []*oas.Parameter{
			{
				Name:        "Authorization",
				In:          "header",
				Description: "Authorization Key",
				Required:    true,
			},
		}
		responseHeader := map[string]*oas.Header{
			"X-Fns-Request-Id": {
				Description: "request id",
				Schema: &oas.Schema{
					Type: "string",
				},
			},
			"X-Fns-Latency": {
				Description: "request latency",
				Schema: &oas.Schema{
					Type: "string",
				},
			},
		}
		// responses
		api.Components.Responses["400"] = &oas.Response{
			Description: "***BAD REQUEST***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"application/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["401"] = &oas.Response{
			Description: "***UNAUTHORIZED***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"application/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["403"] = &oas.Response{
			Description: "***FORBIDDEN***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"application/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["404"] = &oas.Response{
			Description: "***NOT FOUND***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"application/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["406"] = &oas.Response{
			Description: "***NOT ACCEPTABLE***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"application/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["408"] = &oas.Response{
			Description: "***TIMEOUT***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"application/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["500"] = &oas.Response{
			Description: "***SERVICE EXECUTE FAILED***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"application/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["501"] = &oas.Response{
			Description: "***SERVICE NOT IMPLEMENTED***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"application/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["503"] = &oas.Response{
			Description: "***SERVICE UNAVAILABLE***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"application/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["555"] = &oas.Response{
			Description: "***WARNING***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"application/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		//
		// range
		for _, service := range doc.Services {
			// tags
			api.Tags = append(api.Tags, &oas.Tag{
				Name:        service.Namespace,
				Description: service.Description,
			})
			// fn
			for _, fn := range service.Fns {
				// path
				path := &oas.Path{
					Post: &oas.Operation{
						OperationId: fmt.Sprintf("%s_%s", service.Namespace, fn.Name),
						Tags:        []string{service.Namespace},
						Summary:     fn.Title,
						Description: fn.Description,
						Deprecated:  fn.Deprecated,
						Parameters: func() []*oas.Parameter {
							if fn.HasAuthorization {
								return authorizationHeaderParams
							}
							return nil
						}(),
						RequestBody: &oas.RequestBody{
							Required:    func() bool { return fn.Argument != nil }(),
							Description: "",
							Content: func() (c map[string]*oas.MediaType) {
								if fn.Argument == nil {
									return
								}
								c = oas.ApplicationJsonContent(fn.Argument.schema())
								return
							}(),
						},
						Responses: map[string]oas.Response{
							"200": {
								Content: func() (c map[string]*oas.MediaType) {
									if fn.Result == nil {
										c = oas.ApplicationJsonContent(oas.RefSchema("fns_Empty"))
										return
									}
									c = oas.ApplicationJsonContent(fn.Result.schema())
									return
								}(),
							},
							"400": {Ref: "#/components/responses/400"},
							"401": {Ref: "#/components/responses/401"},
							"403": {Ref: "#/components/responses/403"},
							"404": {Ref: "#/components/responses/404"},
							"406": {Ref: "#/components/responses/406"},
							"408": {Ref: "#/components/responses/408"},
							"500": {Ref: "#/components/responses/500"},
							"501": {Ref: "#/components/responses/501"},
							"503": {Ref: "#/components/responses/503"},
							"555": {Ref: "#/components/responses/555"},
						},
					},
				}
				api.Paths[fmt.Sprintf("/%s/%s", service.Namespace, fn.Name)] = path
				// schemas
				//if fn.Argument != nil {
				//	api.Components.Schemas[fn.Argument.key()] = fn.Argument.schema()
				//}
				//if fn.Result != nil {
				//	api.Components.Schemas[fn.Result.key()] = fn.Result.schema()
				//}
				/*
					if fn.Argument != nil && fn.Argument.objects() != nil && len(fn.Argument.objects()) > 0 {
						for key, obj := range fn.Argument.objects() {
							if _, has := api.Components.Schemas[key]; has {
								continue
							}
							api.Components.Schemas[key] = obj.schema()
						}
					}
					if fn.Result != nil && fn.Result.objects() != nil && len(fn.Result.objects()) > 0 {
						for key, obj := range fn.Result.objects() {
							if _, has := api.Components.Schemas[key]; has {
								continue
							}
							api.Components.Schemas[key] = obj.schema()
						}
					}
				*/
			}
		}
		p, encodeErr := json.Marshal(api)
		if encodeErr != nil {
			doc.openAPI = []byte(fmt.Sprintf("{\"failed\": \"%s\"}", encodeErr.Error()))
		} else {
			doc.openAPI = p
		}
	})
	v = doc.openAPI
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewServiceDocument(namespace string, description string) *ServiceDocument {
	return &ServiceDocument{
		Namespace:   namespace,
		Description: description,
		Fns:         make(map[string]*FnDocument),
	}
}

type ServiceDocument struct {
	// Namespace
	// as tag
	Namespace string `json:"namespace,omitempty"`
	// Description
	// as description of tag, support markdown
	Description string `json:"description,omitempty"`
	// Fns
	// key: fn
	Fns map[string]*FnDocument `json:"fns,omitempty"`
}

func (doc *ServiceDocument) AddFn(fn *FnDocument) {
	doc.Fns[fn.Name] = fn
}

func (doc *ServiceDocument) objects() (v map[string]*ObjectDocument) {
	v = make(map[string]*ObjectDocument)
	if doc.Fns == nil || len(doc.Fns) == 0 {
		return
	}

	for _, fn := range doc.Fns {
		// argument
		argObjects := fn.Argument.objects()
		if argObjects != nil && len(argObjects) > 0 {
			for k, obj := range argObjects {
				if _, has := v[k]; !has {
					v[k] = obj
				}
			}
		}
		// result
		resultObjects := fn.Result.objects()
		if resultObjects != nil && len(resultObjects) > 0 {
			for k, obj := range resultObjects {
				if _, has := v[k]; !has {
					v[k] = obj
				}
			}
		}
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewFnDocument(name string, title string, description string, hasAuthorization bool, deprecated bool) *FnDocument {
	return &FnDocument{
		Name:             name,
		Title:            title,
		Description:      description,
		HasAuthorization: hasAuthorization,
		Argument:         nil,
		Result:           nil,
		Deprecated:       deprecated,
	}
}

type FnDocument struct {
	Name             string          `json:"name,omitempty"`
	Title            string          `json:"title,omitempty"`
	Description      string          `json:"description,omitempty"`
	HasAuthorization bool            `json:"hasAuthorization,omitempty"`
	Argument         *ObjectDocument `json:"argument,omitempty"`
	Result           *ObjectDocument `json:"result,omitempty"`
	Deprecated       bool            `json:"deprecated,omitempty"`
}

func (doc *FnDocument) SetArgument(v *ObjectDocument) {
	doc.Argument = v
}

func (doc *FnDocument) SetResult(v *ObjectDocument) {
	doc.Result = v
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewObjectDocument(pkg string, name string, typ string, format string, title string, description string) *ObjectDocument {
	return &ObjectDocument{
		Package:     pkg,
		Name:        name,
		Title:       title,
		Description: description,
		Type:        typ,
		Format:      format,
		Enum:        make([]interface{}, 0, 1),
		Required:    false,
		Validation:  "",
		Properties:  make(map[string]*ObjectDocument),
		Additional:  false,
		Deprecated:  false,
	}
}

func StringObjectDocument() *ObjectDocument {
	return NewObjectDocument("builtin", "string", "string", "", "String", "String")
}

func BoolObjectDocument() *ObjectDocument {
	return NewObjectDocument("builtin", "bool", "bool", "", "Bool", "Bool")
}

func IntObjectDocument() *ObjectDocument {
	return Int64ObjectDocument()
}

func Int32ObjectDocument() *ObjectDocument {
	return NewObjectDocument("builtin", "int32", "integer", "int32", "Int32", "Int32")
}

func Int64ObjectDocument() *ObjectDocument {
	return NewObjectDocument("builtin", "int64", "integer", "int64", "Int64", "Int64")
}

func Float32ObjectDocument() *ObjectDocument {
	return NewObjectDocument("builtin", "float32", "number", "float", "Float", "Float")
}

func Float64ObjectDocument() *ObjectDocument {
	return NewObjectDocument("builtin", "float64", "number", "double", "Double", "Double")
}

func DateObjectDocument() *ObjectDocument {
	return NewObjectDocument("builtin", "date", "string", "date", "Date", "Date")
}

func DateTimeObjectDocument() *ObjectDocument {
	return NewObjectDocument("builtin", "datetime", "string", "2006-01-02T15:04:05Z07:00", "Datetime", "Datetime").SetExample("2022-01-10T19:13:07+08:00")
}

func StructObjectDocument(pkg string, name string, title string, description string) *ObjectDocument {
	return NewObjectDocument(pkg, name, "object", "", title, description)
}

func JsonRawObjectDocument() *ObjectDocument {
	v := NewObjectDocument("fns", "JsonRawMessage", "object", "", "Json Raw", "Json Raw Message")
	v.Additional = true
	v.AddProperty("", EmptyObjectDocument())
	return v
}

func EmptyObjectDocument() *ObjectDocument {
	return NewObjectDocument("fns", "Empty", "object", "", "Empty", "Empty Struct")
}

func ArrayObjectDocument(name string, title string, description string, item *ObjectDocument) *ObjectDocument {
	v := NewObjectDocument(item.Package, name, "array", "", title, description)
	v.AddProperty("", item)
	return v
}

func MapObjectDocument(name string, title string, description string, item *ObjectDocument) *ObjectDocument {
	v := NewObjectDocument(item.Package, name, "object", "", title, description)
	v.Additional = true
	v.AddProperty("", item)
	return v
}

type ObjectDocument struct {
	Package     string                     `json:"package,omitempty"`
	Name        string                     `json:"name,omitempty"`
	Title       string                     `json:"title,omitempty"`
	Description string                     `json:"description,omitempty"`
	Type        string                     `json:"type,omitempty"`
	Format      string                     `json:"format,omitempty"`
	Enum        []interface{}              `json:"enum,omitempty"`
	Required    bool                       `json:"required,omitempty"`
	Validation  string                     `json:"validation,omitempty"`
	Properties  map[string]*ObjectDocument `json:"properties,omitempty"`
	Additional  bool                       `json:"additional,omitempty"`
	Deprecated  bool                       `json:"deprecated,omitempty"`
	Example     interface{}                `json:"example,omitempty"`
}

func (obj *ObjectDocument) AsRequired(validation string) *ObjectDocument {
	obj.Required = true
	obj.Validation = validation
	return obj
}

func (obj *ObjectDocument) AsDeprecated() *ObjectDocument {
	obj.Deprecated = true
	return obj
}

func (obj *ObjectDocument) SetValidation(validation string) *ObjectDocument {
	obj.Validation = validation
	return obj
}

func (obj *ObjectDocument) SetTitle(title string) *ObjectDocument {
	obj.Title = title
	return obj
}

func (obj *ObjectDocument) SetDescription(description string) *ObjectDocument {
	obj.Description = description
	return obj
}

func (obj *ObjectDocument) SetExample(example interface{}) *ObjectDocument {
	obj.Example = example
	return obj
}

func (obj *ObjectDocument) SetFormat(format string) *ObjectDocument {
	obj.Format = format
	return obj
}

func (obj *ObjectDocument) AddEnum(v ...interface{}) *ObjectDocument {
	obj.Enum = append(obj.Enum, v...)
	return obj
}

func (obj *ObjectDocument) isEmpty() (ok bool) {
	ok = obj.isObject() && len(obj.Properties) == 0
	return
}

func (obj *ObjectDocument) isBuiltin() (ok bool) {
	ok = obj.Type == "builtin"
	return
}

func (obj *ObjectDocument) isFns() (ok bool) {
	ok = obj.Type == "fns"
	return
}

func (obj *ObjectDocument) isObject() (ok bool) {
	ok = obj.Type == "object"
	return
}

func (obj *ObjectDocument) isArray() (ok bool) {
	ok = obj.Type == "array"
	return
}

func (obj *ObjectDocument) isAdditional() (ok bool) {
	ok = obj.isObject() && obj.Additional
	return
}

func (obj *ObjectDocument) AddProperty(name string, prop *ObjectDocument) *ObjectDocument {
	if obj.isObject() || obj.isArray() || obj.isAdditional() {
		obj.Properties[name] = prop
	}
	return obj
}

func (obj *ObjectDocument) objects() (v map[string]*ObjectDocument) {
	v = make(map[string]*ObjectDocument)
	if !obj.isBuiltin() && !obj.isFns() {
		key := obj.key()
		if _, has := v[key]; !has {
			v[key] = obj
			for _, property := range obj.Properties {
				deps := property.objects()
				if deps != nil && len(deps) > 0 {
					for depKey, dep := range deps {
						if _, hasDep := v[depKey]; !hasDep {
							v[depKey] = dep
						}
					}
				}
			}
		}
	}
	return
}

func (obj *ObjectDocument) key() (v string) {
	v = fmt.Sprintf("%s_%s", strings.ReplaceAll(obj.Package, "/", "."), obj.Name)
	return
}

func (obj *ObjectDocument) schema() (v *oas.Schema) {
	// fns
	if obj.isFns() {
		v = oas.RefSchema(obj.key())
		return
	}
	v = &oas.Schema{
		Key:                  obj.key(),
		Title:                obj.Title,
		Description:          "",
		Type:                 obj.Type,
		Required:             nil,
		Format:               obj.Format,
		Enum:                 obj.Enum,
		Properties:           nil,
		Items:                nil,
		AdditionalProperties: nil,
		Deprecated:           obj.Deprecated,
		Example:              obj.Example,
		Ref:                  "",
	}
	// Description
	description := "### Description" + " "
	description = description + obj.Description + " "
	if obj.Validation != "" {
		description = description + "**Validation**" + " "
		description = description + "`" + obj.Validation + "`" + " "
	}
	if obj.Enum != nil && len(obj.Enum) > 0 {
		description = description + "**Enum**" + " "
		description = description + fmt.Sprintf("%v", obj.Enum) + " "
	}
	v.Description = description
	// builtin
	if obj.isBuiltin() {
		return
	}
	// object
	if obj.isObject() && !obj.isEmpty() {
		required := make([]string, 0, 1)
		v.Properties = make(map[string]*oas.Schema)
		for name, prop := range obj.Properties {
			if prop.Required {
				required = append(required, name)
			}
			v.Properties[name] = prop.schema()
		}
		v.Required = required
		return
	}
	// array
	if obj.isArray() {
		v.Items = obj.Properties[""].schema()
		return
	}
	// map
	if obj.isAdditional() {
		v.AdditionalProperties = obj.Properties[""].schema()
		return
	}
	return
}
