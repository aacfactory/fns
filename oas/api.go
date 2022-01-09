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

package oas

type API struct {
	Openapi    string           `json:"openapi,omitempty"`
	Info       *Info            `json:"info,omitempty"`
	Servers    []*Server        `json:"servers,omitempty"`
	Paths      map[string]*Path `json:"paths,omitempty"`
	Components *Components      `json:"components,omitempty"`
	Tags       []*Tag           `json:"tags,omitempty"`
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
