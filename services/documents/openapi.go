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

package documents

import (
	"fmt"
	"github.com/aacfactory/fns/commons/oas"
	"sort"
	"strings"
)

func Openapi(title string, description string, term string, openapiVersion string, document Document) (api oas.API) {
	if openapiVersion == "" {
		openapiVersion = "3.1.0"
	}
	// oas
	api = oas.API{
		Openapi: openapiVersion,
		Info: &oas.Info{
			Title:          title,
			Description:    description,
			TermsOfService: term,
			Contact:        nil,
			License:        nil,
			Version:        document.Version.String(),
		},
		Servers: []*oas.Server{},
		Paths:   make(map[string]*oas.Path),
		Components: &oas.Components{
			Schemas:   make(map[string]*oas.Schema),
			Responses: make(map[string]*oas.Response),
		},
		Tags: make([]*oas.Tag, 0, 1),
	}
	// schemas
	codeErr := codeErrOpenapiSchema()
	api.Components.Schemas[codeErr.Key] = codeErr
	jsr := jsonRawMessageOpenapiSchema()
	api.Components.Schemas[jsr.Key] = jsr
	empty := emptyOpenapiSchema()
	api.Components.Schemas[empty.Key] = empty

	for status, response := range responseStatusOpenapi() {
		api.Components.Responses[status] = response
	}
	api.Tags = append(api.Tags, &oas.Tag{
		Name:        "builtin",
		Description: "fns builtins",
	})
	healthURI, healthPathSchema := healthPath()
	api.Paths[healthURI] = healthPathSchema

	// documents
	endpoints := document.Endpoints
	if endpoints != nil || len(endpoints) > 0 {
		for _, endpoint := range endpoints {
			if !endpoint.Defined() {
				continue
			}
			// tags
			api.Tags = append(api.Tags, &oas.Tag{
				Name:        endpoint.Name,
				Description: endpoint.Description,
			})
			// doc
			if endpoint.Elements != nil {
				for _, element := range endpoint.Elements {
					if _, hasElement := api.Components.Schemas[element.Key()]; !hasElement {
						api.Components.Schemas[element.Key()] = ElementSchema(element)
					}
				}
			}
			for _, fn := range endpoint.Functions {
				description := fn.Description
				if fn.Errors != nil && len(fn.Errors) > 0 {
					description = description + "\n----------\n"
					description = description + "Errors:\n"
					for _, errorDocument := range fn.Errors {
						description = description + "* " + errorDocument.Name + "\n"
						i18nKeys := make([]string, 0, 1)
						for _, i18nKey := range errorDocument.Descriptions {
							i18nKeys = append(i18nKeys, i18nKey.Name)
						}
						sort.Strings(i18nKeys)
						for _, i18nKey := range i18nKeys {
							i18nVal, hasI18nValue := errorDocument.Descriptions.Get(i18nKey)
							if hasI18nValue {
								description = description + "\t* " + i18nKey + ": " + i18nVal + "\n"
							}
						}
					}
				}
				path := &oas.Path{
					Post: &oas.Operation{
						OperationId: fmt.Sprintf("%s_%s", endpoint.Name, fn.Name),
						Tags:        []string{endpoint.Name},
						Summary:     fn.Title,
						Description: description,
						Deprecated:  fn.Deprecated,
						Parameters: func() []*oas.Parameter {
							params := requestHeadersOpenapiParams()
							if fn.Authorization {
								params = append(params, requestAuthHeadersOpenapiParams()...)
								return params
							}
							return params
						}(),
						RequestBody: &oas.RequestBody{
							Required:    func() bool { return fn.Param.Exist() }(),
							Description: "",
							Content: func() (c map[string]*oas.MediaType) {
								if !fn.Param.Exist() {
									return
								}
								c = oas.ApplicationJsonContent(ElementSchema(fn.Param))
								return
							}(),
						},
						Responses: map[string]oas.Response{
							"200": {
								Content: func() (c map[string]*oas.MediaType) {
									if !fn.Result.Exist() {
										c = oas.ApplicationJsonContent(oas.RefSchema("github.com/aacfactory/fns/service.Empty"))
										return
									}
									c = oas.ApplicationJsonContent(ElementSchema(fn.Result))
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
				api.Paths[fmt.Sprintf("/%s/%s", endpoint.Name, fn.Name)] = path
			}
		}
	}
	return
}

func codeErrOpenapiSchema() *oas.Schema {
	return &oas.Schema{
		Key:         "github.com/aacfactory/errors.CodeError",
		Title:       "CodeError",
		Description: "Fns Code Error",
		Type:        "object",
		Required:    []string{"id", "code", "name", "message", "stacktrace"},
		Properties: map[string]*oas.Schema{
			"id": {
				Title: "Id",
				Type:  "string",
			},
			"code": {
				Title: "Code",
				Type:  "string",
			},
			"name": {
				Title: "Name",
				Type:  "string",
			},
			"message": {
				Title: "Message",
				Type:  "string",
			},
			"meta": {
				Title:                "Meta",
				Type:                 "object",
				AdditionalProperties: &oas.Schema{Type: "string"},
			},
			"stacktrace": {
				Title: "Stacktrace",
				Type:  "object",
				Properties: map[string]*oas.Schema{
					"fn":   {Type: "string"},
					"file": {Type: "string"},
					"line": {Type: "string"},
				},
			},
			"cause": oas.RefSchema("github.com/aacfactory/errors.CodeError"),
		},
	}
}

func jsonRawMessageOpenapiSchema() *oas.Schema {
	return &oas.Schema{
		Key:         "github.com/aacfactory/json.RawMessage",
		Title:       "JsonRawMessage",
		Description: "Json Raw Message",
		Type:        "object",
	}
}

func emptyOpenapiSchema() *oas.Schema {
	return &oas.Schema{
		Key:         "github.com/aacfactory/fns/service.Empty",
		Title:       "Empty",
		Description: "Empty Object",
		Type:        "object",
	}
}

func requestAuthHeadersOpenapiParams() []*oas.Parameter {
	return []*oas.Parameter{
		{
			Name:        "Authorization",
			In:          "header",
			Description: "Authorization Key",
			Required:    true,
		},
	}
}

func requestHeadersOpenapiParams() []*oas.Parameter {
	return []*oas.Parameter{
		{
			Name:        "X-Fns-Device-Id",
			In:          "header",
			Description: "Client device uuid",
			Required:    true,
		},
		{
			Name:        "X-Fns-Device-Ip",
			In:          "header",
			Description: "Client device ip",
			Required:    false,
		},
		{
			Name:        "X-Fns-Request-Timeout",
			In:          "header",
			Description: "request timeout(Millisecond)",
			Required:    false,
		},
		{
			Name:        "X-Fns-Request-Id",
			In:          "header",
			Description: "request id",
			Required:    false,
		},
		{
			Name:        "X-Fns-Request-Version",
			In:          "header",
			Description: "Applicable version range, e.g.: endpointName1=v0.0.1:v1.0.0, endpointName2=v0.0.1:v1.0.0, ...",
			Required:    false,
		},
	}
}

func responseHeadersOpenapi() map[string]*oas.Header {
	return map[string]*oas.Header{
		"X-Fns-Endpoint-Id": {
			Description: "endpoint id",
			Schema: &oas.Schema{
				Type: "string",
			},
		},
		"X-Fns-Endpoint-Version": {
			Description: "app version",
			Schema: &oas.Schema{
				Type: "string",
			},
		},
		"X-Fns-Handle-Latency": {
			Description: "request latency",
			Schema: &oas.Schema{
				Type: "string",
			},
		},
	}
}

func responseStatusOpenapi() map[string]*oas.Response {
	return map[string]*oas.Response{
		"400": {
			Description: "***BAD REQUEST***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"401": {
			Description: "***UNAUTHORIZED***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"403": {
			Description: "***FORBIDDEN***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"404": {
			Description: "***NOT FOUND***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"406": {
			Description: "***NOT ACCEPTABLE***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"408": {
			Description: "***TIMEOUT***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"500": {
			Description: "***SERVICE EXECUTE FAILED***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"501": {
			Description: "***SERVICE NOT IMPLEMENTED***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"503": {
			Description: "***SERVICE UNAVAILABLE***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"555": {
			Description: "***WARNING***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
	}
}

func healthPath() (uri string, path *oas.Path) {
	uri = "/application/health"
	path = &oas.Path{
		Get: &oas.Operation{
			OperationId: "application_health",
			Tags:        []string{"builtin"},
			Summary:     "Health Check",
			Description: "Check fns system health status",
			Deprecated:  false,
			Parameters:  nil,
			RequestBody: nil,
			Responses: map[string]oas.Response{
				"200": {
					Content: func() (c map[string]*oas.MediaType) {
						schema := &oas.Schema{
							Key:         "github.com/aacfactory/fns/handlers.Health",
							Title:       "Health Check Result",
							Description: "",
							Type:        "object",
							Required:    []string{"name", "id", "version", "running", "now"},
							Properties: map[string]*oas.Schema{
								"name": {
									Title: "Application name",
									Type:  "string",
								},
								"id": {
									Title: "Application id",
									Type:  "string",
								},
								"version": {
									Title: "Application version",
									Type:  "string",
								},
								"running": {
									Title: "Application running status",
									Type:  "boolean",
								},
								"launch": {
									Title:                "Application launch times",
									Type:                 "string",
									Format:               "2006-01-02T15:04:05Z07:00",
									AdditionalProperties: &oas.Schema{Type: "string"},
								},
								"now": {
									Title:                "Now",
									Type:                 "string",
									Format:               "2006-01-02T15:04:05Z07:00",
									AdditionalProperties: &oas.Schema{Type: "string"},
								},
							},
						}
						c = oas.ApplicationJsonContent(schema)
						return
					}(),
				},
				"503": {Ref: "#/components/responses/503"},
			},
		},
	}
	return
}

func ElementSchema(element Element) (v *oas.Schema) {
	// fns
	if element.isRef() {
		v = oas.RefSchema(element.Key())
		return
	}
	v = &oas.Schema{
		Key:         element.Key(),
		Title:       element.Title,
		Description: "",
		Type:        element.Type,
		Required:    nil,
		Format:      element.Format,
		Enum: func(enums []string) (v []interface{}) {
			if enums == nil || len(enums) == 0 {
				return
			}
			v = make([]interface{}, 0, len(enums))
			for _, enum := range enums {
				v = append(v, enum)
			}
			return
		}(element.Enums),
		Properties:           nil,
		Items:                nil,
		AdditionalProperties: nil,
		Deprecated:           element.Deprecated,
		Ref:                  "",
	}
	// Description
	description := "### Description" + "\n"
	description = description + element.Description + " "
	if element.Validation.Enable {
		description = description + "\n\n***Validation***" + " "
		description = description + "`" + element.Validation.Name + "`" + " "
		if element.Validation.I18ns != nil && len(element.Validation.I18ns) > 0 {
			description = description + "\n"
			i18nKeys := make([]string, 0, 1)
			for _, i18n := range element.Validation.I18ns {
				i18nKeys = append(i18nKeys, i18n.Name)
			}
			sort.Strings(i18nKeys)
			for _, i18nKey := range i18nKeys {
				i18nValue, hasI18nValue := element.Validation.I18ns.Get(i18nKey)
				if hasI18nValue {
					description = description + "  " + i18nKey + ": " + i18nValue + "\n"
				}
			}
		}
	}
	if element.Enums != nil && len(element.Enums) > 0 {
		description = description + "\n\n***Enum***" + " "
		description = description + fmt.Sprintf("[%s]", strings.Join(element.Enums, ", ")) + " "
	}
	v.Description = description
	// builtin
	if element.isBuiltin() {
		return
	}
	// object
	if element.isObject() && !element.isEmpty() {
		required := make([]string, 0, 1)
		v.Properties = make(map[string]*oas.Schema)
		for _, prop := range element.Properties {
			if prop.Element.Required {
				required = append(required, prop.Name)
			}
			v.Properties[prop.Name] = ElementSchema(prop.Element)
		}
		v.Required = required
		return
	}
	// array
	if element.isArray() {
		item, hasItem := element.getItem()
		if hasItem {
			v.Items = ElementSchema(item)
		}
		return
	}
	// map
	if element.isAdditional() {
		item, hasItem := element.getItem()
		if hasItem {
			v.AdditionalProperties = ElementSchema(item)
		}
		return
	}
	return
}
