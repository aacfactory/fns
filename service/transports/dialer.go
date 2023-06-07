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

package transports

import (
	"github.com/aacfactory/errors"
	"golang.org/x/sync/singleflight"
	"sync"
)

type Dialer interface {
	Dial(address string) (client Client, err error)
}

func NewDialer(opts *FastHttpClientOptions) (dialer Dialer, err error) {
	dialer = &fastDialer{
		opts:    opts,
		group:   &singleflight.Group{},
		clients: sync.Map{},
	}
	return
}

type fastDialer struct {
	opts    *FastHttpClientOptions
	group   *singleflight.Group
	clients sync.Map
}

func (dialer *fastDialer) Dial(address string) (client Client, err error) {
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
	client = cc.(Client)
	return
}

func (dialer *fastDialer) createClient(address string) (client Client, err error) {
	client, err = NewFastClient(address, dialer.opts)
	if err != nil {
		return
	}
	return
}

func (dialer *fastDialer) Close() {
	dialer.clients.Range(func(key, value any) bool {
		client, ok := value.(Client)
		if ok {
			client.Close()
		}
		return true
	})
}
