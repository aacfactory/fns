package transports_test

import (
	"fmt"
	"github.com/aacfactory/fns/transports"
	"strings"
	"testing"
	"time"
)

type Id int64

type Date time.Time

type Range struct {
	Offset int `json:"offset"`
	Length int `json:"length"`
}

type Param struct {
	Range
	Id    Id          `json:"id"`
	Name  string      `json:"name"`
	Score float64     `json:"score"`
	Age   uint        `json:"age"`
	Date  Date        `json:"date"`
	Dates []time.Time `json:"dates"`
}

func TestDecodeParams(t *testing.T) {
	params := transports.NewParams()
	params.Set([]byte("id"), []byte("1"))
	params.Set([]byte("name"), []byte("name"))
	params.Set([]byte("score"), []byte("99.99"))
	params.Set([]byte("age"), []byte("13"))
	params.Set([]byte("date"), []byte(time.Now().Format(time.RFC3339)))
	params.Set([]byte("dates"), []byte(strings.Join([]string{time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339)}, ",")))
	//params.Add([]byte("dates"), []byte(time.Now().Format(time.RFC3339)))
	//params.Add([]byte("dates"), []byte(time.Now().Format(time.RFC3339)))

	params.Set([]byte("offset"), []byte("10"))
	params.Set([]byte("length"), []byte("50"))

	fmt.Println(string(params.Encode()))

	param := Param{}
	decodeErr := transports.DecodeParams(params, &param)
	if decodeErr != nil {
		fmt.Println(fmt.Sprintf("%+v", decodeErr))
		return
	}
	fmt.Println(fmt.Sprintf("%+v", param))
}
