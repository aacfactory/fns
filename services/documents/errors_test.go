package documents_test

import (
	"fmt"
	"github.com/aacfactory/fns/services/documents"
	"testing"
)

func TestNewErrors(t *testing.T) {
	s := "user_not_found\nzh: zh_message\nen: en_message"
	errs := documents.NewErrors(s)
	for _, err := range errs {
		fmt.Println(err.Name)
		for _, description := range err.Descriptions {
			fmt.Println(description.Name, description.Value)
		}
		fmt.Println("--")
	}
}
