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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"golang.org/x/sync/singleflight"
	"sync"
)

func NewDialer(opts *ClientConfig) (dialer *Dialer, err error) {
	dialer = &Dialer{
		config:  opts,
		group:   &singleflight.Group{},
		clients: sync.Map{},
	}
	return
}

type Dialer struct {
	config  *ClientConfig
	group   *singleflight.Group
	clients sync.Map
}

func (dialer *Dialer) Dial(addressBytes []byte) (client transports.Client, err error) {
	address := bytex.ToString(addressBytes)
	cc, doErr, _ := dialer.group.Do(address, func() (clients interface{}, err error) {
		hosted, has := dialer.clients.Load(address)
		if has {
			clients = hosted
			return
		}
		hosted, err = dialer.createClient(address)
		if err != nil {
			return
		}
		dialer.clients.Store(address, hosted)
		clients = hosted
		return
	})
	if doErr != nil {
		err = errors.Warning("http2: dial failed").WithMeta("address", address).WithCause(doErr)
		return
	}
	client = cc.(*Client)
	return
}

func (dialer *Dialer) createClient(address string) (client transports.Client, err error) {
	client, err = NewClient(address, dialer.config)
	if err != nil {
		return
	}
	return
}

func (dialer *Dialer) Close() {
	dialer.clients.Range(func(key, value any) bool {
		client, ok := value.(*Client)
		if ok {
			client.host.CloseIdleConnections()
		}
		return true
	})
}
