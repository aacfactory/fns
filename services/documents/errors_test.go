package documents_test

import (
	"fmt"
	"github.com/aacfactory/fns/services/documents"
	"testing"
)

func TestNewErrors(t *testing.T) {
	s := `
	name1
	zh: chinese
	en: english
	name2
	zh: chinese
	en: english`
	errs := documents.NewErrors(s)
	for _, err := range errs {
		fmt.Println(err.Name)
		for _, description := range err.Descriptions {
			fmt.Println(description.Name, description.Value)
		}
		fmt.Println("--")
	}
}
