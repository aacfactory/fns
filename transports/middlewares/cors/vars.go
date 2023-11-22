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

package cors

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"net/http"
)

var (
	varyHeader                               = transports.VaryHeaderName
	accessControlRequestMethodHeader         = transports.AccessControlRequestMethodHeaderName
	originHeader                             = transports.OriginHeaderName
	accessControlRequestHeadersHeader        = transports.AccessControlRequestHeadersHeaderName
	accessControlRequestPrivateNetworkHeader = transports.AccessControlRequestPrivateNetworkHeaderName
	accessControlAllowOriginHeader           = transports.AccessControlAllowOriginHeaderName
	accessControlAllowMethodsHeader          = transports.AccessControlAllowMethodsHeaderName
	accessControlAllowHeadersHeader          = transports.AccessControlAllowHeadersHeaderName
	accessControlAllowCredentialsHeader      = transports.AccessControlAllowCredentialsHeaderName
	accessControlAllowPrivateNetworkHeader   = transports.AccessControlAllowPrivateNetworkHeaderName
	accessControlMaxAgeHeader                = transports.AccessControlMaxAgeHeaderName
	accessControlExposeHeadersHeader         = transports.AccessControlExposeHeadersHeaderName
)

var (
	methodOptions = bytex.FromString(http.MethodOptions)
	methodHead    = bytex.FromString(http.MethodHead)
	methodGet     = bytex.FromString(http.MethodGet)
	methodPost    = bytex.FromString(http.MethodPost)
)

var (
	all       = []byte{'*'}
	trueBytes = []byte{'t', 'r', 'u', 'e'}
	joinBytes = []byte{',', ' '}
)
