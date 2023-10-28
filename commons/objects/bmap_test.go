package objects_test

import (
	"fmt"
	"github.com/aacfactory/fns/commons/objects"
	"testing"
)

func TestNewBMap(t *testing.T) {
	bm := objects.NewBMap()
	bm.Set([]byte("a"), []byte("a"))
	bm.Add([]byte("b"), []byte("b1"))
	bm.Add([]byte("b"), []byte("b2"))
	bm.Set([]byte("c"), []byte("c"))
	v, has := bm.Get([]byte("c"))
	fmt.Println(string(v), has)
	bm.Remove([]byte("c"))
	v, has = bm.Get([]byte("c"))
	fmt.Println(string(v), has)
	vv, has := bm.Values([]byte("b"))
	fmt.Println(vv, has)
}
