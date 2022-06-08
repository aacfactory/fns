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
	"github.com/aacfactory/fns/internal/oas"
	"strings"
)

func NewElement(pkg string, name string, typ string, format string, title string, description string) *Element {
	return &Element{
		Package:     pkg,
		Name:        name,
		Title:       title,
		Description: description,
		Type:        typ,
		Format:      format,
		Enum:        make([]interface{}, 0, 1),
		Required:    false,
		Validation:  "",
		Properties:  make(map[string]*Element),
		Additional:  false,
		Deprecated:  false,
	}
}

func String() *Element {
	return NewElement("builtin", "string", "string", "", "String", "String")
}

func Bool() *Element {
	return NewElement("builtin", "bool", "boolean", "", "Bool", "Bool")
}

func Int() *Element {
	return Int64()
}

func Int32() *Element {
	return NewElement("builtin", "int32", "integer", "int32", "Int32", "Int32")
}

func Int64() *Element {
	return NewElement("builtin", "int64", "integer", "int64", "Int64", "Int64")
}

func Float32() *Element {
	return NewElement("builtin", "float32", "number", "float", "Float", "Float")
}

func Float64() *Element {
	return NewElement("builtin", "float64", "number", "double", "Double", "Double")
}

func Date() *Element {
	return NewElement("builtin", "date", "string", "date", "Date", "Date")
}

func DateTime() *Element {
	return NewElement("builtin", "datetime", "string", "2006-01-02T15:04:05Z07:00", "Datetime", "Datetime").SetExample("2022-01-10T19:13:07+08:00")
}

func Struct(pkg string, name string, title string, description string) *Element {
	return NewElement(pkg, name, "object", "", title, description)
}

func JsonRaw() *Element {
	v := NewElement("fns", "JsonRawMessage", "object", "", "Json Raw", "Json Raw Message")
	v.Additional = true
	v.AddProperty("", Empty())
	return v
}

func Empty() *Element {
	return NewElement("fns", "Empty", "object", "", "Empty", "Empty Value")
}

func Array(name string, title string, description string, item *Element) *Element {
	v := NewElement(item.Package, name, "array", "", title, description)
	v.AddProperty("", item)
	return v
}

func Map(name string, title string, description string, item *Element) *Element {
	v := NewElement(item.Package, name, "object", "", title, description)
	v.Additional = true
	v.AddProperty("", item)
	return v
}

type Element struct {
	Package     string              `json:"package,omitempty"`
	Name        string              `json:"name,omitempty"`
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Type        string              `json:"type,omitempty"`
	Format      string              `json:"format,omitempty"`
	Enum        []interface{}       `json:"enum,omitempty"`
	Required    bool                `json:"required,omitempty"`
	Validation  string              `json:"validation,omitempty"`
	Properties  map[string]*Element `json:"properties,omitempty"`
	Additional  bool                `json:"additional,omitempty"`
	Deprecated  bool                `json:"deprecated,omitempty"`
	Example     interface{}         `json:"example,omitempty"`
}

func (element *Element) AsRequired(validation string) *Element {
	element.Required = true
	element.Validation = validation
	return element
}

func (element *Element) AsDeprecated() *Element {
	element.Deprecated = true
	return element
}

func (element *Element) SetValidation(validation string) *Element {
	element.Validation = validation
	return element
}

func (element *Element) SetTitle(title string) *Element {
	element.Title = title
	return element
}

func (element *Element) SetDescription(description string) *Element {
	element.Description = description
	return element
}

func (element *Element) SetExample(example interface{}) *Element {
	element.Example = example
	return element
}

func (element *Element) SetFormat(format string) *Element {
	element.Format = format
	return element
}

func (element *Element) AddEnum(v ...interface{}) *Element {
	element.Enum = append(element.Enum, v...)
	return element
}

func (element *Element) isEmpty() (ok bool) {
	ok = element.isObject() && len(element.Properties) == 0
	return
}

func (element *Element) isBuiltin() (ok bool) {
	ok = element.Type == "builtin"
	return
}

func (element *Element) isFns() (ok bool) {
	ok = element.Type == "fns"
	return
}

func (element *Element) isObject() (ok bool) {
	ok = element.Type == "object"
	return
}

func (element *Element) isArray() (ok bool) {
	ok = element.Type == "array"
	return
}

func (element *Element) isAdditional() (ok bool) {
	ok = element.isObject() && element.Additional
	return
}

func (element *Element) AddProperty(name string, prop *Element) *Element {
	if element.isObject() || element.isArray() || element.isAdditional() {
		element.Properties[name] = prop
	}
	return element
}

func (element *Element) objects() (v map[string]*Element) {
	v = make(map[string]*Element)
	if !element.isBuiltin() && !element.isFns() {
		key := element.Key()
		if _, has := v[key]; !has {
			v[key] = element
			for _, property := range element.Properties {
				deps := property.objects()
				if deps != nil && len(deps) > 0 {
					for depKey, dep := range deps {
						if _, hasDep := v[depKey]; !hasDep {
							v[depKey] = dep
						}
					}
				}
			}
		}
	}
	return
}

func (element *Element) Key() (v string) {
	v = fmt.Sprintf("%s_%s", strings.ReplaceAll(element.Package, "/", "."), element.Name)
	return
}

func (element *Element) Schema() (v *oas.Schema) {
	// fns
	if element.isFns() {
		v = oas.RefSchema(element.Key())
		return
	}
	v = &oas.Schema{
		Key:                  element.Key(),
		Title:                element.Title,
		Description:          "",
		Type:                 element.Type,
		Required:             nil,
		Format:               element.Format,
		Enum:                 element.Enum,
		Properties:           nil,
		Items:                nil,
		AdditionalProperties: nil,
		Deprecated:           element.Deprecated,
		Example:              element.Example,
		Ref:                  "",
	}
	// Description
	description := "### Description" + " "
	description = description + element.Description + " "
	if element.Validation != "" {
		description = description + "***Validation***" + " "
		description = description + "`" + element.Validation + "`" + " "
	}
	if element.Enum != nil && len(element.Enum) > 0 {
		description = description + "***Enum***" + " "
		description = description + fmt.Sprintf("%v", element.Enum) + " "
	}
	v.Description = description
	// builtin
	if element.isBuiltin() {
		return
	}
	// object
	if element.isObject() && !element.isEmpty() {
		required := make([]string, 0, 1)
		v.Properties = make(map[string]*oas.Schema)
		for name, prop := range element.Properties {
			if prop.Required {
				required = append(required, name)
			}
			v.Properties[name] = prop.Schema()
		}
		v.Required = required
		return
	}
	// array
	if element.isArray() {
		v.Items = element.Properties[""].Schema()
		return
	}
	// map
	if element.isAdditional() {
		v.AdditionalProperties = element.Properties[""].Schema()
		return
	}
	return
}
