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
	oas2 "github.com/aacfactory/fns/commons/oas"
)

func codeErrOpenapiSchema() *oas2.Schema {
	return &oas2.Schema{
		Key:         "github.com/aacfactory/errors.CodeError",
		Title:       "CodeError",
		Description: "Fns Code Error",
		Type:        "object",
		Required:    []string{"id", "code", "name", "message", "stacktrace"},
		Properties: map[string]*oas2.Schema{
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
				AdditionalProperties: &oas2.Schema{Type: "string"},
			},
			"stacktrace": {
				Title: "Stacktrace",
				Type:  "object",
				Properties: map[string]*oas2.Schema{
					"fn":   {Type: "string"},
					"file": {Type: "string"},
					"line": {Type: "string"},
				},
			},
			"cause": oas2.RefSchema("github.com/aacfactory/errors.CodeError"),
		},
	}
}

func jsonRawMessageOpenapiSchema() *oas2.Schema {
	return &oas2.Schema{
		Key:         "github.com/aacfactory/json.RawMessage",
		Title:       "JsonRawMessage",
		Description: "Json Raw Message",
		Type:        "object",
	}
}

func emptyOpenapiSchema() *oas2.Schema {
	return &oas2.Schema{
		Key:         "github.com/aacfactory/fns/service.Empty",
		Title:       "Empty",
		Description: "Empty Object",
		Type:        "object",
	}
}

func requestAuthHeadersOpenapiParams() []*oas2.Parameter {
	return []*oas2.Parameter{
		{
			Name:        "Authorization",
			In:          "header",
			Description: "Authorization Key",
			Required:    true,
		},
	}
}

func requestHeadersOpenapiParams() []*oas2.Parameter {
	return []*oas2.Parameter{
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

func responseHeadersOpenapi() map[string]*oas2.Header {
	return map[string]*oas2.Header{
		"X-Fns-Id": {
			Description: "app id",
			Schema: &oas2.Schema{
				Type: "string",
			},
		},
		"X-Fns-Name": {
			Description: "app name",
			Schema: &oas2.Schema{
				Type: "string",
			},
		},
		"X-Fns-Version": {
			Description: "app version",
			Schema: &oas2.Schema{
				Type: "string",
			},
		},
		"X-Fns-Request-Id": {
			Description: "request id",
			Schema: &oas2.Schema{
				Type: "string",
			},
		},
		"X-Fns-Handle-Latency": {
			Description: "request latency",
			Schema: &oas2.Schema{
				Type: "string",
			},
		},
	}
}

func responseStatusOpenapi() map[string]*oas2.Response {
	return map[string]*oas2.Response{
		"400": {
			Description: "***BAD REQUEST***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas2.MediaType{
				"Document/json": {
					Schema: oas2.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"401": {
			Description: "***UNAUTHORIZED***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas2.MediaType{
				"Document/json": {
					Schema: oas2.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"403": {
			Description: "***FORBIDDEN***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas2.MediaType{
				"Document/json": {
					Schema: oas2.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"404": {
			Description: "***NOT FOUND***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas2.MediaType{
				"Document/json": {
					Schema: oas2.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"406": {
			Description: "***NOT ACCEPTABLE***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas2.MediaType{
				"Document/json": {
					Schema: oas2.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"408": {
			Description: "***TIMEOUT***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas2.MediaType{
				"Document/json": {
					Schema: oas2.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"500": {
			Description: "***SERVICE EXECUTE FAILED***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas2.MediaType{
				"Document/json": {
					Schema: oas2.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"501": {
			Description: "***SERVICE NOT IMPLEMENTED***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas2.MediaType{
				"Document/json": {
					Schema: oas2.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"503": {
			Description: "***SERVICE UNAVAILABLE***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas2.MediaType{
				"Document/json": {
					Schema: oas2.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
		"555": {
			Description: "***WARNING***",
			Header:      responseHeadersOpenapi(),
			Content: map[string]*oas2.MediaType{
				"Document/json": {
					Schema: oas2.RefSchema("github.com/aacfactory/errors.CodeError"),
				},
			},
		},
	}
}

func healthPath() (uri string, path *oas2.Path) {
	uri = "/application/health"
	path = &oas2.Path{
		Get: &oas2.Operation{
			OperationId: "application_health",
			Tags:        []string{"builtin"},
			Summary:     "Health Check",
			Description: "Check fns system health status",
			Deprecated:  false,
			Parameters:  nil,
			RequestBody: nil,
			Responses: map[string]oas2.Response{
				"200": {
					Content: func() (c map[string]*oas2.MediaType) {
						schema := &oas2.Schema{
							Key:         "github.com/aacfactory/fns/service.ApplicationHealth",
							Title:       "Health Check Result",
							Description: "",
							Type:        "object",
							Required:    []string{"name", "id", "version", "running", "now"},
							Properties: map[string]*oas2.Schema{
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
									AdditionalProperties: &oas2.Schema{Type: "string"},
								},
								"now": {
									Title:                "Now",
									Type:                 "string",
									Format:               "2006-01-02T15:04:05Z07:00",
									AdditionalProperties: &oas2.Schema{Type: "string"},
								},
							},
						}
						c = oas2.ApplicationJsonContent(schema)
						return
					}(),
				},
				"503": {Ref: "#/components/responses/503"},
			},
		},
	}
	return
}
