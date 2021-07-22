/*
 * Copyright 2021 Wang Min Xiang
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
 */

package fns_test

import (
	"github.com/aacfactory/fns"
	"strings"
	"testing"
)

func TestNewJsonObject(t *testing.T) {
	o := fns.NewJsonObject()
	_ = o.Put("id", "id")
	t.Log(o.Contains("id"))
	_ = o.Rem("id")
	t.Log(o.Contains("id"))
	m := make(map[string]string)
	m["1"] = "1"
	m["2"] = "2"
	_ = o.Put("m", m)
	nm := make(map[string]string)
	getErr := o.Get("m", &nm)
	t.Log(getErr, nm)
}

func TestNewJsonArray(t *testing.T) {
	a := fns.NewJsonArray()
	_ = a.Add(1, 2, 3, 4, 5)
	t.Log(a.Len(), string(a.Raw()))
	_ = a.Rem(1)
	t.Log(a.Len(), string(a.Raw()))
	n0 := 0
	_ = a.Get(0, &n0)
	t.Log(n0)
	t.Log(strings.Index("fns-", "fns"))
}

type FooTimed struct {
	Time fns.DateTime `json:"time,omitempty"`
}

func TestDateTime_MarshalJSON(t *testing.T) {

	foo := &FooTimed{
		Time: fns.DateTimeNow(),
	}

	b, encodeErr := fns.JsonAPI().Marshal(foo)
	if encodeErr != nil {
		t.Error("encode failed", encodeErr)
		return
	}
	t.Log(string(b))
	bar := FooTimed{}
	decodeErr := fns.JsonAPI().Unmarshal(b, &bar)
	if decodeErr != nil {
		t.Error("decode failed", decodeErr)
		return
	}
	t.Log(bar)
}
