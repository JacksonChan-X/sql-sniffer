package helper

import (
	"fmt"
	"net"
)

// GetAllInterfaces 获取本机所有网卡信息
func GetAllInterfaces() ([]string, error) {
	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("获取网络接口失败: %v", err)
	}

	var result []string
	for _, iface := range interfaces {
		// 过滤掉 loopback 接口和状态为 down 的接口
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			result = append(result, iface.Name)
		}
	}

	return result, nil
}

func OutboundIP() string {
	conn, err := net.Dial("udp", "123.123.123.123:1")
	if err != nil {
		return ""
	}
	defer conn.Close()
	ip, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	return ip
}
