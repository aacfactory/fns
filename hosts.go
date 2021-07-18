package fns

import (
	"fmt"
	"net"
	"os"
)

//IpFromHostname 通过host获取ip列表中的第一个
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
