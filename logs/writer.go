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

package logs

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/fns/context"
	"io"
	"os"
)

type WriterOptions struct {
	Config configures.Config
}

type Writer interface {
	io.Writer
	Shutdown(ctx context.Context)
	Construct(options WriterOptions) (err error)
}

var (
	writers = map[string]Writer{
		"stdout": new(StdOut),
		"stderr": new(StdErr),
	}
)

func RegisterWriter(name string, writer Writer) {
	writers[name] = writer
}

func getWriter(name string) (writer Writer, has bool) {
	if name == "" {
		name = "stdout"
	}
	writer, has = writers[name]
	return
}

type StdOut struct {
	out *os.File
}

func (s *StdOut) Write(p []byte) (n int, err error) {
	n, err = s.out.Write(p)
	return
}

func (s *StdOut) Shutdown(_ context.Context) {
}

func (s *StdOut) Construct(_ WriterOptions) (err error) {
	s.out = os.Stdout
	return nil
}

type StdErr struct {
	out *os.File
}

func (s *StdErr) Write(p []byte) (n int, err error) {
	n, err = s.out.Write(p)
	return
}

func (s *StdErr) Shutdown(_ context.Context) {
}

func (s *StdErr) Construct(_ WriterOptions) (err error) {
	s.out = os.Stderr
	return nil
}
