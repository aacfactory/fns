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
	"strings"
)

type EndpointGetOption func(options *EndpointGetOptions)

type EndpointGetOptions struct {
	id              string
	requestVersions versions.Interval
}

func EndpointId(id string) EndpointGetOption {
	return func(options *EndpointGetOptions) {
		options.id = strings.TrimSpace(id)
		return
	}
}

func EndpointVersions(requestVersions versions.Interval) EndpointGetOption {
	return func(options *EndpointGetOptions) {
		options.requestVersions = requestVersions
		return
	}
}

type Discovery interface {
	Get(ctx context.Context, service string, options ...EndpointGetOption) (endpoint Endpoint, has bool)
}
