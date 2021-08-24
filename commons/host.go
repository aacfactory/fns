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

package commons

import (
	"fmt"
	"net"
	"os"
)

// IpFromHostname
// 通过host获取ip列表中的第一个
func IpFromHostname(enableIpV6 bool) (ip string, err error) {
	hostname, has := os.LookupEnv("HOSTNAME")
	if !has {
		hostname, _ = os.Hostname()
	}
	if hostname == "" {
		err = fmt.Errorf("fns get ip from hostname failed")
		return
	}
	ips, ipErr := net.LookupIP(hostname)
	if ipErr != nil {
		err = fmt.Errorf("fns get ip from hostname %s failed, %v", hostname, ipErr)
		return
	}
	if ips == nil || len(ips) == 0 {
		err = fmt.Errorf("fns get ip from hostname %s failed, no ip found", hostname)
		return
	}
	for _, hostIp := range ips {
		if hostIp.IsLoopback() {
			continue
		}
		if hostIp.IsMulticast() {
			continue
		}
		if enableIpV6 {
			ipv6 := hostIp.To16()
			if ipv6 != nil {
				ip = ipv6.String()
				return
			}
		}
		ipv4 := hostIp.To4()
		if ipv4 != nil {
			ip = ipv4.String()
			return
		}
	}
	if ip == "" {
		err = fmt.Errorf("fns get ip from hostname %s failed, no ip found", hostname)
		return
	}
	return
}
