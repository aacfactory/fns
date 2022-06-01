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

package fns

import (
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	B = 1 << (10 * iota)
	KB
	MB
	GB
	TB
	PB
	EB

	activeSystemEnvKey = "FNS-ACTIVE"
)

func defaultConfigRetrieverOption() (option configuares.RetrieverOption) {
	path, pathErr := filepath.Abs("./config")
	if pathErr != nil {
		panic(fmt.Sprintf("fns create default config retriever failed, cant not get './config'"))
		return
	}
	active, _ := os.LookupEnv(activeSystemEnvKey)
	active = strings.TrimSpace(active)
	store := configuares.NewFileStore(path, "fns", '-')
	option = configuares.RetrieverOption{
		Active: active,
		Format: "YAML",
		Store:  store,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type Config struct {
	Port          int              `json:"port"`
	TLS           *TLSConfig       `json:"tls"`
	Websocket     WebsocketConfig  `json:"websocket"`
	Cors          *HttpCorsConfig  `json:"cors"`
	Client        HttpClientConfig `json:"client"`
	ServerOptions json.RawMessage  `json:"serverOptions"`
	Log           LogConfig        `json:"log"`
	Cluster       *ClusterConfig   `json:"cluster"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type ClusterConfig struct {
	Kind    string          `json:"kind"`
	Options json.RawMessage `json:"options"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type HttpClientConfig struct {
	MaxIdleConnSeconds  int `json:"maxIdleConnSeconds"`
	MaxConnsPerHost     int `json:"maxConnsPerHost"`
	MaxIdleConnsPerHost int `json:"maxIdleConnsPerHost"`
}

type HttpCorsConfig struct {
	AllowedOrigins   []string `json:"allowedOrigins,omitempty"`
	AllowedHeaders   []string `json:"allowedHeaders,omitempty"`
	ExposedHeaders   []string `json:"exposedHeaders,omitempty"`
	AllowCredentials bool     `json:"allowCredentials,omitempty"`
	MaxAge           int      `json:"maxAge,omitempty"`
}

type WebsocketConfig struct {
	HandshakeTimeoutSeconds int    `json:"handshakeTimeoutSeconds"`
	ReadBufferSize          string `json:"readBufferSize"`
	WriteBufferSize         string `json:"writeBufferSize"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type TLSConfig struct {
	// Kind
	// ACME
	// SSC(SELF-SIGN-CERT)
	// DEFAULT
	Kind    string          `json:"kind"`
	Options json.RawMessage `json:"options"`
}

func (config *TLSConfig) Load(options configuares.Config) (serverTLS *tls.Config, clientTLS *tls.Config, err error) {
	config.Kind = strings.ToUpper(strings.TrimSpace(config.Kind))
	switch config.Kind {
	case "ACME":
		loader, has := tlsLoaders["ACME"]
		if !has {
			err = fmt.Errorf("fns: please register ACME tls loader first")
			return
		}
		serverTLS, clientTLS, err = loader(options)
	case "SSC", "SELF-SIGN-CERT":
		loader, _ := tlsLoaders["SSC"]
		serverTLS, clientTLS, err = loader(options)
	default:
		loader, _ := tlsLoaders["DEFAULT"]
		serverTLS, clientTLS, err = loader(options)
	}
	return
}
