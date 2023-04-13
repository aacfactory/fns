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
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/gcg"
	"os"
	"path/filepath"
)

func NewDeploysFile(dir string, services Services) (file CodeFileWriter) {
	file = &DeploysFile{
		filename: filepath.ToSlash(filepath.Join(dir, "fns.go")),
		services: services,
	}
	return
}

type DeploysFile struct {
	filename string
	services Services
}

func (s *DeploysFile) Name() (name string) {
	name = s.filename
	return
}

func (s *DeploysFile) Write(ctx context.Context) (err error) {
	if s.filename == "" {
		return
	}
	if ctx.Err() != nil {
		err = errors.Warning("sources: services write failed").
			WithMeta("kind", "services").WithMeta("file", s.Name()).
			WithCause(ctx.Err())
		return
	}

	file := gcg.NewFileWithoutNote("modules")
	file.FileComments("NOTE: this file has been automatically generated, DON'T EDIT IT!!!\n")

	fn := gcg.Func()
	fn.Name("services")
	fn.AddResult("v", gcg.Token("[]service.Service", gcg.NewPackage("github.com/aacfactory/fns/service")))
	body := gcg.Statements()
	if s.services != nil && s.services.Len() > 0 {
		body.Token("v = []service.Service{").Line()
		for _, service := range s.services {
			body.Tab().Token(fmt.Sprintf("%s.Service()", service.PathIdent), gcg.NewPackage(service.Path)).Symbol(",").Line()
		}
		body.Token("}").Line()
	}
	body.Return()
	fn.Body(body)
	file.AddCode(fn.Build())

	buf := bytes.NewBuffer([]byte{})

	renderErr := file.Render(buf)
	if renderErr != nil {
		err = errors.Warning("sources: services code file write failed").
			WithMeta("kind", "services").WithMeta("file", s.Name()).
			WithCause(renderErr)
		return
	}

	writer, openErr := os.OpenFile(s.Name(), os.O_CREATE|os.O_TRUNC|os.O_RDWR|os.O_SYNC, 0644)
	if openErr != nil {
		err = errors.Warning("sources: services code file write failed").
			WithMeta("kind", "services").WithMeta("file", s.Name()).
			WithCause(openErr)
		return
	}

	n := 0
	bodyLen := buf.Len()
	content := buf.Bytes()
	for n < bodyLen {
		nn, writeErr := writer.Write(content[n:])
		if writeErr != nil {
			err = errors.Warning("sources: services code file write failed").
				WithMeta("kind", "services").WithMeta("file", s.Name()).
				WithCause(writeErr)
			return
		}
		n += nn
	}
	syncErr := writer.Sync()
	if syncErr != nil {
		err = errors.Warning("sources: services code file write failed").
			WithMeta("kind", "services").WithMeta("file", s.Name()).
			WithCause(syncErr)
		return
	}
	closeErr := writer.Close()
	if closeErr != nil {
		err = errors.Warning("sources: services code file write failed").
			WithMeta("kind", "services").WithMeta("file", s.Name()).
			WithCause(closeErr)
		return
	}
	return
}
