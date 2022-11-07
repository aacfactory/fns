package wildcard_test

import (
	"fmt"
	"github.com/aacfactory/fns/commons/wildcard"
	"testing"
)

func TestNew(t *testing.T) {
	w := wildcard.New("abc")
	fmt.Println(w.Match("abc"))
	fmt.Println(w.Match("abcd"))
}
