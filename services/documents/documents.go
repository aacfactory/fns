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
	"bytes"
	"fmt"
	"github.com/aacfactory/fns/commons/bytex"
	oas2 "github.com/aacfactory/fns/commons/oas"
	"github.com/aacfactory/fns/commons/versions"
	"sort"
)

type NameSortedDocuments []*Document

func (documents NameSortedDocuments) Len() int {
	return len(documents)
}

func (documents NameSortedDocuments) Less(i, j int) bool {
	return documents[i].Name < documents[j].Name
}

func (documents NameSortedDocuments) Swap(i, j int) {
	documents[i], documents[j] = documents[j], documents[i]
	return
}

func NewDocuments(id []byte, name []byte, version versions.Version) *Documents {
	return &Documents{
		Id:        string(id),
		Name:      string(name),
		Version:   version,
		Endpoints: make(NameSortedDocuments, 0, 1),
	}
}

type Documents struct {
	Id        string              `json:"id"`
	Name      string              `json:"name"`
	Version   versions.Version    `json:"version"`
	Endpoints NameSortedDocuments `json:"endpoints"`
}

func (documents *Documents) Add(doc *Document) {
	if doc == nil {
		return
	}
	documents.Endpoints = append(documents.Endpoints, doc)
	sort.Sort(documents.Endpoints)
}

func (documents *Documents) Get(name []byte) (v *Document) {
	for _, endpoint := range documents.Endpoints {
		if bytes.Equal(name, bytex.FromString(endpoint.Name)) {
			return endpoint
		}
	}
	return nil
}

func (documents *Documents) Openapi(openapiVersion string) (api oas2.API) {
	if openapiVersion == "" {
		openapiVersion = "3.1.0"
	}
	// oas
	api = oas2.API{
		Openapi: openapiVersion,
		Info: &oas2.Info{
			Title:          documents.Name,
			Description:    fmt.Sprintf("%s(%s)", documents.Name, documents.Id),
			TermsOfService: "",
			Contact:        nil,
			License:        nil,
			Version:        documents.Version.String(),
		},
		Servers: []*oas2.Server{},
		Paths:   make(map[string]*oas2.Path),
		Components: &oas2.Components{
			Schemas:   make(map[string]*oas2.Schema),
			Responses: make(map[string]*oas2.Response),
		},
		Tags: make([]*oas2.Tag, 0, 1),
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
	api.Tags = append(api.Tags, &oas2.Tag{
		Name:        "builtin",
		Description: "fns builtins",
	})
	healthURI, healthPathSchema := healthPath()
	api.Paths[healthURI] = healthPathSchema

	// documents
	endpoints := documents.Endpoints
	if endpoints != nil || len(endpoints) > 0 {
		for _, document := range endpoints {
			if document == nil || document.Internal || len(document.Fns) == 0 {
				continue
			}
			// tags
			api.Tags = append(api.Tags, &oas2.Tag{
				Name:        document.Name,
				Description: document.Description,
			})
			// doc
			if document.Elements != nil {
				for _, element := range document.Elements {
					if _, hasElement := api.Components.Schemas[element.Key()]; !hasElement {
						api.Components.Schemas[element.Key()] = element.Schema()
					}
				}
			}
			for _, fn := range document.Fns {
				description := fn.Description
				if fn.Errors != nil && len(fn.Errors) > 0 {
					description = description + "\n----------\n"
					description = description + "Errors:\n"
					for _, errorDocument := range fn.Errors {
						description = description + "* " + errorDocument.Name() + "\n"
						i18nKeys := make([]string, 0, 1)
						for i18nKey := range errorDocument.Descriptions() {
							i18nKeys = append(i18nKeys, i18nKey)
						}
						sort.Strings(i18nKeys)
						for _, i18nKey := range i18nKeys {
							i18nVal := errorDocument.Descriptions()[i18nKey]
							description = description + "\t* " + i18nKey + ": " + i18nVal + "\n"
						}
					}
				}
				path := &oas2.Path{
					Post: &oas2.Operation{
						OperationId: fmt.Sprintf("%s_%s", document.Name, fn.Name),
						Tags:        []string{document.Name},
						Summary:     fn.Title,
						Description: description,
						Deprecated:  fn.Deprecated,
						Parameters: func() []*oas2.Parameter {
							params := requestHeadersOpenapiParams()
							if fn.Authorization {
								params = append(params, requestAuthHeadersOpenapiParams()...)
								return params
							}
							return params
						}(),
						RequestBody: &oas2.RequestBody{
							Required:    func() bool { return fn.Argument != nil }(),
							Description: "",
							Content: func() (c map[string]*oas2.MediaType) {
								if fn.Argument == nil {
									return
								}
								c = oas2.ApplicationJsonContent(fn.Argument.Schema())
								return
							}(),
						},
						Responses: map[string]oas2.Response{
							"200": {
								Content: func() (c map[string]*oas2.MediaType) {
									if fn.Result == nil {
										c = oas2.ApplicationJsonContent(oas2.RefSchema("github.com/aacfactory/fns/service.Empty"))
										return
									}
									c = oas2.ApplicationJsonContent(fn.Result.Schema())
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
				api.Paths[fmt.Sprintf("/%s/%s", document.Name, fn.Name)] = path
			}
		}
	}
	return
}
