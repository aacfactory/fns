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

package fns

import (
	"os"
	"path"
	"strings"
)

var gopaths = make([]string, 0, 2)

func initEnv() {
	// goroot
	goroot := os.Getenv("GOROOT")
	if len(goroot) > 0 {
		if strings.Contains(goroot, `\`) && strings.Contains(goroot, ":") { // win
			goroot = strings.Replace(goroot, `\`, "/", -1)
		}
		gopaths = append(gopaths, goroot)
	}
	gopath := os.Getenv("GOPATH")
	if len(gopath) == 0 {
		return
	}
	if strings.Contains(gopath, `\`) && strings.Contains(gopath, ":") { // win
		gopath = strings.Replace(gopath, `\`, "/", -1)
		if strings.Contains(gopath, ";") {
			gopaths := strings.Split(gopath, ";")
			for _, item := range gopaths {
				gopaths = append(gopaths, strings.TrimSpace(item))
			}
		} else {
			gopaths = append(gopaths, strings.TrimSpace(gopath))
		}
	} else { // unix
		if strings.Contains(gopath, ":") {
			gopaths := strings.Split(gopath, ":")
			for _, item := range gopaths {
				gopaths = append(gopaths, strings.TrimSpace(item))
			}
		} else {
			gopaths = append(gopaths, strings.TrimSpace(gopath))
		}
	}
}

func goEnv() []string {
	return gopaths
}

func fileNameSubGoPath(src string) (file string) {
	file = src
	goHomes := goEnv()
	if goHomes == nil {
		return
	}
	for _, goHome := range goHomes {
		if strings.Contains(file, goHome) {
			file = strings.Replace(file, path.Join(goHome, "src"), "", 1)[1:]
			return
		}
	}
	return
}
