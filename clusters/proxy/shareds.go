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

package proxy

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
)

var (
	sharedHandlerPath = append(handlerPathPrefix, []byte("/clusters/shared")...)
	sharedHeader      = []byte("X-Fns-Shared")
)

func NewShared(client transports.Client, signature signatures.Signature) shareds.Shared {
	return &Shared{
		lockers: &Lockers{
			client:    client,
			signature: signature,
		},
		store: &Store{
			client:    client,
			signature: signature,
		},
	}
}

type Shared struct {
	lockers shareds.Lockers
	store   shareds.Store
}

func (shared *Shared) Construct(_ shareds.Options) (err error) {
	return
}

func (shared *Shared) Lockers() (lockers shareds.Lockers) {
	lockers = shared.lockers
	return
}

func (shared *Shared) Store() (store shareds.Store) {
	store = shared.store
	return
}

func (shared *Shared) Close() {}

// +-------------------------------------------------------------------------------------------------------------------+

func NewSharedHandler(shared shareds.Shared) transports.Handler {
	return &SharedHandler{
		lockers: NewSharedLockersHandler(shared.Lockers()),
		store:   NewSharedStoreHandler(shared.Store()),
	}
}

type SharedHandler struct {
	lockers transports.Handler
	store   transports.Handler
}

func (handler *SharedHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	kind := r.Header().Get(sharedHeader)
	if bytes.Equal(kind, sharedHeaderLockersValue) {
		handler.lockers.Handle(w, r)
	} else if bytes.Equal(kind, sharedHeaderStoreValue) {
		handler.store.Handle(w, r)
	} else {
		w.Failed(errors.Warning("fns: X-Fns-Shared is required"))
	}
	return
}
