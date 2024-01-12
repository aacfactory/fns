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

package clusters_test

import (
	"github.com/aacfactory/avro"
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/commons/avros"
	"github.com/aacfactory/fns/services"
	"testing"
	"time"
)

func TestRequestBody(t *testing.T) {
	rb := clusters.RequestBody{
		ContextUserValues: nil,
		Params:            avro.MustMarshal("123"),
	}
	p, encodeErr := avro.Marshal(rb)
	if encodeErr != nil {
		t.Error(encodeErr)
		return
	}
	v := clusters.RequestBody{}
	decodeErr := avro.Unmarshal(p, &v)
	if decodeErr != nil {
		t.Error(decodeErr)
		return
	}
	param := avros.RawMessage(rb.Params)
	s := ""
	err := param.Unmarshal(&s)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(s)
}

func TestResponseBody(t *testing.T) {
	type Data struct {
		Id  string
		Now time.Time
	}
	p, _ := avro.Marshal(Data{
		Id:  "id",
		Now: time.Now(),
	})

	rsb := clusters.ResponseBody{
		Succeed:     true,
		Data:        p,
		Attachments: nil,
	}
	rsb.Attachments = append(rsb.Attachments, clusters.Entry{
		Key:   []byte("key"),
		Value: []byte("value"),
	})

	rp, _ := avro.Marshal(rsb)
	t.Log(len(rp))
	rsb = clusters.ResponseBody{}
	decodeErr := avro.Unmarshal(rp, &rsb)
	if decodeErr != nil {
		t.Error(decodeErr)
		return
	}
	data := Data{}
	decodeErr = avro.Unmarshal(rsb.Data, &data)
	if decodeErr != nil {
		t.Error(decodeErr)
		return
	}
	t.Log(data)
	t.Log(string(rsb.Attachments[0].Key), string(rsb.Attachments[0].Value))
	d2, d2Err := services.ValueOfResponse[Data](avros.RawMessage(rsb.Data))
	if d2Err != nil {
		t.Error(d2Err)
		return
	}
	t.Log(d2)
}
