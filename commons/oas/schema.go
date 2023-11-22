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

import "fmt"

func ComponentsSchemaURI(key string) (v string) {
	v = fmt.Sprintf("#/components/schemas/%s", key)
	return
}

func RefSchema(key string) (v *Schema) {
	v = &Schema{}
	v.Ref = ComponentsSchemaURI(key)
	return
}

type Schema struct {
	Key                  string             `json:"-"`
	Title                string             `json:"title,omitempty"`
	Description          string             `json:"description,omitempty"`
	Type                 string             `json:"type,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Format               string             `json:"format,omitempty"`
	Enum                 []interface{}      `json:"enum,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Deprecated           bool               `json:"deprecated,omitempty"`
	Example              interface{}        `json:"example,omitempty"`
	Ref                  string             `json:"$ref,omitempty"`
}
