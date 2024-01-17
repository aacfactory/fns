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

package versions_test

import (
	"github.com/aacfactory/fns/commons/versions"
	"testing"
)

func TestParseMajor(t *testing.T) {
	v, err := versions.Parse([]byte("v1"))
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(v.String())
}

func TestParseMajorMiner(t *testing.T) {
	v, err := versions.Parse([]byte("v1.0"))
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(v.String())
}

func TestParseMajorMinerPatch(t *testing.T) {
	v, err := versions.Parse([]byte("v1.0.1"))
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(v.String())
}

func TestVersion_Equals(t *testing.T) {
	t.Log(versions.New(1, 0, 0), versions.New(1, -1, -1))
	t.Log(versions.New(1, 0, 0).Equals(versions.New(1, -1, -1)))
}
