package log

import (
	"github.com/aacfactory/configures"
	"io"
	"os"
)

type WriterOptions struct {
	Config configures.Config
}

type Writer interface {
	io.WriteCloser
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

func (s *StdOut) Close() error {
	return nil
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

func (s *StdErr) Close() error {
	return nil
}

func (s *StdErr) Construct(_ WriterOptions) (err error) {
	s.out = os.Stderr
	return nil
}
