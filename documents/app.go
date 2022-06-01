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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/oas"
	"github.com/aacfactory/json"
	"sync"
)

type Contact struct {
	Name  string `json:"name"`
	Url   string `json:"url"`
	Email string `json:"email"`
}

type License struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

func New() *Application {
	return &Application{
		Title:         "FNS",
		Description:   "",
		Terms:         "",
		Contact:       Contact{},
		License:       License{},
		Version:       "0.0.0",
		Services:      make(map[string]*Service),
		URL:           "",
		once:          sync.Once{},
		oasRAW:        nil,
		convertOasErr: nil,
		raw:           nil,
		encodeErr:     nil,
	}
}

type Application struct {
	Title         string              `json:"title,omitempty"`
	Description   string              `json:"description,omitempty"`
	Terms         string              `json:"terms,omitempty"`
	Contact       Contact             `json:"contact,omitempty"`
	License       License             `json:"license,omitempty"`
	Version       string              `json:"version,omitempty"`
	Services      map[string]*Service `json:"services"`
	URL           string              `json:"url"`
	once          sync.Once
	oasRAW        []byte
	convertOasErr errors.CodeError
	raw           []byte
	encodeErr     errors.CodeError
}

func (app *Application) AddService(name string, value *Service) {
	app.Services[name] = value
	return
}

func (app *Application) encode() {
	app.once.Do(func() {
		oasRAW, oasErr := app.convertToOpenAPI()
		if oasRAW != nil {
			app.convertOasErr = errors.Warning("fns: convert documents to oas failed").WithCause(oasErr)
		} else {
			app.oasRAW = oasRAW
		}
		raw, rawErr := json.Marshal(app)
		if rawErr != nil {
			app.encodeErr = errors.Warning("fns: encode documents to json failed").WithCause(rawErr)
		} else {
			app.raw = raw
		}
	})
	return
}

func (app *Application) Json() (p []byte, err errors.CodeError) {
	app.encode()
	if app.encodeErr != nil {
		err = app.encodeErr
		return
	}
	p = app.raw
	return
}

func (app *Application) OAS() (p []byte, err errors.CodeError) {
	app.encode()
	if app.convertOasErr != nil {
		err = app.convertOasErr
		return
	}
	p = app.oasRAW
	return
}

func (app *Application) convertToOpenAPI() (p []byte, err error) {
	api := &oas.API{
		Openapi: "3.0.3",
		Info: &oas.Info{
			Title:          app.Title,
			Description:    app.Description,
			TermsOfService: app.Terms,
			Contact:        nil,
			License:        nil,
			Version:        app.Version,
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
	api.Info.SetContact(app.Contact.Name, app.Contact.Url, app.Contact.Email)
	api.Info.SetLicense(app.License.Name, app.License.Url)
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
	for _, service := range app.Services {
		// tags
		api.Tags = append(api.Tags, &oas.Tag{
			Name:        service.Name,
			Description: service.Description,
		})
		// fn
		for _, fn := range service.Fns {
			// path
			path := &oas.Path{
				Post: &oas.Operation{
					OperationId: fmt.Sprintf("%s_%s", service.Name, fn.Name),
					Tags:        []string{service.Name},
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
			api.Paths[fmt.Sprintf("/%s/%s", service.Name, fn.Name)] = path
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
	p, err = json.Marshal(api)
	return
}
