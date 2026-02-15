package helper

import (
	"bytes"
	"device-agent-windows/internal/config"
	"device-agent-windows/internal/model"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type IPInfoResponse struct {
	Loc string `json:"loc"`
}

func getDeviceID() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if (iface.Flags&net.FlagLoopback) != 0 || (iface.Flags&net.FlagUp) == 0 {
			continue
		}

		// Get MAC address
		if len(iface.HardwareAddr) > 0 {
			return iface.HardwareAddr.String(), nil
		}
	}

	return "unknown", nil
}

func getOSDetails() (string, error) {
	cmd := exec.Command("cmd", "/C", "ver")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func GetLocation() (model.GeoResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://ipinfo.io/json")
	if err != nil {
		return model.GeoResponse{}, err
	}
	defer resp.Body.Close()

	var data IPInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return model.GeoResponse{}, err
	}

	parts := strings.Split(data.Loc, ",")
	if len(parts) != 2 {
		return model.GeoResponse{}, fmt.Errorf("invalid loc format")
	}

	var lat, lon float64
	fmt.Sscanf(parts[0], "%f", &lat)
	fmt.Sscanf(parts[1], "%f", &lon)

	return model.GeoResponse{
		Lat: lat,
		Lon: lon,
	}, nil
}

func ConstructBackendURL(cfg *config.Config) string {
	return "http://" + cfg.BackendConfig.BackendIp + ":" + cfg.BackendConfig.BackendPort
}

func SetDeviceDetails(cfg *config.Config) {
	deviceID, err := getDeviceID()
	if err != nil {
		cfg.DeviceDetails.MacID = "unknown"
		if cfg.DeviceDetails.DeviceID == "" {
			cfg.DeviceDetails.DeviceID = "unknown"
		}
	} else {
		cfg.DeviceDetails.MacID = deviceID
		if cfg.DeviceDetails.DeviceID == "" {
			cfg.DeviceDetails.DeviceID = deviceID
		}
	}

	osDetails, err := getOSDetails()
	if err != nil {
		if cfg.DeviceDetails.OSDetails == "" {
			cfg.DeviceDetails.OSDetails = "unknown"
		}
	} else {
		cfg.DeviceDetails.OSDetails = osDetails
	}
}

func RunCommand(cmd string, args ...string) (string, error) {
	c := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	if err != nil {
		return "", err
	}
	return stdout.String(), nil
}
