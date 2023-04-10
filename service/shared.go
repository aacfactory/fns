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
	"github.com/aacfactory/fns/service/shareds"
	"github.com/aacfactory/systems/memory"
)

type Shared interface {
	Lockers() (lockers shareds.Lockers)
	Store() (store shareds.Store)
	Caches() (cache shareds.Caches)
}

func newLocalShared() (Shared, error) {
	maxCacheSize := uint64(0)
	mem, _ := memory.Stats()
	if mem != nil {
		maxCacheSize = mem.Available / 4
	}
	return &localShared{
		lockers: shareds.LocalLockers(),
		store:   shareds.LocalStore(),
		cache:   shareds.LocalCaches(maxCacheSize),
	}, nil
}

type localShared struct {
	lockers shareds.Lockers
	store   shareds.Store
	cache   shareds.Caches
}

func (s localShared) Lockers() (lockers shareds.Lockers) {
	return s.lockers
}

func (s localShared) Store() (store shareds.Store) {
	return s.store
}

func (s localShared) Caches() (cache shareds.Caches) {
	return s.cache
}
