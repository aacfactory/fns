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

package ssl

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configures"
	"net"
)

type ListenerFunc func(inner net.Listener) (ln net.Listener)

type Dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

type Config interface {
	Construct(options configures.Config) (err error)
	Server() (srvTLS *tls.Config, ln ListenerFunc)
	Client() (cliTLS *tls.Config, dialer Dialer)
}

var (
	configs = map[string]Config{
		"DEFAULT": &DefaultConfig{},
		"SSC":     &SSCConfig{},
	}
)

func RegisterConfig(kind string, config Config) {
	if kind == "" || config == nil {
		return
	}
	_, has := configs[kind]
	if has {
		panic(fmt.Errorf("fns: regisger tls config failed for existed"))
	}
	configs[kind] = config
}

func GetConfig(kind string) (config Config, has bool) {
	config, has = configs[kind]
	return
}
