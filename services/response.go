package services

import "github.com/aacfactory/json"

type Response interface {
	json.Marshaler
	Exist() (ok bool)
	Scan(v interface{}) (err error)
}
