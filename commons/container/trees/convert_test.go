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

package trees_test

import (
	"encoding/json"
	"fmt"
	"github.com/aacfactory/fns/commons/container/trees"
	"testing"
)

type Node struct {
	Id       string `tree:"Parent+Children"`
	Parent   string
	Children []*Node
}

func TestConvertListToTree(t *testing.T) {
	nodes := []*Node{
		{Id: "A", Parent: ""},
		{Id: "A1", Parent: "A"},
		{Id: "A2", Parent: "A"},
		{Id: "A11", Parent: "A1"},
		{Id: "B", Parent: ""},
		{Id: "B1", Parent: "B"},
		{Id: "B1", Parent: "B"},
	}
	v, treesErr := trees.ConvertListToTree[*Node](nodes)
	if treesErr != nil {
		t.Errorf("%+v", treesErr)
		return
	}
	p, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	fmt.Println(string(p))
}
