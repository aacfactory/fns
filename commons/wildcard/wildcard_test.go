package wildcard_test

import (
	"fmt"
	"github.com/aacfactory/fns/commons/wildcard"
	"testing"
)

func TestNew(t *testing.T) {
	w := wildcard.New([]byte("abc"))
	fmt.Println(w.Match([]byte("abc")))
	fmt.Println(w.Match([]byte("abcd")))
}
