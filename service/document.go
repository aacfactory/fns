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

package service

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/internal/oas"
	"github.com/aacfactory/json"
)

type Document interface {
	Name() (name string)
	Description() (description string)
	Fns() []FnDocument
	Elements() (elements map[string]ElementDocument)
}

type FnDocument interface {
	Name() (name string)
	Title() (title string)
	Description() (description string)
	Authorization() (has bool)
	Deprecated() (deprecated bool)
	Argument() (argument ElementDocument)
	Result() (result ElementDocument)
}

type ElementDocument interface {
	Key() (v string)
	Schema() (schema *oas.Schema)
}

func encodeDocuments(appId string, appName string, appVersion versions.Version, eps map[string]*endpoint) (p []byte, err error) {
	documents := make(map[string]Document)
	for name, ep := range eps {
		document := ep.Document()
		if document == nil {
			continue
		}
		documents[name] = document
	}
	obj := json.NewObject()
	_ = obj.Put("id", appId)
	_ = obj.Put("name", appName)
	_ = obj.Put("version", appVersion)
	err = obj.Put("services", documents)
	if err != nil {
		err = errors.Warning("fns: encode services document failed").WithCause(err)
		return
	}
	p = obj.Raw()
	return
}

func encodeOpenapi(appId string, appName string, appVersion versions.Version, eps map[string]*endpoint) (p []byte, err error) {
	// oas
	api := &oas.API{
		Openapi: "3.0.3",
		Info: &oas.Info{
			Title:          appName,
			Description:    fmt.Sprintf("%s(%s) @%s", appName, appId, appVersion.String()),
			TermsOfService: "",
			Contact:        nil,
			License:        nil,
			Version:        appVersion.String(),
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

	if eps != nil || len(eps) > 0 {
		for _, ep := range eps {
			if ep.Internal() {
				continue
			}
			document := ep.Document()
			if document == nil {
				continue
			}
			// tags
			api.Tags = append(api.Tags, &oas.Tag{
				Name:        document.Name(),
				Description: document.Description(),
			})
			// doc
			if document.Elements() != nil {
				for _, element := range document.Elements() {
					if _, hasElement := api.Components.Schemas[element.Key()]; !hasElement {
						api.Components.Schemas[element.Key()] = element.Schema()
					}
				}
			}
			for _, fn := range document.Fns() {
				path := &oas.Path{
					Post: &oas.Operation{
						OperationId: fmt.Sprintf("%s_%s", document.Name(), fn.Name()),
						Tags:        []string{document.Name()},
						Summary:     fn.Title(),
						Description: fn.Description(),
						Deprecated:  fn.Deprecated(),
						Parameters: func() []*oas.Parameter {
							params := requestHeadersOpenapiParams()
							if fn.Authorization() {
								params = append(params, requestAuthHeadersOpenapiParams()...)
								return params
							}
							return params
						}(),
						RequestBody: &oas.RequestBody{
							Required:    func() bool { return fn.Argument() != nil }(),
							Description: "",
							Content: func() (c map[string]*oas.MediaType) {
								if fn.Argument() == nil {
									return
								}
								c = oas.ApplicationJsonContent(fn.Argument().Schema())
								return
							}(),
						},
						Responses: map[string]oas.Response{
							"200": {
								Content: func() (c map[string]*oas.MediaType) {
									if fn.Result() == nil {
										c = oas.ApplicationJsonContent(oas.RefSchema("github.com.aacfactory.fns.service@Empty"))
										return
									}
									c = oas.ApplicationJsonContent(fn.Result().Schema())
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
				api.Paths[fmt.Sprintf("/%s/%s", document.Name(), fn.Name())] = path
			}
		}
	}
	p, err = json.Marshal(api)
	if err != nil {
		err = errors.Warning("fns: encode services openapi failed").WithCause(err)
		return
	}
	return
}

func codeErrOpenapiSchema() *oas.Schema {
	return &oas.Schema{
		Key:         "github.com.aacfactory.errors@CodeError",
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
			"cause": oas.RefSchema("github.com.aacfactory.errors@CodeError"),
		},
	}
}

func jsonRawMessageOpenapiSchema() *oas.Schema {
	return &oas.Schema{
		Key:         "github.com.aacfactory.json@RawMessage",
		Title:       "JsonRawMessage",
		Description: "Json Raw Message",
		Type:        "object",
	}
}

func emptyOpenapiSchema() *oas.Schema {
	return &oas.Schema{
		Key:         "github.com.aacfactory.fns.service@Empty",
		Title:       "Empty",
		Description: "Empty object",
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
			Description: "Applicable version range, e.g.: 0.0.1,1.0.1",
			Required:    false,
		},
		{
			Name:        "X-Fns-Request-Signature",
			In:          "header",
			Description: "request signature",
			Required:    false,
		},
	}
}

func responseHeadersOpenapi() map[string]*oas.Header {
	return map[string]*oas.Header{
		"X-Fns-Id": {
			Description: "app id",
			Schema: &oas.Schema{
				Type: "string",
			},
		},
		"X-Fns-Name": {
			Description: "app name",
			Schema: &oas.Schema{
				Type: "string",
			},
		},
		"X-Fns-Version": {
			Description: "app version",
			Schema: &oas.Schema{
				Type: "string",
			},
		},
		"X-Fns-Request-Id": {
			Description: "request id",
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
					Schema: oas.RefSchema("github.com.aacfactory.errors@CodeError"),
				},
			},
		},
		"401": {
			Description: "***UNAUTHORIZED***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com.aacfactory.errors@CodeError"),
				},
			},
		},
		"403": {
			Description: "***FORBIDDEN***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com.aacfactory.errors@CodeError"),
				},
			},
		},
		"404": {
			Description: "***NOT FOUND***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com.aacfactory.errors@CodeError"),
				},
			},
		},
		"406": {
			Description: "***NOT ACCEPTABLE***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com.aacfactory.errors@CodeError"),
				},
			},
		},
		"408": {
			Description: "***TIMEOUT***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com.aacfactory.errors@CodeError"),
				},
			},
		},
		"500": {
			Description: "***SERVICE EXECUTE FAILED***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com.aacfactory.errors@CodeError"),
				},
			},
		},
		"501": {
			Description: "***SERVICE NOT IMPLEMENTED***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com.aacfactory.errors@CodeError"),
				},
			},
		},
		"503": {
			Description: "***SERVICE UNAVAILABLE***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com.aacfactory.errors@CodeError"),
				},
			},
		},
		"555": {
			Description: "***WARNING***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("github.com.aacfactory.errors@CodeError"),
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
			Summary:     "Check Health",
			Description: "Check fns system health status",
			Deprecated:  false,
			Parameters:  nil,
			RequestBody: nil,
			Responses: map[string]oas.Response{
				"200": {
					Content: func() (c map[string]*oas.MediaType) {
						schema := &oas.Schema{
							Key:         "github.com.aacfactory.fns.service@ApplicationHealth",
							Title:       "Check Health Result",
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
