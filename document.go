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
	"github.com/go-openapi/spec"
	"strings"
	"sync"
)

func newDocument(info documentInfo, publicAddress string, https bool) (doc *document) {
	scheme := "http"
	if https {
		scheme = "https"
	}
	doc = &document{
		mapOnce:  sync.Once{},
		info:     info,
		host:     publicAddress,
		scheme:   scheme,
		consumes: "application/json",
		produces: "application/json",
		services: make(map[string]*ServiceDocument),
		objects:  make(map[string]*ObjectDocument),
		tags:     make(map[string]*documentTag),
	}
	return
}

type document struct {
	mapOnce  sync.Once
	openAPI  []byte
	info     documentInfo
	host     string
	scheme   string
	consumes string
	produces string
	services map[string]*ServiceDocument
	objects  map[string]*ObjectDocument
	tags     map[string]*documentTag
}

func (doc *document) addServiceDocument(s *ServiceDocument) {
	if _, has := doc.services[s.Namespace]; has {
		return
	}
	doc.services[s.Namespace] = s
	// tags
	doc.tags[s.Namespace] = &documentTag{
		Name:        s.Namespace,
		Description: s.Description,
	}
	// definitions
	objects := s.objects()
	for key, object := range objects {
		if _, has := doc.objects[key]; !has {
			doc.objects[key] = &object
		}
	}
}

func (doc *document) mapToFailedOpenApi(err interface{}) {
	p := `{
	"swagger": "2.0",
    "info": {
        "description": "#description",
        "title": "fns",
    },
	"paths": {
		"/health": {
			"get": {
                "summary": "health check",
                "description": "",
                "operationId": "health_check",
                "produces": [
                    "application/json"
                ],
                "parameters": [],
                "responses": {
                    "200": {
                        "description": "successful operation"
                    }
                }
            }
		}
	}
}`
	p = strings.ReplaceAll(p, "#description", fmt.Sprintf("%v", err))
	doc.openAPI = []byte(p)
	return
}

func (doc *document) mapToOpenApi() (v []byte) {
	doc.mapOnce.Do(func() {
		defer func() {
			if err := recover(); err != nil {
				doc.mapToFailedOpenApi(err)
			}
		}()
		swagger := spec.Swagger{}
		swagger.Swagger = "2.0"
		// info
		swagger.Info = &spec.Info{}
		swagger.Info.Title = doc.info.Title
		swagger.Info.Description = doc.info.Description
		swagger.Info.TermsOfService = doc.info.TermsOfService
		swagger.Info.Version = doc.info.version
		if doc.info.Contact != nil {
			swagger.Info.Contact = &spec.ContactInfo{
				ContactInfoProps: spec.ContactInfoProps{
					Name:  doc.info.Contact.Name,
					URL:   doc.info.Contact.Url,
					Email: doc.info.Contact.Email,
				},
				VendorExtensible: spec.VendorExtensible{},
			}
		}
		if doc.info.License != nil {
			swagger.Info.License = &spec.License{
				LicenseProps: spec.LicenseProps{
					Name: doc.info.License.Name,
					URL:  doc.info.License.Url,
				},
				VendorExtensible: spec.VendorExtensible{},
			}
		}
		// host
		swagger.Host = doc.host
		// basePath
		swagger.BasePath = ""
		// schemes
		swagger.Schemes = []string{doc.scheme}
		// consumes
		swagger.Consumes = []string{doc.consumes}
		// produces
		swagger.Produces = []string{doc.produces}
		// tags
		tags := make([]spec.Tag, 0, 1)
		tags = append(tags, spec.Tag{
			VendorExtensible: spec.VendorExtensible{},
			TagProps: spec.TagProps{
				Description:  "fns group",
				Name:         "fns",
				ExternalDocs: nil,
			},
		})
		for _, tag := range doc.tags {
			tags = append(tags, spec.Tag{
				VendorExtensible: spec.VendorExtensible{},
				TagProps: spec.TagProps{
					Description:  tag.Description,
					Name:         tag.Name,
					ExternalDocs: nil,
				},
			})
		}
		swagger.Tags = tags
		// path
		paths := &spec.Paths{
			VendorExtensible: spec.VendorExtensible{},
			Paths:            make(map[string]spec.PathItem),
		}
		healthCheckPathItem := spec.PathItem{
			Refable:          spec.Refable{},
			VendorExtensible: spec.VendorExtensible{},
			PathItemProps:    spec.PathItemProps{},
		}
		healthCheckPathItem.Get = &spec.Operation{
			VendorExtensible: spec.VendorExtensible{},
			OperationProps: spec.OperationProps{
				Description:  "health check",
				Consumes:     nil,
				Produces:     []string{doc.produces},
				Tags:         []string{"fns"},
				Summary:      "health check",
				ExternalDocs: nil,
				ID:           "fns_health_check",
				Deprecated:   false,
				Security:     nil,
				Parameters:   nil,
				Responses: &spec.Responses{
					VendorExtensible: spec.VendorExtensible{},
					ResponsesProps: spec.ResponsesProps{
						StatusCodeResponses: map[int]spec.Response{
							200: {
								Refable: spec.Refable{},
								ResponseProps: spec.ResponseProps{
									Description: "result",
									Schema: &spec.Schema{
										VendorExtensible: spec.VendorExtensible{},
										SchemaProps: spec.SchemaProps{
											Type: []string{"object"},
											Properties: map[string]spec.Schema{
												"name":    *spec.StringProperty(),
												"version": *spec.StringProperty(),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		paths.Paths[healthCheckPath] = healthCheckPathItem
		for _, serviceDocument := range doc.services {
			for path, fnDocument := range serviceDocument.Fns {
				pathItem := spec.PathItem{
					Refable:          spec.Refable{},
					VendorExtensible: spec.VendorExtensible{},
					PathItemProps:    spec.PathItemProps{},
				}
				parameters := make([]spec.Parameter, 0, 1)
				if fnDocument.HasAuthorization {
					parameters = append(parameters, *spec.HeaderParam("Authorization").Typed("string", "").AsRequired())
				}
				parameters = append(parameters, *spec.BodyParam("body", fnDocument.Argument.mapToSwaggerRequestSchema()).AsRequired())

				pathItem.Post = &spec.Operation{
					VendorExtensible: spec.VendorExtensible{},
					OperationProps: spec.OperationProps{
						Description:  fnDocument.Description,
						Consumes:     []string{doc.consumes},
						Produces:     []string{doc.produces},
						Schemes:      []string{doc.scheme},
						Tags:         []string{serviceDocument.Namespace},
						Summary:      fnDocument.Title,
						ExternalDocs: nil,
						ID:           fmt.Sprintf("%s:%s", serviceDocument.Namespace, fnDocument.Name),
						Deprecated:   fnDocument.Deprecated,
						Security:     nil,
						Parameters:   parameters,
						Responses:    fnDocument.Result.mapResultToOpenApiResponses(),
					},
				}

				paths.Paths[path] = pathItem
			}
		}
		swagger.Paths = paths

		// definitions
		swagger.Definitions = make(map[string]spec.Schema)
		// codeError
		codeErrorDefinition := spec.Schema{}
		codeErrorDefinition.Type = []string{"object"}
		codeErrorDefinition.Properties = spec.SchemaProperties{}
		codeErrorDefinition.Properties["id"] = *spec.StringProperty().WithDescription("")
		codeErrorDefinition.Properties["code"] = *spec.Int32Property().WithDescription("")
		codeErrorDefinition.Properties["name"] = *spec.StringProperty().WithDescription("")
		codeErrorDefinition.Properties["message"] = *spec.StringProperty().WithDescription("")
		codeErrorDefinition.Properties["meta"] = *spec.MapProperty(spec.StringProperty())
		codeErrorStacktraceSchema := spec.Schema{}
		codeErrorStacktraceSchema.Type = []string{"object"}
		codeErrorStacktraceSchema.Properties = spec.SchemaProperties{}
		codeErrorStacktraceSchema.Properties["fn"] = *spec.StringProperty().WithDescription("")
		codeErrorStacktraceSchema.Properties["file"] = *spec.StringProperty().WithDescription("")
		codeErrorStacktraceSchema.Properties["line"] = *spec.Int32Property().WithDescription("")
		codeErrorDefinition.Properties["stacktrace"] = codeErrorStacktraceSchema
		codeErrorDefinition.Properties["cause"] = *spec.RefSchema("#/definitions/fns_errors_CodeError")
		swagger.Definitions["fns_errors_CodeError"] = codeErrorDefinition
		// empty
		emptyDefinition := spec.Schema{}
		emptyDefinition.Description = "empty"
		emptyDefinition.Example = struct{}{}
		emptyDefinition.Type = []string{"object"}
		codeErrorDefinition.Properties = spec.SchemaProperties{}
		swagger.Definitions["fns_empty"] = emptyDefinition
		// raw
		rawDefinition := spec.StringProperty()
		rawDefinition.Description = "raw"
		rawDefinition.Example = "application/json"
		rawDefinition.Type = []string{"string"}
		swagger.Definitions["fns_raw"] = *rawDefinition
		//
		for key, object := range doc.objects {
			swagger.Definitions[key] = *object.mapToSwaggerDefinitionSchema()
		}
		// json
		p, encodeErr := swagger.MarshalJSON()
		if encodeErr != nil {
			panic(fmt.Sprintf("fns: encode open api failed, %v", encodeErr))
			return
		}
		doc.openAPI = p
	})
	v = doc.openAPI
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type documentInfo struct {
	Title          string               `json:"title,omitempty"`
	Description    string               `json:"description,omitempty"`
	TermsOfService string               `json:"termsOfService,omitempty"`
	Contact        *documentInfoContact `json:"contact,omitempty"`
	License        *documentInfoLicense `json:"license,omitempty"`
	version        string
}

type documentInfoContact struct {
	Name  string `json:"name,omitempty"`
	Url   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

type documentInfoLicense struct {
	Name string `json:"name,omitempty"`
	Url  string `json:"url,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type documentTag struct {
	Name        string
	Description string
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewServiceDocument(namespace string, description string) *ServiceDocument {
	return &ServiceDocument{
		Namespace:   namespace,
		Description: description,
		Fns:         make(map[string]FnDocument),
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
	Fns map[string]FnDocument `json:"fns,omitempty"`
}

func (doc *ServiceDocument) AddFn(fn FnDocument) {
	doc.Fns[fn.Name] = fn
}

func (doc *ServiceDocument) objects() (v map[string]ObjectDocument) {
	v = make(map[string]ObjectDocument)
	if doc.Fns == nil || len(doc.Fns) == 0 {
		return
	}

	for _, fn := range doc.Fns {
		// argument
		if !fn.Argument.isSimple() && !fn.Argument.isEmpty() {
			key := fn.Argument.Name
			if _, has := v[key]; !has {
				v[key] = fn.Argument
			}
			deps := fn.Argument.getDeps()
			if len(deps) > 0 {
				for depKey, dep := range deps {
					if _, has := v[depKey]; !has {
						v[depKey] = dep
					}
				}
			}
		}
		// result
		if !fn.Result.isSimple() && !fn.Result.isEmpty() {
			key := fn.Result.Name
			if _, has := v[key]; !has {
				v[key] = fn.Result
			}
			deps := fn.Result.getDeps()
			if len(deps) > 0 {
				for depKey, dep := range deps {
					if _, has := v[depKey]; !has {
						v[depKey] = dep
					}
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
		Argument:         JsonRawObjectDocument(),
		Result:           JsonRawObjectDocument(),
		Deprecated:       deprecated,
	}
}

type FnDocument struct {
	Name             string         `json:"name,omitempty"`
	Title            string         `json:"title,omitempty"`
	Description      string         `json:"description,omitempty"`
	HasAuthorization bool           `json:"hasAuthorization,omitempty"`
	Argument         ObjectDocument `json:"argument,omitempty"`
	Result           ObjectDocument `json:"result,omitempty"`
	Deprecated       bool           `json:"deprecated,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

func JsonRawObjectDocument() *ObjectDocument {
	return &ObjectDocument{
		Package:     "fns",
		Name:        "JsonRawMessage",
		Title:       "Json Raw",
		Description: "Json Raw Message",
		Type:        "additional",
		Properties:  make([]*ObjectPropertyDocument, 0, 1),
	}
}

func EmptyObjectDocument() *ObjectDocument {
	return &ObjectDocument{
		Package:     "fns",
		Name:        "Empty",
		Title:       "Empty",
		Description: "Empty Struct",
		Type:        "object",
		Properties:  make([]*ObjectPropertyDocument, 0, 1),
	}
}

func NewObjectDocument(pkg string, name string, title string, description string) *ObjectDocument {
	return &ObjectDocument{
		Package:     pkg,
		Name:        name,
		Title:       title,
		Description: description,
		Type:        "object",
		Properties:  make([]*ObjectPropertyDocument, 0, 1),
	}
}

func ArrayObjectDocument(name string, title string, description string, item *ObjectPropertyDocument) *ObjectDocument {
	pkg := ""
	if item.Reference != nil {
		pkg = item.Reference.Package
	} else {
		pkg = item.Package
	}
	v := &ObjectDocument{
		Package:     pkg,
		Name:        name,
		Title:       title,
		Description: description,
		Type:        "array",
		Properties:  make([]*ObjectPropertyDocument, 0, 1),
	}
	v.AddProperty(item)
	return v
}

func MapObjectDocument(name string, title string, description string, item *ObjectPropertyDocument) *ObjectDocument {
	pkg := ""
	if item.Reference != nil {
		pkg = item.Reference.Package
	} else {
		pkg = item.Package
	}
	v := &ObjectDocument{
		Package:     pkg,
		Name:        name,
		Title:       title,
		Description: description,
		Type:        "map",
		Properties:  make([]*ObjectPropertyDocument, 0, 1),
	}
	v.AddProperty(item)
	return v
}

type ObjectDocument struct {
	Package     string                    `json:"package,omitempty"`
	Name        string                    `json:"name,omitempty"`
	Title       string                    `json:"title,omitempty"`
	Description string                    `json:"description,omitempty"`
	Type        string                    `json:"type,omitempty"`
	Properties  []*ObjectPropertyDocument `json:"properties,omitempty"`
}

func (obj *ObjectDocument) isEmpty() (ok bool) {
	ok = obj.isObject() && len(obj.Properties) == 0
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

func (obj *ObjectDocument) isMap() (ok bool) {
	ok = obj.Type == "map"
	return
}

func (obj *ObjectDocument) AddProperty(prop *ObjectPropertyDocument) {
	if obj.isObject() {
		obj.Properties = append(obj.Properties, prop)
	}
}

func (obj *ObjectDocument) mapToOpenAPISchemaURI() (v string) {
	v = fmt.Sprintf("#/components/%s", obj.mapToOpenAPIComponentsKey())
	return
}

func (obj *ObjectDocument) mapToOpenAPIComponentsKey() (v string) {
	v = fmt.Sprintf("%s+%s", strings.ReplaceAll(obj.Package, "/", "."), obj.Name)
	return
}

func (obj *ObjectDocument) mapToOpenAPIComponent() (v []byte) {
	// todo
	// v = json bytes
	return

}

func (obj *ObjectDocument) getDeps() (v map[string]*ObjectDocument) {
	v = make(map[string]*ObjectDocument)
	if obj.isEmpty() {
		return
	}
	if obj.Properties == nil || len(obj.Properties) == 0 {
		return
	}
	for _, property := range obj.Properties {
		ref := property.Reference
		if ref == nil {
			continue
		}
		if _, has := v[ref.mapToOpenAPIComponentsKey()]; !has {
			v[ref.mapToOpenAPIComponentsKey()] = ref
		}
		refDeps := ref.getDeps()
		if len(refDeps) > 0 {
			for key, refDep := range refDeps {
				if _, has := v[key]; !has {
					v[key] = refDep
				}
			}
		}
	}
	return
}

func (obj ObjectDocument) mapResultToOpenApiResponses() (v *spec.Responses) {
	// todo: mv to path response
	headers := map[string]spec.Header{
		"Server":           *spec.ResponseHeader().Typed("string", ""),
		"X-Fns-Request-Id": *spec.ResponseHeader().Typed("string", ""),
		"X-Fns-Latency":    *spec.ResponseHeader().Typed("string", ""),
	}

	v = &spec.Responses{}
	v.StatusCodeResponses = make(map[int]spec.Response)
	// 200
	v.StatusCodeResponses[200] = spec.Response{
		Refable: spec.Refable{},
		ResponseProps: spec.ResponseProps{
			Description: "SUCCEED",
			Schema:      spec.RefSchema(fmt.Sprintf("#/definitions/%s", obj.Name)),
			Headers:     headers,
			Examples:    nil,
		},
		VendorExtensible: spec.VendorExtensible{},
	}
	// 400
	v.StatusCodeResponses[400] = spec.Response{
		Refable: spec.Refable{},
		ResponseProps: spec.ResponseProps{
			Description: "BAD REQUEST",
			Schema:      spec.RefSchema("#/definitions/fns_errors_CodeError"),
			Headers:     headers,
			Examples: map[string]interface{}{
				"id":      "id of error",
				"code":    400,
				"name":    "name of error",
				"message": "message of error",
				"meta": map[string]string{
					"foo": "bar",
				},
				"stacktrace": map[string]interface{}{
					"fn":   "fn name",
					"file": "source code file path",
					"line": 10,
				},
				"cause": struct{}{},
			},
		},
		VendorExtensible: spec.VendorExtensible{},
	}
	// 401
	v.StatusCodeResponses[401] = spec.Response{
		Refable: spec.Refable{},
		ResponseProps: spec.ResponseProps{
			Description: "UNAUTHORIZED",
			Schema:      spec.RefSchema("#/definitions/fns_errors_CodeError"),
			Headers:     headers,
			Examples: map[string]interface{}{
				"id":      "id of error",
				"code":    401,
				"name":    "name of error",
				"message": "message of error",
				"meta": map[string]string{
					"foo": "bar",
				},
				"stacktrace": map[string]interface{}{
					"fn":   "fn name",
					"file": "source code file path",
					"line": 10,
				},
				"cause": struct{}{},
			},
		},
		VendorExtensible: spec.VendorExtensible{},
	}
	// 403
	v.StatusCodeResponses[403] = spec.Response{
		Refable: spec.Refable{},
		ResponseProps: spec.ResponseProps{
			Description: "FORBIDDEN",
			Schema:      spec.RefSchema("#/definitions/fns_errors_CodeError"),
			Headers:     headers,
			Examples: map[string]interface{}{
				"id":      "id of error",
				"code":    403,
				"name":    "name of error",
				"message": "message of error",
				"meta": map[string]string{
					"foo": "bar",
				},
				"stacktrace": map[string]interface{}{
					"fn":   "fn name",
					"file": "source code file path",
					"line": 10,
				},
				"cause": struct{}{},
			},
		},
		VendorExtensible: spec.VendorExtensible{},
	}
	// 404
	v.StatusCodeResponses[404] = spec.Response{
		Refable: spec.Refable{},
		ResponseProps: spec.ResponseProps{
			Description: "NOT FOUND",
			Schema:      spec.RefSchema("#/definitions/fns_errors_CodeError"),
			Headers:     headers,
			Examples: map[string]interface{}{
				"id":      "id of error",
				"code":    404,
				"name":    "name of error",
				"message": "message of error",
				"meta": map[string]string{
					"foo": "bar",
				},
				"stacktrace": map[string]interface{}{
					"fn":   "fn name",
					"file": "source code file path",
					"line": 10,
				},
				"cause": struct{}{},
			},
		},
		VendorExtensible: spec.VendorExtensible{},
	}
	// 406
	v.StatusCodeResponses[406] = spec.Response{
		Refable: spec.Refable{},
		ResponseProps: spec.ResponseProps{
			Description: "NOT ACCEPTABLE",
			Schema:      spec.RefSchema("#/definitions/fns_errors_CodeError"),
			Headers:     headers,
			Examples: map[string]interface{}{
				"id":      "id of error",
				"code":    406,
				"name":    "name of error",
				"message": "message of error",
				"meta": map[string]string{
					"foo": "bar",
				},
				"stacktrace": map[string]interface{}{
					"fn":   "fn name",
					"file": "source code file path",
					"line": 10,
				},
				"cause": struct{}{},
			},
		},
		VendorExtensible: spec.VendorExtensible{},
	}
	// 408
	v.StatusCodeResponses[408] = spec.Response{
		Refable: spec.Refable{},
		ResponseProps: spec.ResponseProps{
			Description: "TIMEOUT",
			Schema:      spec.RefSchema("#/definitions/fns_errors_CodeError"),
			Headers:     headers,
			Examples: map[string]interface{}{
				"id":      "id of error",
				"code":    408,
				"name":    "name of error",
				"message": "message of error",
				"meta": map[string]string{
					"foo": "bar",
				},
				"stacktrace": map[string]interface{}{
					"fn":   "fn name",
					"file": "source code file path",
					"line": 10,
				},
				"cause": struct{}{},
			},
		},
		VendorExtensible: spec.VendorExtensible{},
	}
	// 500
	v.StatusCodeResponses[500] = spec.Response{
		Refable: spec.Refable{},
		ResponseProps: spec.ResponseProps{
			Description: "SERVICE EXECUTE FAILED",
			Schema:      spec.RefSchema("#/definitions/fns_errors_CodeError"),
			Headers:     headers,
			Examples: map[string]interface{}{
				"id":      "id of error",
				"code":    500,
				"name":    "name of error",
				"message": "message of error",
				"meta": map[string]string{
					"foo": "bar",
				},
				"stacktrace": map[string]interface{}{
					"fn":   "fn name",
					"file": "source code file path",
					"line": 10,
				},
				"cause": struct{}{},
			},
		},
		VendorExtensible: spec.VendorExtensible{},
	}
	// 501
	v.StatusCodeResponses[501] = spec.Response{
		Refable: spec.Refable{},
		ResponseProps: spec.ResponseProps{
			Description: "SERVICE NOT IMPLEMENTED",
			Schema:      spec.RefSchema("#/definitions/fns_errors_CodeError"),
			Headers:     headers,
			Examples: map[string]interface{}{
				"id":      "id of error",
				"code":    501,
				"name":    "name of error",
				"message": "message of error",
				"meta": map[string]string{
					"foo": "bar",
				},
				"stacktrace": map[string]interface{}{
					"fn":   "fn name",
					"file": "source code file path",
					"line": 10,
				},
				"cause": struct{}{},
			},
		},
		VendorExtensible: spec.VendorExtensible{},
	}
	// 503
	v.StatusCodeResponses[503] = spec.Response{
		Refable: spec.Refable{},
		ResponseProps: spec.ResponseProps{
			Description: "SERVICE UNAVAILABLE",
			Schema:      spec.RefSchema("#/definitions/fns_errors_CodeError"),
			Headers:     headers,
			Examples: map[string]interface{}{
				"id":      "id of error",
				"code":    503,
				"name":    "name of error",
				"message": "message of error",
				"meta": map[string]string{
					"foo": "bar",
				},
				"stacktrace": map[string]interface{}{
					"fn":   "fn name",
					"file": "source code file path",
					"line": 10,
				},
				"cause": struct{}{},
			},
		},
		VendorExtensible: spec.VendorExtensible{},
	}
	// 555
	v.StatusCodeResponses[555] = spec.Response{
		Refable: spec.Refable{},
		ResponseProps: spec.ResponseProps{
			Description: "WARNING",
			Schema:      spec.RefSchema("#/definitions/fns_errors_CodeError"),
			Headers:     headers,
			Examples: map[string]interface{}{
				"id":      "id of error",
				"code":    555,
				"name":    "name of error",
				"message": "message of error",
				"meta": map[string]string{
					"foo": "bar",
				},
				"stacktrace": map[string]interface{}{
					"fn":   "fn name",
					"file": "source code file path",
					"line": 10,
				},
				"cause": struct{}{},
			},
		},
		VendorExtensible: spec.VendorExtensible{},
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func RawObjectPropertyDocument(name string, title string, description string) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "",
		Enum:        nil,
		Type:        "raw",
		Required:    false,
		Reference:   nil,
	}
}

func StringObjectPropertyDocument(name string, title string, description string) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Package:     "builtin",
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "",
		Enum:        nil,
		Type:        "string",
		Required:    false,
		Reference:   nil,
	}
}

func IntObjectPropertyDocument(name string, title string, description string) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Package:     "builtin",
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "",
		Enum:        nil,
		Type:        "int",
		Required:    false,
		Reference:   nil,
	}
}

func Int32ObjectPropertyDocument(name string, title string, description string) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Package:     "builtin",
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "",
		Enum:        nil,
		Type:        "int32",
		Required:    false,
		Reference:   nil,
	}
}

func Int64ObjectPropertyDocument(name string, title string, description string) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Package:     "builtin",
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "",
		Enum:        nil,
		Type:        "int64",
		Required:    false,
		Reference:   nil,
	}
}

func Float32ObjectPropertyDocument(name string, title string, description string) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Package:     "builtin",
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "",
		Enum:        nil,
		Type:        "float32",
		Required:    false,
		Reference:   nil,
	}
}

func Float64ObjectPropertyDocument(name string, title string, description string) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Package:     "builtin",
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "",
		Enum:        nil,
		Type:        "float64",
		Required:    false,
		Reference:   nil,
	}
}

func BoolObjectPropertyDocument(name string, title string, description string) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Package:     "builtin",
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "",
		Enum:        nil,
		Type:        "bool",
		Required:    false,
		Reference:   nil,
	}
}

func DateObjectPropertyDocument(name string, title string, description string) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Package:     "builtin",
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "date",
		Enum:        nil,
		Type:        "string",
		Required:    false,
		Reference:   nil,
	}
}

func DateTimeObjectPropertyDocument(name string, title string, description string) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Package:     "builtin",
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "datetime",
		Enum:        nil,
		Type:        "string",
		Required:    false,
		Reference:   nil,
	}
}

func ArrayObjectPropertyDocument(name string, title string, description string, item ObjectDocument) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "",
		Enum:        nil,
		Type:        "array",
		Required:    false,
		Reference:   &item,
	}
}

func MapObjectPropertyDocument(name string, title string, description string, item ObjectDocument) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "",
		Enum:        nil,
		Type:        "map",
		Required:    false,
		Reference:   &item,
	}
}

func RefObjectPropertyDocument(name string, title string, description string, ref ObjectDocument) ObjectPropertyDocument {
	return ObjectPropertyDocument{
		Name:        name,
		Title:       title,
		Description: description,
		Format:      "",
		Enum:        nil,
		Type:        "object",
		Required:    false,
		Reference:   &ref,
	}
}

type ObjectPropertyDocument struct {
	Package     string          `json:"package,omitempty"`
	Name        string          `json:"name,omitempty"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	Format      string          `json:"format,omitempty"`
	Enum        []interface{}   `json:"enum,omitempty"`
	Type        string          `json:"type,omitempty"`
	Required    bool            `json:"required,omitempty"`
	Validation  string          `json:"validation,omitempty"`
	Reference   *ObjectDocument `json:"reference,omitempty"`
}

func (prop ObjectPropertyDocument) isEmpty() (ok bool) {
	ok = prop.isObject() && prop.Reference == nil
	return
}

func (prop ObjectPropertyDocument) isSimple() (ok bool) {
	ok = !prop.isObject() && !prop.isArray() && !prop.isMap()
	return
}

func (prop ObjectPropertyDocument) isObject() (ok bool) {
	ok = prop.Type == "object" && prop.Reference != nil
	return
}

func (prop ObjectPropertyDocument) isArray() (ok bool) {
	ok = prop.Type == "array"
	return
}

func (prop ObjectPropertyDocument) isMap() (ok bool) {
	ok = prop.Type == "map"
	return
}

func (prop ObjectPropertyDocument) mapToSwaggerDefinitionSchema() (v *spec.Schema) {
	v = &spec.Schema{}
	if prop.isSimple() {
		switch prop.Type {
		case "string":
			v = spec.StringProperty()
		case "int":
			v = spec.Int64Property()
		case "int32":
			v = spec.Int64Property()
		case "int64":
			v = spec.Int64Property()
		case "float32":
			v = spec.Float32Property()
		case "float64":
			v = spec.Float64Property()
		case "bool":
			v = spec.BoolProperty()
		case "date":
			v = spec.DateProperty()
		case "datetime":
			v = spec.DateTimeProperty()
		case "raw":
			v = spec.RefSchema(fmt.Sprintf("#/definitions/fns_raw"))
		default:
			panic(fmt.Sprintf("fns: create swagger schema failed, type(%s) is not supported", prop.Type))
		}
		if prop.Format != "" {
			v.Format = prop.Format
		}
		if prop.Enum != nil && len(prop.Enum) > 0 {
			v.Enum = prop.Enum
		}
	} else if prop.isEmpty() {
		v = spec.RefSchema(fmt.Sprintf("#/definitions/fns_empty"))
	} else if prop.isObject() {
		v = spec.RefSchema(fmt.Sprintf("#/definitions/%s", prop.Name))
	} else if prop.isArray() {
		item := prop.Reference
		v = spec.ArrayProperty(item.mapToSwaggerRequestSchema())
	} else if prop.isMap() {
		if prop.Reference == nil {
			v = spec.MapProperty(spec.StringProperty())
		} else {
			item := prop.Reference
			v = spec.MapProperty(item.mapToSwaggerRequestSchema())
		}
	}
	v.Title = prop.Title
	v.Description = prop.Description
	return
}

func (prop ObjectPropertyDocument) mapToSwaggerRequestSchema() (v *spec.Schema) {
	v = &spec.Schema{}
	if prop.isSimple() {
		switch prop.Type {
		case "string":
			v = spec.StringProperty()
		case "int":
			v = spec.Int64Property()
		case "int32":
			v = spec.Int64Property()
		case "int64":
			v = spec.Int64Property()
		case "float32":
			v = spec.Float32Property()
		case "float64":
			v = spec.Float64Property()
		case "bool":
			v = spec.BoolProperty()
		case "date":
			v = spec.DateProperty()
		case "datetime":
			v = spec.DateTimeProperty()
		case "raw":
			v = spec.RefSchema(fmt.Sprintf("#/definitions/fns_raw"))
		default:
			panic(fmt.Sprintf("fns: create swagger schema failed, type(%s) is not supported", prop.Type))
		}
		if prop.Format != "" {
			v.Format = prop.Format
		}
		if prop.Enum != nil && len(prop.Enum) > 0 {
			v.Enum = prop.Enum
		}
	} else if prop.isEmpty() {
		v = spec.RefSchema(fmt.Sprintf("#/definitions/fns_empty"))
	} else if prop.isObject() {
		v = spec.RefSchema(fmt.Sprintf("#/definitions/%s", prop.Name))
	} else if prop.isArray() {
		item := prop.Reference
		v = spec.ArrayProperty(item.mapToSwaggerRequestSchema())
	} else if prop.isMap() {
		if prop.Reference == nil {
			v = spec.MapProperty(spec.StringProperty())
		} else {
			item := prop.Reference
			v = spec.MapProperty(item.mapToSwaggerRequestSchema())
		}
	}
	return
}
