package bmap_test

import (
	"fmt"
	"github.com/aacfactory/fns/commons/container/bmap"
	"testing"
)

func TestNewBMap(t *testing.T) {
	bm := bmap.New[string, []byte]()
	bm.Set("a", []byte("a"))
	bm.Set("c", []byte("c"))
	bm.Add("b", []byte("b1"))
	bm.Add("b", []byte("b2"))
	bm.Foreach(func(key string, values [][]byte) {
		fmt.Println(key)
	})
	v, has := bm.Get("c")
	fmt.Println(string(v), has)
	bm.Remove("c")
	v, has = bm.Get("c")
	fmt.Println(string(v), has)
	vv, has := bm.Values("b")
	fmt.Println(vv, has)
}
