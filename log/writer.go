package log

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
