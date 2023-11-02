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

package documents

import (
	"github.com/aacfactory/fns/commons/oas"
)

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
			Description: "Applicable version range, e.g.: endpointName1=0.0.1:1.0.0, endpointName2=0.0.1:1.0.0, ...",
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
