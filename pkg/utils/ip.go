package utils

import (
	"fmt"
	"net"
)

// GetOutboundIP 决定首选的出站 IP 地址。
func GetOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		addrs, netErr := net.InterfaceAddrs()
		if netErr != nil {
			return "", fmt.Errorf("dial public DNS failed and failed to get interface addrs: %v, %v", err, netErr)
		}
		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String(), nil
				}
			}
		}
		return "", fmt.Errorf("dial public DNS failed and no suitable non-loopback IPv4 found: %v", err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
