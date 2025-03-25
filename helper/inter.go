package helper

import (
	"net"
	"strings"
)

var ips = make([]string, 0)

func GetLocalIpByInterface(inter string) error {
	ip4s, ip6s, err := GetIPByInterface(inter)
	if err != nil {
		return err
	}
	ips = append(ips, ip4s...)
	ips = append(ips, ip6s...)
	return nil
}

func IsLocalIp(ip string) bool {
	for _, v := range ips {
		if v == ip {
			return true
		}
	}
	return false
}

// GetIPByInterface 根据网卡获取该网卡下的ipv4或ipv6地址
func GetIPByInterface(name string) (ip4s []string, ip6s []string, err error) {
	ip4s = append(ip4s, "127.0.0.1")
	netInterface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, nil, err
	}
	address, err := netInterface.Addrs()
	if err != nil {
		return nil, nil, err
	}

	for _, a := range address {
		if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			ip4s = append(ip4s, ipNet.IP.String())
		}

		if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To16() != nil {
			if strings.Contains(ipNet.IP.String(), ":") {
				ip6s = append(ip6s, ipNet.IP.String())
			}
		}
	}
	return ip4s, ip6s, nil
}
