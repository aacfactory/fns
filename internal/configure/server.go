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

package configure

import (
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/ssl"
	"github.com/aacfactory/json"
	"strings"
)

type Cors struct {
	AllowedOrigins   []string `json:"allowedOrigins"`
	AllowedHeaders   []string `json:"allowedHeaders"`
	ExposedHeaders   []string `json:"exposedHeaders"`
	AllowCredentials bool     `json:"allowCredentials"`
	MaxAge           int      `json:"maxAge"`
}

type TLS struct {
	// Kind
	// ACME
	// SSC(SELF-SIGN-CERT)
	// DEFAULT
	Kind    string          `json:"kind"`
	Options json.RawMessage `json:"options"`
}

func (config *TLS) Config() (serverTLS *tls.Config, clientTLS *tls.Config, err error) {
	kind := strings.TrimSpace(config.Kind)
	loader, hasLoader := ssl.GetLoader(kind)
	if !hasLoader {
		err = errors.Warning(fmt.Sprintf("fns: can not get %s tls loader", kind))
		return
	}
	loaderConfig, loaderConfigErr := configures.NewJsonConfig(config.Options)
	if loaderConfigErr != nil {
		err = errors.Warning(fmt.Sprintf("fns: can not get options of %s tls loader", kind)).WithCause(loaderConfigErr)
		return
	}
	serverTLS, clientTLS, err = loader(loaderConfig)
	return
}

type Server struct {
	Port         int                        `json:"port"`
	Cors         *Cors                      `json:"cors"`
	TLS          *TLS                       `json:"tls"`
	Options      json.RawMessage            `json:"options"`
	Interceptors map[string]json.RawMessage `json:"interceptors"`
}
