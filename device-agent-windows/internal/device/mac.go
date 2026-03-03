package device

import (
	"net"
)

func GetMacID() string {
	interfaces, _ := net.Interfaces()
	for _, i := range interfaces {
		if i.Flags&net.FlagUp != 0 && len(i.HardwareAddr) > 0 {
			return i.HardwareAddr.String()
		}
	}
	return "unknown"
}
