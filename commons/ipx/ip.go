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
	"net"
	"os"
)

func GetGlobalUniCastIpFromHostname() (ipv4 string) {
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
			ipv4 = ip.To4().String()
			break
		}
	}
	return
}