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

package oas

import (
	"github.com/aacfactory/json"
	"sort"
)

type API struct {
	Openapi    string           `json:"openapi,omitempty"`
	Info       *Info            `json:"info,omitempty"`
	Servers    []*Server        `json:"servers,omitempty"`
	Paths      map[string]*Path `json:"paths,omitempty"`
	Components *Components      `json:"components,omitempty"`
	Tags       []*Tag           `json:"tags,omitempty"`
}

func (api *API) Merge(o *API) {
	if o.Paths == nil || len(o.Paths) == 0 {
		return
	}
	if api.Paths == nil {
		api.Paths = make(map[string]*Path)
	}
	if api.Components == nil {
		api.Components = &Components{}
	}
	if api.Components.Schemas == nil {
		api.Components.Schemas = make(map[string]*Schema)
	}
	if api.Components.Responses == nil {
		api.Components.Responses = make(map[string]*Response)
	}
	if api.Tags == nil {
		api.Tags = make([]*Tag, 0, 1)
	}
	for name, path := range o.Paths {
		_, has := api.Paths[name]
		if has {
			continue
		}
		api.Paths[name] = path
	}
	if o.Components != nil {
		if o.Components.Schemas != nil {
			for name, schema := range o.Components.Schemas {
				_, has := api.Components.Schemas[name]
				if has {
					continue
				}
				api.Components.Schemas[name] = schema
			}
		}
		if o.Components.Responses != nil {
			for name, responses := range o.Components.Responses {
				_, has := api.Components.Responses[name]
				if has {
					continue
				}
				api.Components.Responses[name] = responses
			}
		}
	}
	if o.Tags != nil && len(o.Tags) > 0 {
		deltas := make([]int, 0, 1)
		for i, tag := range o.Tags {
			pos := sort.Search(len(api.Tags), func(i int) bool {
				return api.Tags[i].Name == tag.Name
			})
			if pos == len(api.Tags) {
				deltas = append(deltas, i)
			}
		}
		if len(deltas) > 0 {
			for _, delta := range deltas {
				api.Tags = append(api.Tags, o.Tags[delta])
			}
		}
	}
}

func (api *API) Encode() (p []byte, err error) {
	obj := json.NewObject()
	_ = obj.Put("openapi", api.Openapi)
	if api.Info != nil {
		_ = obj.Put("info", api.Info)
	}
	if api.Servers != nil && len(api.Servers) > 0 {
		array := json.NewArray()
		for _, server := range api.Servers {
			if server != nil {
				_ = array.Add(server)
			}
		}
		if array.Len() > 0 {
			_ = obj.PutRaw("servers", array.Raw())
		}
	}
	if api.Paths != nil && len(api.Paths) > 0 {
		keys := make([]string, 0, 1)
		for key := range api.Paths {
			keys = append(keys, key)
		}
		if len(keys) > 0 {
			sort.Strings(keys)
			paths := json.NewObject()
			for _, key := range keys {
				path, has := api.Paths[key]
				if !has {
					continue
				}
				_ = paths.Put(key, path)
			}
			if !paths.Empty() {
				_ = obj.PutRaw("paths", paths.Raw())
			}
		}
	}
	if api.Components != nil {
		components := json.NewObject()
		schemas := api.Components.Schemas
		if schemas != nil && len(schemas) > 0 {
			keys := make([]string, 0, 1)
			for key := range schemas {
				keys = append(keys, key)
			}
			if len(keys) > 0 {
				sort.Strings(keys)
				schemasObj := json.NewObject()
				for _, key := range keys {
					schema, has := schemas[key]
					if !has {
						continue
					}
					_ = schemasObj.Put(key, schema)
				}
				if !schemasObj.Empty() {
					_ = components.PutRaw("schemas", schemasObj.Raw())
				}
			}
		}
		responses := api.Components.Responses
		if responses != nil && len(responses) > 0 {
			keys := make([]string, 0, 1)
			for key := range responses {
				keys = append(keys, key)
			}
			if len(keys) > 0 {
				sort.Strings(keys)
				responsesObj := json.NewObject()
				for _, key := range keys {
					response, has := responses[key]
					if !has {
						continue
					}
					_ = responsesObj.Put(key, response)
				}
				if !responsesObj.Empty() {
					_ = components.PutRaw("responses", responsesObj.Raw())
				}
			}
		}
		if !components.Empty() {
			_ = obj.PutRaw("components", components.Raw())
		}
	}
	if api.Tags != nil && len(api.Tags) > 0 {
		keys := make([]string, 0, 1)
		for _, tag := range api.Tags {
			if tag != nil {
				keys = append(keys, tag.Name)
			}
		}
		if len(keys) > 0 {
			sort.Strings(keys)
			tags := json.NewArray()
			for _, key := range keys {
				if key == "builtin" {
					continue
				}
				for _, tag := range api.Tags {
					if tag.Name == key {
						_ = tags.Add(tag)
						break
					}
				}
			}
			_ = tags.Add(&Tag{
				Name:        "builtin",
				Description: "fns builtins",
			})
			_ = obj.PutRaw("tags", tags.Raw())
		}
	}
	p, err = obj.MarshalJSON()
	return
}

type Info struct {
	Title          string      `json:"title,omitempty"`
	Description    string      `json:"description,omitempty"`
	TermsOfService string      `json:"termsOfService,omitempty"`
	Contact        interface{} `json:"contact,omitempty"`
	License        interface{} `json:"license,omitempty"`
	Version        string      `json:"version,omitempty"`
}

func (v *Info) SetContact(name string, url string, email string) {
	v.Contact = map[string]string{
		"name":  name,
		"url":   url,
		"email": email,
	}
}

func (v *Info) SetLicense(name string, url string) {
	v.License = map[string]string{
		"name": name,
		"url":  url,
	}
}

type Server struct {
	Url         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}

type Components struct {
	Schemas   map[string]*Schema   `json:"schemas,omitempty"`
	Responses map[string]*Response `json:"responses,omitempty"`
}

type Tag struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type Response struct {
	Description string                `json:"description,omitempty"`
	Header      map[string]*Header    `json:"header,omitempty"`
	Content     map[string]*MediaType `json:"content,omitempty"`
	Ref         string                `json:"$ref,omitempty"`
}

type Header struct {
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

func ApplicationJsonContent(v *Schema) (c map[string]*MediaType) {
	c = map[string]*MediaType{
		"application/json": {Schema: v},
	}
	return
}

type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}
