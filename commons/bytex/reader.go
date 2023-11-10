package bytex

import (
	"bytes"
	"io"
)

func NewReadCloser(p []byte) io.ReadCloser {
	return &ReadCloser{
		Reader: bytes.NewReader(p),
	}
}

type ReadCloser struct {
	*bytes.Reader
}

func (r *ReadCloser) Close() error {
	return nil
}
