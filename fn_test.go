package fns_test

import (
	"fmt"
	"github.com/aacfactory/fns"
	"reflect"
	"testing"
)

type ArgA struct {
	Id    string
	Names []string
	File  fns.FnFile
	Files []fns.FnFile
}

func TestFnArguments_Scan(t *testing.T) {

	a := &ArgA{}

	_type := reflect.TypeOf(a)

	elemType := _type.Elem()

	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		fmt.Println(field.Type.String())
		fmt.Println(field.Type == reflect.TypeOf(fns.FnFile{}), field.Type == reflect.TypeOf([]fns.FnFile{}))
	}

	fmt.Println("xxx", reflect.TypeOf([]string{}) == reflect.TypeOf(make([]string, 0, 1)))

}
