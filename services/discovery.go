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

package services

import (
	"context"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/services/documents"
)

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

type EndpointInfo struct {
	Id       []byte
	Name     []byte
	Version  versions.Version
	Internal bool
	Document *documents.Document
}

type Discovery interface {
	Endpoints(ctx context.Context) (infos []EndpointInfo)
	Get(ctx context.Context, name []byte, options ...EndpointGetOption) (endpoint Endpoint, has bool)
}
