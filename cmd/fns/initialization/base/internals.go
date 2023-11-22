package base

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/files"
	"os"
	"path/filepath"
)

func NewInternalGeneratorsFile(dir string) (f *InternalGeneratorsFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fns: new generator file failed").WithCause(err).WithMeta("dir", dir)
			return
		}
	}
	dir = filepath.ToSlash(filepath.Join(dir, "internal", "generator"))

	f = &InternalGeneratorsFile{
		dir:      dir,
		filename: filepath.ToSlash(filepath.Join(dir, "main.go")),
	}
	return
}

type InternalGeneratorsFile struct {
	filename string
	dir      string
}

func (f *InternalGeneratorsFile) Name() (name string) {
	name = f.dir
	return
}

func (f *InternalGeneratorsFile) Write(_ context.Context) (err error) {
	if !files.ExistFile(f.dir) {
		mdErr := os.MkdirAll(f.dir, 0644)
		if mdErr != nil {
			err = errors.Warning("fns: generator file write failed").WithCause(mdErr).WithMeta("dir", f.dir)
			return
		}
	}
	const (
		content = `package main

import (
	"context"
	"fmt"
	"github.com/aacfactory/fns/cmd/generates"
	"os"
)

func main() {
	g := generates.New("generator")
	if err := g.Execute(context.Background(), os.Args...); err != nil {
		fmt.Println(fmt.Sprintf("%+v", err))
	}
}

`
	)
	writeErr := os.WriteFile(f.filename, []byte(content), 0644)
	if writeErr != nil {
		err = errors.Warning("fns: generator file write failed").WithCause(writeErr).WithMeta("filename", f.filename)
		return
	}
	return
}
