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

package sources

import (
	"fmt"
	"go/ast"
	"strings"
)

func NewImportsFromAstFileImports(specs []*ast.ImportSpec) (v Imports) {
	v = Imports{}
	if specs == nil || len(specs) == 0 {
		return
	}
	for _, spec := range specs {
		path := strings.ReplaceAll(spec.Path.Value, "\"", "")
		alias := ""
		if spec.Name != nil && spec.Name.Name != "" {
			alias = spec.Name.Name
		}
		v.Add(&Import{
			Alias: alias,
			Path:  path,
		})
	}
	return
}

// Imports 一个fn文件一个，所以key不会重复，
type Imports map[string]*Import

func (s Imports) Find(ident string) (v *Import, has bool) {
	v, has = s[ident]
	return
}

func (s Imports) Path(path string) (v *Import, has bool) {
	for _, i := range s {
		if i.Path == path {
			v = i
			has = true
			return
		}
	}
	return
}

func (s Imports) Len() (n int) {
	n = len(s)
	return
}

func (s Imports) Add(i *Import) {
	_, has := s.Find(i.Ident())
	if !has {
		s[i.Ident()] = i
		return
	}
	return
}

type Import struct {
	Path  string
	Alias string
}

func (i *Import) Ident() (ident string) {
	if i.Alias != "" {
		ident = i.Alias
		return
	}
	ident = i.Name()
	return
}

func (i *Import) Name() (name string) {
	idx := strings.LastIndex(i.Path, "/")
	if idx < 0 {
		name = i.Path
	} else {
		name = i.Path[idx+1:]
	}
	return
}

// MergeImports 在service里增加fn的imports用
func MergeImports(ss []Imports) (v Imports) {
	idents := make(map[string]int)
	v = make(map[string]*Import)
	for _, s := range ss {
		for _, i := range s {
			_, has := v.Path(i.Path)
			if has {
				continue
			}
			vv := &Import{
				Path:  i.Path,
				Alias: "",
			}
			_, hasIdent := v.Find(vv.Name())
			if hasIdent {
				times, hasIdents := idents[vv.Ident()]
				if hasIdents {
					times++
				}
				vv.Alias = fmt.Sprintf("%s%d", vv.Name(), times)
				idents[vv.Name()] = times
			}
			v.Add(vv)
		}
	}
	return
}
