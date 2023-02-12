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

package shareds

import (
	"context"
	"github.com/aacfactory/errors"
	"sync"
	"time"
)

func NewLocalLockers() *LocalLockers {
	return &LocalLockers{
		mutex:   new(sync.Mutex),
		lockers: make(map[string]sync.Locker),
	}
}

type LocalLockers struct {
	mutex   *sync.Mutex
	lockers map[string]sync.Locker
}

func (lockers *LocalLockers) Get(ctx context.Context, key []byte, timeout time.Duration) (locker sync.Locker, err errors.CodeError) {
	lockers.mutex.Lock()
	defer lockers.mutex.Unlock()
	mKey := string(key)
	has := false
	locker, has = lockers.lockers[mKey]
	if has {
		locker = &sync.Mutex{}
		lockers.lockers[mKey] = locker
	}
	return
}
