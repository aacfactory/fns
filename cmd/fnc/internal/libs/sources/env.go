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

package sources

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/fnc/internal/libs/files"
	"os"
	"path/filepath"
	"strings"
)

func GOPATH() (gopath string, has bool) {
	gopath, has = os.LookupEnv("GOPATH")
	if has {
		gopath = strings.TrimSpace(gopath)
		has = gopath != ""
	}
	return
}

func GOROOT() (goroot string, has bool) {
	goroot, has = os.LookupEnv("GOROOT")
	if has {
		goroot = strings.TrimSpace(goroot)
		has = goroot != ""
	}
	return
}

var pkgDir = ""

func initPkgDir() (err error) {
	gopath, hasGOPATH := GOPATH()
	if hasGOPATH {
		pkgDir = filepath.ToSlash(filepath.Join(gopath, "pkg/mod"))
		if !files.ExistFile(pkgDir) {
			pkgDir = ""
			err = errors.Warning("sources: GOPATH was found but no 'pkg/mod' dir")
			return
		}
		return
	}
	goroot, hasGOROOT := GOROOT()
	if hasGOROOT {
		pkgDir = filepath.ToSlash(filepath.Join(goroot, "pkg/mod"))
		if !files.ExistFile(pkgDir) {
			pkgDir = ""
			err = errors.Warning("sources: GOROOT was found but no 'pkg/mod' dir")
			return
		}
		return
	}
	if !hasGOPATH && !hasGOROOT {
		err = errors.Warning("sources: GOPATH and GOROOT were not found")
		return
	}
	return
}

func PKG() (pkg string) {
	pkg = pkgDir
	return
}
