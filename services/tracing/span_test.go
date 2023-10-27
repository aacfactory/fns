package tracing_test

import (
	"fmt"
	"github.com/aacfactory/fns/services/tracing"
	"testing"
)

func TestSpan(t *testing.T) {
	fmt.Println(1%2, 2%2, 3%2, 4%2, 0%2)
}

func TestTags_Merge(t *testing.T) {
	tags := make(tracing.Tags)
	ss := []string{"a", "a", "b", "b", "c", "c"}
	fmt.Println(tags.Merge(ss))
	fmt.Println(tags)
}
