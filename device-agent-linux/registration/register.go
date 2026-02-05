package registration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const (
	DeviceInfoFile = "/var/lib/device-agent-linux/device.json"
)

type DeviceInfo struct {
	DeviceID    string `json:"device_id"`
	Status      string `json:"status"`
	RecoveryKey string `json:"recovery_key"`
	MacID       string `json:"mac_id"`
	Location    string `json:"location"`
	OSDetails   string `json:"os_details"`
	BIOSPass    string `json:"bios_pass,omitempty"`
}

type RegisterRequest struct {
	DeviceID  string `json:"device_id,omitempty"`
	MacID     string `json:"mac_id"`
	Location  string `json:"location"`
	OSDetails string `json:"os_details"`
	BIOSPass  string `json:"bios_pass,omitempty"`
}

type RegisterResponse struct {
	DeviceID    string `json:"device_id"`
	Status      string `json:"status"`
	RecoveryKey string `json:"recovery_key"`
}

// GetMacAddress retrieves the MAC address of the primary network interface
func GetMacAddress() string {
	cmd := exec.Command("cat", "/sys/class/net/$(ip route show default | awk '/default/ {print $5}')/address")
	cmd.Env = append(os.Environ(), "SHELL=/bin/bash")
	out, err := exec.Command("bash", "-c", "cat /sys/class/net/$(ip route show default | awk '/default/ {print $5}')/address").Output()
	if err != nil {
		log.Printf("[REGISTRATION] Failed to get MAC address: %v", err)
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// GetOSDetails retrieves OS information
func GetOSDetails() string {
	out, err := exec.Command("uname", "-a").Output()
	if err != nil {
		return "Linux"
	}
	return strings.TrimSpace(string(out))
}

// LoadDeviceInfo loads device info from local storage
func LoadDeviceInfo() (*DeviceInfo, error) {
	data, err := os.ReadFile(DeviceInfoFile)
	if err != nil {
		return nil, err
	}

	var info DeviceInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// SaveDeviceInfo saves device info to local storage
func SaveDeviceInfo(info *DeviceInfo) error {
	// Ensure directory exists
	dir := "/var/lib/device-agent-linux"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(DeviceInfoFile, data, 0600)
}

// RegisterDevice registers the device with the backend
func RegisterDevice(ip, port, biosPassword string) (*DeviceInfo, error) {
	// Check if already registered
	if info, err := LoadDeviceInfo(); err == nil {
		log.Printf("[REGISTRATION] Device already registered: %s", info.DeviceID)
		return info, nil
	}

	log.Println("[REGISTRATION] Registering device with backend...")

	// Gather device information
	macID := GetMacAddress()
	osDetails := GetOSDetails()
	hostname, _ := os.Hostname()

	reqBody := RegisterRequest{
		MacID:     macID,
		Location:  hostname,
		OSDetails: osDetails,
		BIOSPass:  biosPassword,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Send registration request
	resp, err := http.Post(
		fmt.Sprintf("http://%s:%s/api/register", ip, port),
		"application/json",
		bytes.NewBuffer(body),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to register with backend: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var regResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Save device info locally
	deviceInfo := &DeviceInfo{
		DeviceID:    regResp.DeviceID,
		Status:      regResp.Status,
		RecoveryKey: regResp.RecoveryKey,
		MacID:       macID,
		Location:    hostname,
		OSDetails:   osDetails,
		BIOSPass:    biosPassword,
	}

	if err := SaveDeviceInfo(deviceInfo); err != nil {
		return nil, fmt.Errorf("failed to save device info: %v", err)
	}

	log.Printf("[REGISTRATION] Device registered successfully: %s", deviceInfo.DeviceID)
	log.Printf("[REGISTRATION] Recovery Key: %s", deviceInfo.RecoveryKey)

	return deviceInfo, nil
}

// GetDeviceID returns the device ID (loads from file or returns hostname)
func GetDeviceID() string {
	if info, err := LoadDeviceInfo(); err == nil {
		return info.DeviceID
	}

	// Fallback to hostname
	hostname, _ := os.Hostname()
	return hostname
}
