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

package server

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/oas"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"sync"
)

const (
	httpDocumentRawPath = "/documents/raw"
	httpDocumentOASPath = "/documents/oas"
)

type documentHandler struct {
	log       logs.Logger
	doc       *Document
	endpoints service.Endpoints
	once      sync.Once
	raw       []byte
	oas       []byte
}

func (h *documentHandler) Name() (name string) {
	name = "documents"
	return
}

func (h *documentHandler) Build(options *HandlerOptions) (err error) {
	h.log = options.Log
	h.doc = &Document{}
	configErr := options.Config.As(h.doc)
	if configErr != nil {
		err = errors.Warning("fns: document handler build failed").WithCause(configErr)
		return
	}
	if h.doc.Title == "" {
		h.doc.Title = "FNS"
	}
	if h.doc.Description == "" {
		h.doc.Description = "Fn services"
	}
	h.doc.version = options.AppVersion

	h.endpoints = options.Endpoints
	h.once = sync.Once{}
	return
}

func (h *documentHandler) Accept(request *http.Request) (ok bool) {
	ok = request.Method == http.MethodGet && (request.URL.Path == httpDocumentRawPath || request.URL.Path == httpDocumentOASPath)
	return
}

func (h *documentHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case httpDocumentRawPath:
		h.encode()
		h.write(writer, h.raw)
		break
	case httpDocumentOASPath:
		h.encode()
		h.write(writer, h.oas)
		break
	default:
		h.write(writer, []byte{'{', '}'})
		return
	}
	return
}

func (h *documentHandler) Close() {
}

func (h *documentHandler) write(writer http.ResponseWriter, body []byte) {
	writer.Header().Set(httpServerHeader, httpServerHeaderValue)
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(http.StatusOK)
	if body == nil || len(body) == 0 {
		return
	}
	_, _ = writer.Write(body)
}

func (h *documentHandler) encode() {
	h.once.Do(func() {
		// raw
		sds := h.endpoints.Documents()
		if sds != nil || len(sds) > 0 {
			dr, drErr := json.Marshal(sds)
			if drErr != nil {
				if h.log.WarnEnabled() {
					h.log.Warn().Cause(drErr).Message("fns: encode service documents failed")
				}
				h.raw = []byte{'{', '}'}
			} else {
				h.raw = dr
			}
		}
		// oas
		api := &oas.API{
			Openapi: "3.0.3",
			Info: &oas.Info{
				Title:          h.doc.Title,
				Description:    h.doc.Description,
				TermsOfService: h.doc.Terms,
				Contact:        nil,
				License:        nil,
				Version:        h.doc.version,
			},
			Servers: []*oas.Server{},
			Paths:   make(map[string]*oas.Path),
			Components: &oas.Components{
				Schemas:   make(map[string]*oas.Schema),
				Responses: make(map[string]*oas.Response),
			},
			Tags: make([]*oas.Tag, 0, 1),
		}
		// info
		if h.doc.Contact != nil {
			api.Info.SetContact(h.doc.Contact.Name, h.doc.Contact.Url, h.doc.Contact.Email)
		}
		if h.doc.License != nil {
			api.Info.SetLicense(h.doc.License.Name, h.doc.License.Url)
		}
		// servers
		if h.doc.Addresses != nil && len(h.doc.Addresses) > 0 {
			for _, address := range h.doc.Addresses {
				api.Servers = append(api.Servers, &oas.Server{
					Url:         address.URL,
					Description: address.Description,
				})
			}
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
			Description: "Empty object",
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
				"Document/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["401"] = &oas.Response{
			Description: "***UNAUTHORIZED***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["403"] = &oas.Response{
			Description: "***FORBIDDEN***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["404"] = &oas.Response{
			Description: "***NOT FOUND***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["406"] = &oas.Response{
			Description: "***NOT ACCEPTABLE***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["408"] = &oas.Response{
			Description: "***TIMEOUT***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["500"] = &oas.Response{
			Description: "***SERVICE EXECUTE FAILED***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["501"] = &oas.Response{
			Description: "***SERVICE NOT IMPLEMENTED***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["503"] = &oas.Response{
			Description: "***SERVICE UNAVAILABLE***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		api.Components.Responses["555"] = &oas.Response{
			Description: "***WARNING***",
			Header:      responseHeader,
			Content: map[string]*oas.MediaType{
				"Document/json": {
					Schema: oas.RefSchema("fns_CodeError"),
				},
			},
		}
		// builtin
		api.Tags = append(api.Tags, &oas.Tag{
			Name:        "builtin",
			Description: "fns builtins",
		})
		checkHealthPath := &oas.Path{
			Post: &oas.Operation{
				OperationId: "check_health",
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
								Key:         "fns_check_health_result",
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
		api.Paths["/health"] = checkHealthPath
		// service
		if sds != nil || len(sds) > 0 {
			for _, document := range sds {
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
								if fn.Authorization() {
									return authorizationHeaderParams
								}
								return nil
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
											c = oas.ApplicationJsonContent(oas.RefSchema("fns_Empty"))
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
		or, orErr := json.Marshal(api)
		if orErr != nil {
			if h.log.WarnEnabled() {
				h.log.Warn().Cause(orErr).Message("fns: encode open api documents failed")
			}
			h.oas = []byte{'{', '}'}
		} else {
			h.oas = or
		}
	})
}

type Contact struct {
	Name  string `json:"name"`
	Url   string `json:"url"`
	Email string `json:"email"`
}

type License struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

type Address struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type Document struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Terms       string    `json:"terms"`
	Contact     *Contact  `json:"contact"`
	License     *License  `json:"license"`
	Addresses   []Address `json:"servers"`
	version     string
}

func defaultDocument() *Document {
	return &Document{
		Title:       "Fns",
		Description: "Fn services",
		Terms:       "",
		Contact:     nil,
		License:     nil,
		version:     "",
		Addresses:   nil,
	}
}
