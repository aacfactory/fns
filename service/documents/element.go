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
	"github.com/aacfactory/fns/service/internal/oas"
	"reflect"
	"sort"
	"strings"
)

func NewElement(path string, name string, typ string, format string, title string, description string) *Element {
	return &Element{
		Path:        strings.ReplaceAll(path, "/", "."),
		Name:        name,
		Title:       title,
		Description: description,
		Type:        typ,
		Format:      format,
		Enum:        make([]interface{}, 0, 1),
		Required:    false,
		Validation:  nil,
		Properties:  make(map[string]*Element),
		Additional:  false,
		Deprecated:  false,
	}
}

func String() *Element {
	return NewElement("_", "string", "string", "", "String", "String")
}

func Bytes() *Element {
	return NewElement("_", "bytes", "string", "byte", "Bytes", "Bytes")
}

func Bool() *Element {
	return NewElement("_", "bool", "boolean", "", "Bool", "Bool")
}

func Int() *Element {
	return Int64()
}

func Int32() *Element {
	return NewElement("_", "int32", "integer", "int32", "Int32", "Int32")
}

func Int64() *Element {
	return NewElement("_", "int64", "integer", "int64", "Int64", "Int64")
}

func Uint() *Element {
	return Uint64()
}

func Uint8() *Element {
	return NewElement("_", "uint8", "integer", "int32", "Uint8", "Uint8")
}

func Uint32() *Element {
	return NewElement("_", "uint32", "integer", "int32", "Uint32", "Uint32")
}

func Uint64() *Element {
	return NewElement("_", "uint64", "integer", "int64", "Uint64", "Uint64")
}

func Float32() *Element {
	return NewElement("_", "float32", "number", "float", "Float", "Float")
}

func Float64() *Element {
	return NewElement("_", "float64", "number", "double", "Double", "Double")
}

func Complex64() *Element {
	return NewElement("_", "complex64", "string", "", "Complex64", "Complex64").SetExample("15+3i")
}

func Complex128() *Element {
	return NewElement("_", "complex128", "string", "", "Complex128", "Complex128").SetExample("15+3i")
}

func Date() *Element {
	return NewElement("_", "date", "string", "date", "Date", "Date")
}

func Time() *Element {
	return NewElement("_", "time", "string", "15:04:05", "Time", "Time value").SetExample("19:13:07")
}

func DateTime() *Element {
	return NewElement("_", "datetime", "string", "2006-01-02T15:04:05Z07:00", "Datetime", "RFC3339").SetExample("2022-01-10T19:13:07+08:00")
}

func Struct(path string, name string, title string, description string) *Element {
	return NewElement(path, name, "object", "", title, description)
}

func Ident(path string, name string, title string, description string, target *Element) *Element {
	rs := reflect.Indirect(reflect.ValueOf(target))
	rv := reflect.New(rs.Type())
	rv.Elem().Set(rs)
	v := rv.Interface().(*Element)
	v.Path = path
	v.Name = name
	v.Title = title
	v.Description = description
	return v
}

func Ref(path string, name string) *Element {
	return NewElement(path, name, "ref", "", "", "")
}

func JsonRaw() *Element {
	v := NewElement("github.com/aacfactory/json", "RawMessage", "object", "", "JsonRawMessage", "Json Raw Message")
	v.Additional = true
	v.AddProperty("", Empty())
	return v
}

func Empty() *Element {
	return NewElement("github.com/aacfactory/fns/service", "Empty", "object", "", "Empty", "Empty Object")
}

func Array(path string, name string, title string, description string, item *Element) *Element {
	v := NewElement(path, name, "array", "", title, description)
	v.AddProperty("", item)
	return v
}

func Map(path string, name string, title string, description string, item *Element) *Element {
	v := NewElement(path, name, "object", "", title, description)
	v.Additional = true
	v.AddProperty("", item)
	return v
}

type ElementValidation struct {
	Name string            `json:"name,omitempty"`
	I18n map[string]string `json:"i18n,omitempty"`
}

type Element struct {
	Path        string              `json:"path,omitempty"`
	Name        string              `json:"name,omitempty"`
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Type        string              `json:"type,omitempty"`
	Format      string              `json:"format,omitempty"`
	Enum        []interface{}       `json:"enum,omitempty"`
	Required    bool                `json:"required,omitempty"`
	Validation  *ElementValidation  `json:"validation,omitempty"`
	Properties  map[string]*Element `json:"properties,omitempty"`
	Additional  bool                `json:"additional,omitempty"`
	Deprecated  bool                `json:"deprecated,omitempty"`
	Example     interface{}         `json:"example,omitempty"`
}

func (element *Element) AsRequired(validation *ElementValidation) *Element {
	element.Required = true
	element.Validation = validation
	return element
}

func (element *Element) AsDeprecated() *Element {
	element.Deprecated = true
	return element
}

func (element *Element) SetValidation(validation *ElementValidation) *Element {
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
	ok = element.Path == "_"
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

func (element *Element) isRef() (ok bool) {
	ok = element.Type == "ref"
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

func (element *Element) unpack() (elements Elements) {
	elements = make([]*Element, 0, 1)
	if element.isBuiltin() || element.isRef() {
		elements = append(elements, element)
		return
	}
	if element.isObject() {
		if element.isAdditional() {
			unpacks := element.getItem().unpack()
			element.Properties[""] = unpacks[0]
			elements = append(elements, element)
			if len(unpacks) > 1 {
				elements = append(elements, unpacks[1:]...)
			}
			return
		}
		elements = append(elements, Ref(element.Path, element.Name))
		for name, property := range element.Properties {
			unpacks := property.unpack()
			element.Properties[name] = unpacks[0]
			if len(unpacks) > 1 {
				elements = append(elements, unpacks[1:]...)
			}
		}
		return
	}
	if element.isArray() {
		if element.Path != "" {
			elements = append(elements, Ref(element.Path, element.Name))
		}
		unpacks := element.getItem().unpack()
		element.Properties[""] = unpacks[0]
		elements = append(elements, element)
		if len(unpacks) > 1 {
			elements = append(elements, unpacks[1:]...)
		}
		return
	}
	return
}

func (element *Element) getItem() (v *Element) {
	v = element.Properties[""]
	return
}

func (element *Element) Key() (v string) {
	v = fmt.Sprintf("%s@%s", element.Path, element.Name)
	return
}

func (element *Element) Schema() (v *oas.Schema) {
	// fns
	if element.isRef() {
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
	description := "### Description" + "\n"
	description = description + element.Description + " "
	if element.Validation != nil {
		description = description + "\n\n***Validation***" + " "
		description = description + "`" + element.Validation.Name + "`" + " "
		if element.Validation.I18n != nil && len(element.Validation.I18n) > 0 {
			description = description + "\n"
			i18nKeys := make([]string, 0, 1)
			for i18nKey := range element.Validation.I18n {
				i18nKeys = append(i18nKeys, i18nKey)
			}
			sort.Strings(i18nKeys)
			for _, i18nKey := range i18nKeys {
				description = description + "  " + i18nKey + ": " + element.Validation.I18n[i18nKey] + "\n"
			}
		}
	}
	if element.Enum != nil && len(element.Enum) > 0 {
		description = description + "\n\n***Enum***" + " "
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
		v.Items = element.getItem().Schema()
		return
	}
	// map
	if element.isAdditional() {
		v.AdditionalProperties = element.getItem().Schema()
		return
	}
	return
}

type Elements []*Element

func (elements Elements) Len() int {
	return len(elements)
}

func (elements Elements) Less(i, j int) bool {
	return elements[i].Key() < elements[j].Key()
}

func (elements Elements) Swap(i, j int) {
	elements[i], elements[j] = elements[j], elements[i]
	return
}
