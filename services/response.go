package services

import (
	"github.com/aacfactory/fns/commons/scanner"
)

type Response interface {
	scanner.Scanner
}

func NewResponse(src interface{}) Response {
	return scanner.New(src)
}
