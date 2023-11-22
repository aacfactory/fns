package base

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/files"
	"os"
	"path/filepath"
)

func NewRepositoryFile(dir string) (hooks *RepositoryFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fns: new repositories file failed").WithCause(err).WithMeta("dir", dir)
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

func (f *RepositoryFile) Write(_ context.Context) (err error) {
	if !files.ExistFile(f.dir) {
		mdErr := os.MkdirAll(f.dir, 0644)
		if mdErr != nil {
			err = errors.Warning("fns: repositories file write failed").WithCause(mdErr).WithMeta("dir", f.dir)
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
		err = errors.Warning("fns: repositories file write failed").WithCause(writeErr).WithMeta("filename", f.filename)
		return
	}
	return
}
