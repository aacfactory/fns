package context_test

import (
	"bytes"
	sc "context"
	"fmt"
	"github.com/aacfactory/fns/context"
	"testing"
)

func BenchmarkE(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bytes.Equal([]byte{'1'}, []byte{'2'})
	}
}

func BenchmarkEs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bytes.Equal([]byte{'1'}, []byte{'2'})
	}
}

func TestAcquire(t *testing.T) {
	ctx := context.Acquire(sc.Background())
	ctx = context.WithValue(ctx, []byte{'1'}, 1)
	set(ctx)
	ctx.UserValues(func(key []byte, val any) {
		fmt.Println(string(key), val)
	})
	context.Release(ctx)
}

func set(ctx context.Context) {
	context.WithValue(ctx, []byte{'2'}, 2)
}