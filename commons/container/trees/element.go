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

package trees

import (
	"fmt"
	"reflect"
	"strings"
)

func NewElement[T any](v T) (element *Element[T], err error) {
	rv := reflect.ValueOf(v)
	ptr := rv.Type().Kind() == reflect.Ptr
	if !ptr {
		rv = reflect.ValueOf(&v)
	}
	ok := false
	identFieldName := ""
	parentFieldName := ""
	childrenFieldName := ""
	childPtr := false
	rt := reflect.Indirect(rv).Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag, has := field.Tag.Lookup(treeTag)
		if !has {
			continue
		}
		idx := strings.Index(tag, "+")
		if idx < 1 || idx == len(tag)-1 {
			err = fmt.Errorf("tree tag is invalid")
			return
		}
		// parent
		parentName := strings.TrimSpace(tag[0:idx])
		parentField, hasParentField := rt.FieldByName(parentName)
		if !hasParentField {
			err = fmt.Errorf("parent field was not found")
			return
		}
		if !parentField.IsExported() {
			err = fmt.Errorf("parent field was not exported")
			return
		}
		if parentField.Anonymous {
			err = fmt.Errorf("parent field can not be anonymous")
			return
		}
		if parentField.Type != field.Type {
			err = fmt.Errorf("type of parent field does not match ident field type")
			return
		}
		parentFieldName = parentName
		// children
		childrenName := strings.TrimSpace(tag[idx+1:])
		childrenField, hasChildrenField := rt.FieldByName(childrenName)
		if !hasChildrenField {
			err = fmt.Errorf("children field was not found")
			return
		}
		if !childrenField.IsExported() {
			err = fmt.Errorf("children field was not exported")
			return
		}
		if childrenField.Anonymous {
			err = fmt.Errorf("children field can not be anonymous")
			return
		}
		if childrenField.Type.Kind() != reflect.Slice {
			err = fmt.Errorf("children field was not slice")
			return
		}
		childrenElementType := childrenField.Type.Elem()
		childPtr = childrenElementType.Kind() == reflect.Ptr
		if childPtr {
			childrenElementType = childrenElementType.Elem()
		}
		if childrenElementType.PkgPath() != rt.PkgPath() && childrenElementType.Name() != rt.Name() {
			err = fmt.Errorf("element type of children field was not matched")
			return
		}
		childrenFieldName = childrenName
		// ident
		identFieldName = field.Name
		ok = true
		break
	}
	if !ok {
		err = fmt.Errorf("%s.%s is not tree struct", rt.PkgPath(), rt.Name())
		return
	}
	element = &Element[T]{
		ptr:           ptr,
		childPtr:      childPtr,
		value:         rv,
		identField:    identFieldName,
		parentField:   parentFieldName,
		childrenField: childrenFieldName,
		children:      make(Elements[T], 0, 1),
		hasParent:     false,
	}
	return
}

type Element[T any] struct {
	ptr           bool
	childPtr      bool
	value         reflect.Value
	identField    string
	parentField   string
	childrenField string
	children      Elements[T]
	hasParent     bool
}

func (element *Element[T]) ident() reflect.Value {
	return element.value.Elem().FieldByName(element.identField)
}

func (element *Element[T]) parent() reflect.Value {
	return element.value.Elem().FieldByName(element.parentField)
}

func (element *Element[T]) Interface() (v T) {
	children := reflect.MakeSlice(element.value.Elem().FieldByName(element.childrenField).Type(), 0, 1)
	for _, child := range element.children {
		cv := child.Interface()
		if element.childPtr {
			if element.ptr {
				children = reflect.Append(children, reflect.ValueOf(cv))
			} else {
				children = reflect.Append(children, reflect.ValueOf(&cv))
			}
		} else {
			if element.ptr {
				children = reflect.Append(children, reflect.ValueOf(cv).Elem())
			} else {
				children = reflect.Append(children, reflect.ValueOf(cv))
			}
		}
	}
	if children.Len() > 0 {
		element.value.Elem().FieldByName(element.childrenField).Set(children)
	}

	if element.ptr {
		v = element.value.Interface().(T)
		return
	}
	v = element.value.Elem().Interface().(T)
	return
}

func (element *Element[T]) contains(child *Element[T]) (ok bool) {
	for _, e := range element.children {
		if e.value.Equal(child.value) {
			ok = true
			break
		}
	}
	return
}

type Elements[T any] []*Element[T]

func (elements Elements[T]) Len() int {
	return len(elements)
}

func (elements Elements[T]) Less(i, j int) bool {
	switch elements[i].ident().Type().Kind() {
	case reflect.String:
		return elements[i].ident().String() < elements[j].ident().String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return elements[i].ident().Int() < elements[j].ident().Int()
	case reflect.Float32, reflect.Float64:
		return elements[i].ident().Float() < elements[j].ident().Float()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return elements[i].ident().Uint() < elements[j].ident().Uint()
	case reflect.Uintptr:
		return elements[i].ident().Interface().(uintptr) < elements[j].ident().Interface().(uintptr)
	default:
		return false
	}
}

func (elements Elements[T]) Swap(i, j int) {
	elements[i], elements[j] = elements[j], elements[i]
}
