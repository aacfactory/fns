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

package runtime

import (
	"bytes"
	"github.com/aacfactory/avro"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"time"
)

var (
	healthPath = bytex.FromString("/health")
)

func CheckHealth(ctx context.Context, client transports.Client) (ok bool) {
	status, header, body, _ := client.Do(ctx, transports.MethodGet, healthPath, nil, nil)
	if status != 200 {
		return
	}
	health := Health{}
	if bytes.Equal(header.Get(transports.ContentTypeHeaderName), transports.ContentTypeAvroHeaderValue) {
		decodeErr := avro.Unmarshal(body, &health)
		if decodeErr != nil {
			return
		}
	} else {
		decodeErr := json.Unmarshal(body, &health)
		if decodeErr != nil {
			return
		}
	}
	ok = health.Running
	return
}

func HealthHandler() transports.MuxHandler {
	return &healthHandler{
		launch: time.Now(),
	}
}

type healthHandler struct {
	launch time.Time
}

func (handler *healthHandler) Name() string {
	return "health"
}

func (handler *healthHandler) Construct(_ transports.MuxHandlerOptions) error {
	return nil
}

func (handler *healthHandler) Match(_ context.Context, method []byte, path []byte, _ transports.Header) bool {
	ok := bytes.Equal(method, transports.MethodGet) && bytes.Equal(path, healthPath)
	return ok
}

func (handler *healthHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	rt := Load(r)
	running, serving := rt.Running()
	w.Succeed(Health{
		Id:      bytex.ToString(rt.AppId()),
		Name:    rt.AppName(),
		Version: rt.AppVersion().String(),
		Running: running,
		Serving: serving,
		Launch:  handler.launch,
		Now:     time.Now(),
	})
	return
}

type Health struct {
	Id      string    `json:"id" avro:"id"`
	Name    string    `json:"name" avro:"name"`
	Version string    `json:"version" avro:"version"`
	Running bool      `json:"running" avro:"running"`
	Serving bool      `json:"serving" avro:"serving"`
	Launch  time.Time `json:"launch" avro:"launch"`
	Now     time.Time `json:"now" avro:"now"`
}
