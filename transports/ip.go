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

package transports

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/ipx"
	"github.com/aacfactory/fns/context"
	"net"
)

func DeviceIp(ctx context.Context) (ip []byte) {
	r := LoadRequest(ctx)
	if r == nil {
		return
	}
	ip = r.Header().Get(DeviceIpHeaderName)
	if len(ip) > 0 {
		return
	}
	if trueClientIp := r.Header().Get(TrueClientIpHeaderName); len(trueClientIp) > 0 {
		ip = trueClientIp
	} else if realIp := r.Header().Get(XRealIpHeaderName); len(realIp) > 0 {
		ip = realIp
	} else if forwarded := r.Header().Get(XForwardedForHeaderName); len(forwarded) > 0 {
		i := bytes.Index(forwarded, []byte{',', ' '})
		if i == -1 {
			i = len(forwarded)
		}
		ip = forwarded[:i]
	} else {
		remoteIp, _, remoteIpErr := net.SplitHostPort(bytex.ToString(r.RemoteAddr()))
		if remoteIpErr != nil {
			remoteIp = bytex.ToString(r.RemoteAddr())
		}
		ip = bytex.FromString(remoteIp)
	}
	ip = ipx.CanonicalizeIp(ip)
	r.Header().Set(DeviceIpHeaderName, ip)
	return
}
