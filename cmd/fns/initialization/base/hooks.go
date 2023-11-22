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

package base

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/files"
	"os"
	"path/filepath"
)

func NewHooksFile(dir string) (f *HooksFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fns: new hooks file failed").WithCause(err).WithMeta("dir", dir)
			return
		}
	}
	dir = filepath.ToSlash(filepath.Join(dir, "hooks"))
	f = &HooksFile{
		dir:      dir,
		filename: filepath.ToSlash(filepath.Join(dir, "doc.go")),
	}
	return
}

type HooksFile struct {
	dir      string
	filename string
}

func (f *HooksFile) Name() (name string) {
	name = f.filename
	return
}

func (f *HooksFile) Write(_ context.Context) (err error) {
	if !files.ExistFile(f.dir) {
		mdErr := os.MkdirAll(f.dir, 0644)
		if mdErr != nil {
			err = errors.Warning("fnc: hooks file write failed").WithCause(mdErr).WithMeta("dir", f.dir)
			return
		}
	}
	const (
		content = `// Package hooks
// read https://github.com/aacfactory/fns/blob/main/docs/hooks.md for more details.
package hooks`
	)
	writeErr := os.WriteFile(f.filename, []byte(content), 0644)
	if writeErr != nil {
		err = errors.Warning("fns: hooks file write failed").WithCause(writeErr).WithMeta("filename", f.filename)
		return
	}
	return
}
