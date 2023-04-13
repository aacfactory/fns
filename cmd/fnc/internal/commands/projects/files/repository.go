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

package files

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/fnc/internal/libs/files"
	"os"
	"path/filepath"
)

func NewRepositoryFile(dir string) (hooks *RepositoryFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fnc: new repositories file failed").WithCause(err).WithMeta("dir", dir)
			return
		}
	}
	dir = filepath.ToSlash(filepath.Join(dir, "repositories"))

	hooks = &RepositoryFile{
		dir:      dir,
		filename: filepath.ToSlash(filepath.Join(dir, "doc.go")),
	}
	return
}

type RepositoryFile struct {
	dir      string
	filename string
}

func (f *RepositoryFile) Name() (name string) {
	name = f.filename
	return
}

func (f *RepositoryFile) Write(ctx context.Context) (err error) {
	if !files.ExistFile(f.dir) {
		mdErr := os.MkdirAll(f.dir, 0644)
		if mdErr != nil {
			err = errors.Warning("fnc: repositories file write failed").WithCause(mdErr).WithMeta("dir", f.dir)
			return
		}
	}
	const (
		content = `// Package repositories
// read https://github.com/aacfactory/fns-contrib/tree/main/databases/sql for more details.
package repositories`
	)
	writeErr := os.WriteFile(f.filename, []byte(content), 0644)
	if writeErr != nil {
		err = errors.Warning("fnc: repositories file write failed").WithCause(writeErr).WithMeta("filename", f.filename)
		return
	}
	return
}
