package device

import "runtime"

type DeviceInfo struct {
	MacID     string  `json:"mac_id"`
	OS        string  `json:"os"`
	Arch      string  `json:"arch"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func CollectDeviceInfo() DeviceInfo {
	loc := GetLocation()
	return DeviceInfo{
		MacID:     GetMacID(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
	}
}
