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

package base_test

import (
	"github.com/aacfactory/fns/cmd/fns/initialization/base"
	"golang.org/x/mod/modfile"
	"path/filepath"
	"testing"
)

func TestDockerImageNameFromMod(t *testing.T) {
	t.Log(base.DockerImageNameFromMod("company.com/scope/app"))
}

func TestGoVersion(t *testing.T) {
	t.Log(modfile.GoVersionRE.MatchString("1.21.6"))
}

func TestPath(t *testing.T) {
	s := "D:/workspace/go/src/github.com/local/s1"
	t.Log(filepath.Base(s))
}
