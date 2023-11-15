package caches_test

import (
	"fmt"
	"github.com/aacfactory/fns/commons/caches"
	"testing"
	"time"
)

func TestMakeKVS(t *testing.T) {
	big := [2 << 16]byte{}
	kvs := caches.MakeKVS([]byte("s"), big[:], 10*time.Second, caches.MemHash{})
	fmt.Println(len(kvs))
	fmt.Println(kvs.Deadline())
	fmt.Println(len(kvs.Value()), len(big))
	for _, kv := range kvs {
		kLen := len(kv.Key())
		vLen := len(kv.Value())
		ok := (4 + kLen + vLen) < (1 << 16)
		fmt.Println("key:", kLen, "val:", vLen, ok)
	}
}
