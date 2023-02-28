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

package service

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/rings"
	"strings"
	"sync"
)

type EndpointDiscoveryGetOption func(options *EndpointDiscoveryGetOptions)

type EndpointDiscoveryGetOptions struct {
	id           string
	native       bool
	versionRange []versions.Version
}

func Exact(id string) EndpointDiscoveryGetOption {
	return func(options *EndpointDiscoveryGetOptions) {
		options.id = strings.TrimSpace(id)
		return
	}
}

func Native() EndpointDiscoveryGetOption {
	return func(options *EndpointDiscoveryGetOptions) {
		options.native = true
		return
	}
}

func VersionRange(left versions.Version, right versions.Version) EndpointDiscoveryGetOption {
	return func(options *EndpointDiscoveryGetOptions) {
		options.versionRange = append(options.versionRange, left, right)
		return
	}
}

type EndpointDiscovery interface {
	Get(ctx context.Context, service string, options ...EndpointDiscoveryGetOption) (endpoint Endpoint, has bool)
}

type Registration struct {
	hostId  string
	id      string
	version versions.Version
	address string
	name    string
	Client  HttpClient
}

func (registration *Registration) Key() (key string) {
	key = registration.id
	return
}

func (registration *Registration) Name() (name string) {
	name = registration.name
	return
}

func (registration *Registration) Internal() (ok bool) {
	ok = true
	return
}

func (registration *Registration) Document() (document Document) {
	return
}

func (registration *Registration) Request(ctx context.Context, r Request) (result Result) {
	//TODO implement me
	panic("implement me")
	// todo device id = hostId, httpRequestInternalHeader and sign
}

func (registration *Registration) RequestSync(ctx context.Context, r Request) (result interface{}, has bool, err errors.CodeError) {
	//TODO implement me
	panic("implement me")
}

type Registrations struct {
	id     string
	locker sync.Mutex
	values sync.Map
}

func (r *Registrations) Add(registration *Registration) {
	var ring *rings.Ring[*Registration]
	v, loaded := r.values.Load(registration.name)
	if !loaded {
		v = rings.New[*Registration](registration.name)
		r.values.Store(registration.name, v)
	}
	ring, _ = v.(*rings.Ring[*Registration])
	r.locker.Lock()
	registration.hostId = r.id
	ring.Push(registration)
	r.locker.Unlock()
	return
}

func (r *Registrations) Remove(id string, name string) {
	v, loaded := r.values.Load(name)
	if !loaded {
		return
	}
	ring, _ := v.(*rings.Ring[*Registration])
	entry, has := ring.Get(id)
	if has {
		entry.Client.Close()
		r.locker.Lock()
		ring.Remove(id)
		r.locker.Unlock()
	}
	return
}

func (r *Registrations) Get(id string, name string) (registration *Registration, has bool) {
	v, loaded := r.values.Load(name)
	if !loaded {
		return
	}
	ring, _ := v.(*rings.Ring[*Registration])
	if id == "" {
		registration = ring.Next()
		has = true
	} else {
		registration, has = ring.Get(id)
	}
	return
}

func (r *Registrations) Close() {
	r.locker.Lock()
	r.values.Range(func(key, value any) bool {
		entries := value.(*rings.Ring[*Registration])
		size := entries.Len()
		for i := 0; i < size; i++ {
			entry, ok := entries.Pop()
			if ok {
				entry.Client.Close()
			}
		}
		return true
	})
	r.locker.Unlock()
	return
}
