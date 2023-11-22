package base

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func NewModFile(path string, dir string) (f *ModFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fns: new mod file failed").WithCause(err).WithMeta("dir", dir)
			return
		}
	}
	f = &ModFile{
		path:     path,
		filename: filepath.ToSlash(filepath.Join(dir, "go.mod")),
	}
	return
}

type ModFile struct {
	path     string
	filename string
}

func (f *ModFile) Name() (name string) {
	name = f.filename
	return
}

func (f *ModFile) Write(ctx context.Context) (err error) {
	mf := &modfile.File{}
	pathErr := mf.AddModuleStmt(f.path)
	if pathErr != nil {
		err = errors.Warning("fns: mod file write failed").WithCause(pathErr).WithMeta("filename", f.filename)
		return
	}
	goVersion := runtime.Version()[2:]
	goVersionItems := strings.Split(goVersion, ".")
	if len(goVersionItems) < 2 {
		err = errors.Warning("fns: mod file write failed").WithCause(errors.Warning("invalid go runtime version").WithMeta("version", runtime.Version())).WithMeta("filename", f.filename)
		return
	}
	versionErr := mf.AddGoStmt(strings.Join(goVersionItems[0:2], "."))
	if versionErr != nil {
		err = errors.Warning("fns: mod file write failed").WithCause(versionErr).WithMeta("filename", f.filename)
		return
	}
	requires := []string{"github.com/aacfactory/fns"}
	for _, require := range requires {
		requireVersion, requireVersionErr := sources.LatestVersion(require)
		if requireVersionErr != nil {
			err = errors.Warning("fns: mod file write failed").WithCause(requireVersionErr).WithMeta("filename", f.filename)
			return
		}
		fnsRequireErr := mf.AddRequire(require, requireVersion)
		if fnsRequireErr != nil {
			err = errors.Warning("fns: mod file write failed").WithCause(fnsRequireErr).WithMeta("filename", f.filename)
			return
		}
	}
	p, encodeErr := mf.Format()
	if encodeErr != nil {
		err = errors.Warning("fns: mod file write failed").WithCause(encodeErr).WithMeta("filename", f.filename)
		return
	}
	writeErr := os.WriteFile(f.filename, p, 0644)
	if writeErr != nil {
		err = errors.Warning("fns: mod file write failed").WithCause(writeErr).WithMeta("filename", f.filename)
		return
	}
	return
}
