/*
 * Copyright 2023 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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
