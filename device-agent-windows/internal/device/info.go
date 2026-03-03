package device

import "runtime"

type DeviceInfo struct {
	MacID     string
	OS        string
	Arch      string
	Latitude  float64
	Longitude float64
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
