package context_test

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"testing"
)

func BenchmarkE(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bytex.Equal([]byte{'1'}, []byte{'2'})
	}
}

func BenchmarkEs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bytes.Equal([]byte{'1'}, []byte{'2'})
	}
}
