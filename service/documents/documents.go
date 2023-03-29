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
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/internal/oas"
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

type VersionSortedDocuments []*Document

func (documents VersionSortedDocuments) Get(version versions.Version) (document *Document, has bool) {
	if documents == nil || len(documents) == 0 {
		return
	}
	for _, document = range documents {
		if document.Version.Equals(version) {
			has = true
			return
		}
	}
	document = nil
	return
}

func (documents VersionSortedDocuments) Merge(o VersionSortedDocuments) VersionSortedDocuments {
	if o == nil || len(o) == 0 {
		return documents
	}
	deltas := make([]int, 0, 1)
	for i, document := range o {
		pos := sort.Search(len(documents), func(i int) bool {
			return documents[i].Version.Equals(document.Version)
		})
		if pos == len(documents) {
			deltas = append(deltas, i)
		}
	}
	if len(deltas) == 0 {
		return documents
	}
	merged := VersionSortedDocuments(make([]*Document, 0, 1))
	for _, document := range documents {
		merged = append(merged, document)
	}
	for _, delta := range deltas {
		merged = append(merged, o[delta])
	}
	sort.Sort(merged)
	return merged
}

func (documents VersionSortedDocuments) Len() int {
	return len(documents)
}

func (documents VersionSortedDocuments) Less(i, j int) bool {
	return documents[i].Version.LessThan(documents[j].Version)
}

func (documents VersionSortedDocuments) Swap(i, j int) {
	documents[i], documents[j] = documents[j], documents[i]
	return
}

func NewDocuments() Documents {
	return make(map[string]VersionSortedDocuments)
}

type Documents map[string]VersionSortedDocuments

func (documents Documents) Add(doc *Document) (ok bool) {
	if doc == nil {
		return
	}
	name := doc.Name
	sorts, has := documents[name]
	if has {
		size := documents.Len()
		pos := sort.Search(size, func(i int) bool {
			return sorts[i].Version.Equals(doc.Version)
		})
		if pos < size {
			return
		}
		sorts = append(sorts, doc)
		sort.Sort(sorts)
		documents[name] = sorts
		ok = true
		return
	}
	sorts = make([]*Document, 0, 1)
	sorts = append(sorts, doc)
	documents[name] = sorts
	ok = true
	return
}

func (documents Documents) Len() int {
	return len(documents)
}

func (documents Documents) FindByVersion(version versions.Version) (v NameSortedDocuments) {
	if documents == nil || len(documents) == 0 {
		return
	}
	v = make([]*Document, 0, 1)
	for _, sorts := range documents {
		if sorts == nil || len(sorts) == 0 {
			continue
		}
		var document *Document
		var has bool
		if version.IsLatest() {
			document = sorts[len(sorts)-1]
			has = true
		} else {
			document, has = sorts.Get(version)
		}
		if has {
			v = append(v, document)
		}
	}
	if len(v) > 0 {
		sort.Sort(v)
	}
	return
}

func (documents Documents) Merge(o Documents) Documents {
	if o == nil || len(o) == 0 {
		return documents
	}
	for name, versionedDocuments := range o {
		if versionedDocuments == nil || len(versionedDocuments) == 0 {
			continue
		}
		doc, has := documents[name]
		if !has {
			documents[name] = doc
			continue
		}
		merged := doc.Merge(versionedDocuments)
		documents[name] = merged
	}
	return documents
}

func (documents Documents) Openapi(openapiVersion string, appId string, appName string, appVersion versions.Version) (api *oas.API) {
	if openapiVersion == "" {
		openapiVersion = "3.1.0"
	}
	// oas
	api = &oas.API{
		Openapi: openapiVersion,
		Info: &oas.Info{
			Title:          appName,
			Description:    fmt.Sprintf("%s(%s)", appName, appId),
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

	// documents
	if documents != nil || len(documents) > 0 {
		for _, sorts := range documents {
			if sorts == nil || len(sorts) == 0 {
				continue
			}
			var document *Document
			var matched bool
			if appVersion.IsLatest() {
				document = sorts[len(sorts)-1]
				matched = true
			} else {
				document, matched = sorts.Get(appVersion)
			}
			if !matched {
				continue
			}
			// tags
			api.Tags = append(api.Tags, &oas.Tag{
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
				path := &oas.Path{
					Post: &oas.Operation{
						OperationId: fmt.Sprintf("%s_%s", document.Name, fn.Name),
						Tags:        []string{document.Name},
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
							Required:    func() bool { return fn.Argument != nil }(),
							Description: "",
							Content: func() (c map[string]*oas.MediaType) {
								if fn.Argument == nil {
									return
								}
								c = oas.ApplicationJsonContent(fn.Argument.Schema())
								return
							}(),
						},
						Responses: map[string]oas.Response{
							"200": {
								Content: func() (c map[string]*oas.MediaType) {
									if fn.Result == nil {
										c = oas.ApplicationJsonContent(oas.RefSchema("github.com/aacfactory/fns/service.Empty"))
										return
									}
									c = oas.ApplicationJsonContent(fn.Result.Schema())
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
