package futures_test

import (
	"context"
	"fmt"
	"github.com/aacfactory/fns/commons/futures"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	p, f := futures.New(func() {
		fmt.Println("cb")
	})
	go func(wg *sync.WaitGroup, p futures.Promise) {
		p.Succeed(1)
		wg.Done()
	}(wg, p)

	wg.Wait()
	r, err := f.Get(context.TODO())
	if err != nil {
		fmt.Println(err)
		return
	}
	v := 0
	err = r.Scan(&v)
	fmt.Println(v, err)
}

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p, f := futures.New()
		p.Succeed(1)
		_, _ = f.Get(context.TODO())
	}
}

func BenchmarkCh(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := make(chan int, 1)
		close(s)
	}
}
