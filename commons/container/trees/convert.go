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
	"github.com/aacfactory/errors"
	"reflect"
	"sort"
	"strings"
)

const (
	treeTag = "tree"
)

var (
	sortType = reflect.TypeOf((*sort.Interface)(nil)).Elem()
)

func ConvertListToTree[T any](items []T) (v []T, err error) {
	if items == nil || len(items) == 0 {
		return
	}
	itemsType := reflect.TypeOf(items)
	elementType := reflect.TypeOf(items[0])
	if elementType.Kind() != reflect.Ptr {
		err = errors.Warning("fns: convert list to tree failed").WithCause(errors.Warning("element type must be ptr"))
		return
	}
	rt := elementType.Elem()
	fieldNum := rt.NumField()
	if fieldNum == 0 {
		err = errors.Warning("fns: convert list to tree failed").WithCause(errors.Warning("no fields"))
		return
	}
	nodeFieldName := ""
	parentFieldName := ""
	childrenFieldName := ""
	for i := 0; i < fieldNum; i++ {
		field := rt.Field(i)
		tag, hasTag := field.Tag.Lookup(treeTag)
		if !hasTag {
			continue
		}
		values := strings.Split(tag, "+")
		if len(values) != 2 {
			err = errors.Warning("fns: convert list to tree failed").WithCause(errors.Warning("tree tag is invalid, value must be parentFieldName+childrenFieldName"))
			return
		}
		nodeFieldName = field.Name
		parentFieldName = strings.TrimSpace(values[0])
		parentField, hasParentField := rt.FieldByName(parentFieldName)
		if !hasParentField {
			err = errors.Warning("fns: convert list to tree failed").WithCause(errors.Warning("parent field was bot found"))
			return
		}
		if !parentField.IsExported() {
			err = errors.Warning("fns: convert list to tree failed").WithCause(errors.Warning("parent field type must be exported"))
			return
		}
		if parentField.Type != field.Type {
			err = errors.Warning("fns: convert list to tree failed").WithCause(errors.Warning("parent field type must be node field type"))
			return
		}
		childrenFieldName = strings.TrimSpace(values[1])
		childrenField, hasChildrenField := rt.FieldByName(childrenFieldName)
		if !hasChildrenField {
			err = errors.Warning("fns: convert list to tree failed").WithCause(errors.Warning("children field was bot found"))
			return
		}
		if !childrenField.IsExported() {
			err = errors.Warning("fns: convert list to tree failed").WithCause(errors.Warning("children field type must be exported"))
			return
		}
		if childrenField.Type != itemsType && !childrenField.Type.ConvertibleTo(itemsType) {
			err = errors.Warning("fns: convert list to tree failed").WithCause(errors.Warning("children field type must be list node type"))
			return
		}
	}
	itemValues := make([]reflect.Value, 0, 1)
	for _, item := range items {
		rv := reflect.ValueOf(item)
		if rv.IsValid() {
			itemValues = append(itemValues, rv)
		}
	}
	rootNodeValues := make([]reflect.Value, 0, 1)
	for _, value := range itemValues {
		parentField := reflect.Indirect(value).FieldByName(parentFieldName)
		if parentField.IsZero() {
			rootNodeValues = append(rootNodeValues, value)
		}
	}
	if len(rootNodeValues) == 0 {
		err = errors.Warning("fns: convert list to tree failed").WithCause(errors.Warning("no root value in list"))
		return
	}
	v = make([]T, 0, len(rootNodeValues))
	for _, value := range rootNodeValues {
		convertTreeNode[T](itemValues, value, nodeFieldName, parentFieldName, childrenFieldName)
		root, ok := value.Interface().(T)
		if !ok {
			err = errors.Warning("fns: convert list to tree failed").WithCause(errors.Warning("type was not matched"))
			return
		}
		v = append(v, root)
	}
	return
}

func convertTreeNode[T any](values []reflect.Value, node reflect.Value, nodeFieldName string, parentFieldName string, childrenFieldName string) {
	nodeField := reflect.Indirect(node).FieldByName(nodeFieldName)
	nodeValue := nodeField.Interface()
	children := make([]T, 0, 1)
	for _, value := range values {
		parentField := reflect.Indirect(value).FieldByName(parentFieldName)
		parentValue := parentField.Interface()
		if nodeValue == parentValue {
			convertTreeNode[T](values, value, nodeFieldName, parentFieldName, childrenFieldName)
			child, ok := value.Interface().(T)
			if !ok {
				continue
			}
			children = append(children, child)
		}
	}
	childrenField := reflect.Indirect(node).FieldByName(childrenFieldName)
	childrenField.Set(reflect.ValueOf(children))
	if childrenField.CanConvert(sortType) {
		sortable := childrenField.Convert(sortType).Interface().(sort.Interface)
		sort.Sort(sortable)
	}
	return
}
