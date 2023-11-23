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

package standard

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
)

const (
	transportName = "standard"
)

type Config struct {
	MaxRequestHeaderSize string        `json:"maxRequestHeaderSize"`
	MaxRequestBodySize   string        `json:"maxRequestBodySize"`
	ReadTimeout          string        `json:"readTimeout"`
	ReadHeaderTimeout    string        `json:"readHeaderTimeout"`
	WriteTimeout         string        `json:"writeTimeout"`
	IdleTimeout          string        `json:"idleTimeout"`
	Client               *ClientConfig `json:"client"`
}

func (config *Config) ClientConfig() *ClientConfig {
	if config.Client == nil {
		return &ClientConfig{}
	}
	return config.Client
}

func New() transports.Transport {
	return &Transport{}
}

type Transport struct {
	server *Server
	dialer *Dialer
}

func (tr *Transport) Name() (name string) {
	name = transportName
	return
}

func (tr *Transport) Construct(options transports.Options) (err error) {
	// log
	log := options.Log.With("transport", transportName)
	// tls
	tlsConfig, tlsConfigErr := options.Config.GetTLS()
	if tlsConfigErr != nil {
		err = errors.Warning("fns: standard transport construct failed").WithCause(tlsConfigErr).WithMeta("transport", transportName)
		return
	}
	// handler
	if options.Handler == nil {
		err = errors.Warning("fns: standard transport construct failed").WithCause(fmt.Errorf("handler is nil")).WithMeta("transport", transportName)
		return
	}

	// port
	port, portErr := options.Config.GetPort()
	if portErr != nil {
		err = errors.Warning("fns: standard transport construct failed").WithCause(portErr).WithMeta("transport", transportName)
		return
	}
	// config
	optConfig, optConfigErr := options.Config.OptionsConfig()
	if optConfigErr != nil {
		err = errors.Warning("fns: standard transport construct failed").WithCause(optConfigErr).WithMeta("transport", transportName)
		return
	}
	config := &Config{}
	configErr := optConfig.As(config)
	if configErr != nil {
		err = errors.Warning("fns: standard transport construct failed").WithCause(configErr).WithMeta("transport", transportName)
		return
	}
	// server
	srv, srvErr := newServer(log, port, tlsConfig, config, options.Handler)
	if srvErr != nil {
		err = errors.Warning("fns: standard transport construct failed").WithCause(srvErr).WithMeta("transport", transportName)
		return
	}
	tr.server = srv

	// dialer
	clientConfig := config.ClientConfig()
	if tlsConfig != nil {
		cliTLS, dialer := tlsConfig.Client()
		clientConfig.TLSConfig = cliTLS
		if dialer != nil {
			clientConfig.TLSDialer = dialer
		}
	}
	dialer, dialerErr := NewDialer(clientConfig)
	if dialerErr != nil {
		err = errors.Warning("http: standard transport construct failed").WithCause(dialerErr)
		return
	}
	tr.dialer = dialer
	return
}

func (tr *Transport) Dial(address []byte) (client transports.Client, err error) {
	client, err = tr.dialer.Dial(address)
	return
}

func (tr *Transport) Port() (port int) {
	port = tr.server.port
	return
}

func (tr *Transport) ListenAndServe() (err error) {
	err = tr.server.ListenAndServe()
	return
}

func (tr *Transport) Shutdown(ctx context.Context) {
	tr.dialer.Close()
	_ = tr.server.Shutdown(ctx)
	return
}
