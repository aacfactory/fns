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

package services

import (
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services/documents"
	"sort"
	"strings"
	"unsafe"
)

type Endpoint interface {
	Name() (name string)
	Internal() (ok bool)
	Document() (document documents.Endpoint)
	Functions() (functions Fns)
	Shutdown(ctx context.Context)
}

type EndpointInfo struct {
	Id        string             `json:"id"`
	Version   versions.Version   `json:"version"`
	Address   string             `json:"address"`
	Name      string             `json:"name"`
	Internal  bool               `json:"internal"`
	Functions FnInfos            `json:"functions"`
	Document  documents.Endpoint `json:"document"`
}

type EndpointInfos []EndpointInfo

func (infos EndpointInfos) Len() int {
	return len(infos)
}

func (infos EndpointInfos) Less(i, j int) bool {
	x := infos[i]
	y := infos[j]
	n := strings.Compare(x.Name, y.Name)
	if n < 0 {
		return true
	} else if n == 0 {
		return x.Version.LessThan(y.Version)
	} else {
		return false
	}
}

func (infos EndpointInfos) Swap(i, j int) {
	infos[i], infos[j] = infos[j], infos[i]
}

func (infos EndpointInfos) Find(name []byte) (info EndpointInfo, found bool) {
	ns := unsafe.String(unsafe.SliceData(name), len(name))
	n := infos.Len()
	if n < 65 {
		for _, endpoint := range infos {
			if endpoint.Name == ns {
				info = endpoint
				found = true
				break
			}
		}
		return
	}
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)
		if strings.Compare(infos[h].Name, ns) < 0 {
			i = h + 1
		} else {
			j = h
		}
	}
	found = i < n && infos[i].Name == ns
	if found {
		info = infos[i]
	}
	return
}

type EndpointGetOption func(options *EndpointGetOptions)

type EndpointGetOptions struct {
	id              []byte
	requestVersions versions.Intervals
}

func (options EndpointGetOptions) Id() []byte {
	return options.id
}

func (options EndpointGetOptions) Versions() versions.Intervals {
	return options.requestVersions
}

func EndpointId(id []byte) EndpointGetOption {
	return func(options *EndpointGetOptions) {
		options.id = id
		return
	}
}

func EndpointVersions(requestVersions versions.Intervals) EndpointGetOption {
	return func(options *EndpointGetOptions) {
		options.requestVersions = requestVersions
		return
	}
}

type Endpoints interface {
	Info() (infos EndpointInfos)
	Get(ctx context.Context, name []byte, options ...EndpointGetOption) (endpoint Endpoint, has bool)
	Request(ctx context.Context, name []byte, fn []byte, param interface{}, options ...RequestOption) (response Response, err error)
}

type Services []Service

func (s Services) Len() int {
	return len(s)
}

func (s Services) Less(i, j int) bool {
	return strings.Compare(s[i].Name(), s[j].Name()) < 0
}

func (s Services) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Services) Add(v Service) Services {
	ss := append(s, v)
	sort.Sort(ss)
	return ss
}

func (s Services) Find(name []byte) (v Service, found bool) {
	ns := unsafe.String(unsafe.SliceData(name), len(name))
	n := s.Len()
	if n < 65 {
		for _, endpoint := range s {
			if endpoint.Name() == ns {
				v = endpoint
				found = true
				break
			}
		}
		return
	}
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)
		if strings.Compare(s[h].Name(), ns) < 0 {
			i = h + 1
		} else {
			j = h
		}
	}
	found = i < n && s[i].Name() == ns
	if found {
		v = s[i]
	}
	return
}
