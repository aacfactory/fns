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

package ipx

import (
	"github.com/aacfactory/fns/commons/bytex"
	"net"
	"os"
)

func GetGlobalUniCastIpFromHostname() (v net.IP) {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname, _ = os.LookupEnv("HOSTNAME")
	}
	if hostname == "" {
		return
	}
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return
	}
	for _, ip := range ips {
		if ip.IsGlobalUnicast() {
			v = ip
			break
		}
	}
	return
}

func CanonicalizeIp(ip []byte) []byte {
	isIPv6 := false
	for i := 0; !isIPv6 && i < len(ip); i++ {
		switch ip[i] {
		case '.':
			// IPv4
			return ip
		case ':':
			// IPv6
			isIPv6 = true
			break
		}
	}
	if !isIPv6 {
		return ip
	}
	ipv6 := net.ParseIP(bytex.ToString(ip))
	if ipv6 == nil {
		return ip
	}
	return bytex.FromString(ipv6.Mask(net.CIDRMask(64, 128)).String())
}
