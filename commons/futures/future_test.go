package futures_test

import (
	"context"
	"fmt"
	"github.com/aacfactory/fns/commons/futures"
	"testing"
)

func TestNew(t *testing.T) {
	p, f := futures.New()
	p.Succeed(1)
	p.Close()

	r, err := f.Get(context.TODO())
	if err != nil {
		fmt.Println(err)
		return
	}
	v := 0
	err = r.Scan(&v)
	fmt.Println(v, err)
}
