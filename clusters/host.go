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

package clusters

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/ipx"
	"os"
)

type HostRetriever func() (host string, err error)

func defaultHostRetriever() (host string, err error) {
	ip := ipx.GetGlobalUniCastIpFromHostname()
	if ip == nil {
		err = errors.Warning("fns: get host from hostname failed")
		return
	}
	host = ip.String()
	return
}

func environmentHostRetriever() (host string, err error) {
	v, has := os.LookupEnv("FNS-HOST")
	if has {
		host = v
		return
	}
	err = errors.Warning("fns: get host from FNS-HOST env failed")
	return
}

var (
	hostRetrievers = map[string]HostRetriever{
		"default": defaultHostRetriever,
		"env":     environmentHostRetriever,
	}
)

func RegisterHostRetriever(name string, fn HostRetriever) {
	hostRetrievers[name] = fn
}

func getHostRetriever(name string) (fn HostRetriever, has bool) {
	fn, has = hostRetrievers[name]
	return
}
