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

package ssl

import (
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configuares"
)

type Loader func(options configuares.Config) (serverTLS *tls.Config, clientTLS *tls.Config, err error)

var (
	loaders = map[string]Loader{"SSC": SSCLoader, "DEFAULT": DefaultLoader}
)

func RegisterLoader(kind string, loader Loader) {
	if kind == "" || loader == nil {
		return
	}
	_, has := loaders[kind]
	if has {
		panic(fmt.Errorf("fns: regisger tls loader failed for existed"))
	}
	loaders[kind] = loader
}

func GetLoader(kind string) (loader Loader, has bool) {
	loader, has = loaders[kind]
	return
}
