package heartbeat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RegisterRequest represents a device registration request
type RegisterRequest struct {
	DeviceID  string `json:"device_id,omitempty"`
	MacID     string `json:"mac_id"`
	Location  string `json:"location"`
	OSDetails string `json:"os_details"`
}

// RegisterResponse represents a device registration response
type RegisterResponse struct {
	DeviceID    string `json:"device_id"`
	Status      string `json:"status"`
	RecoveryKey string `json:"recovery_key"`
}

// HeartbeatRequest represents a heartbeat request
type HeartbeatRequest struct {
	DeviceID string `json:"device_id"`
}

// HeartbeatResponse represents a heartbeat response
type HeartbeatResponse struct {
	Action string `json:"action"`
}

// RecoveryKeyRequest represents a recovery key submission request
type RecoveryKeyRequest struct {
	DeviceID    string `json:"device_id"`
	RecoveryKey string `json:"recovery_key"`
}

// RecoveryKeyResponse represents a recovery key submission response
type RecoveryKeyResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// BackendClient handles all backend communication
type BackendClient struct {
	BackendURL  string
	DeviceID    string
	RecoveryKey string
}

// GetBackendURL retrieves backend URL from environment or uses default
func GetBackendURL() string {
	host := os.Getenv("BACKEND_HOST")
	if host == "" {
		host = "http://localhost:8080"
	}
	// Ensure it has http:// prefix
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	return host
}

// NewBackendClient creates a new backend client
func NewBackendClient() *BackendClient {
	return &BackendClient{
		BackendURL: GetBackendURL(),
	}
}

// Register registers the device with the backend
func (bc *BackendClient) Register() error {
	log.Println("[HEARTBEAT] Registering device with backend")

	macID, _ := getMACAddress()
	hostname, _ := os.Hostname()
	osDetails := getOSDetails()
	location := hostname // Use hostname as default location

	req := RegisterRequest{
		MacID:     macID,
		Location:  location,
		OSDetails: osDetails,
	}

	body, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/api/register", bc.BackendURL)

	// Create request manually to ensure body is properly sent
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("registration request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	var regResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("failed to decode registration response: %w", err)
	}

	bc.DeviceID = regResp.DeviceID
	bc.RecoveryKey = regResp.RecoveryKey

	log.Printf("[HEARTBEAT] Registration successful - Device ID: %s", bc.DeviceID)
	return nil
}

// SendHeartbeat sends a heartbeat to the backend and returns the action
func (bc *BackendClient) SendHeartbeat() (string, error) {
	if bc.DeviceID == "" {
		return "NONE", fmt.Errorf("device not registered")
	}

	req := HeartbeatRequest{
		DeviceID: bc.DeviceID,
	}

	body, err := json.Marshal(req)
	if err != nil {
		log.Printf("[HEARTBEAT] JSON marshal error: %v", err)
		return "NONE", err
	}
	log.Printf("[HEARTBEAT] Sending body: %s", string(body))
	url := fmt.Sprintf("%s/api/heartbeat", bc.BackendURL)

	// Create request manually to ensure body is properly sent
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[HEARTBEAT] Failed to create request: %v", err)
		return "NONE", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.ContentLength = int64(len(body))

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("[HEARTBEAT] Request failed: %v", err)
		return "NONE", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("[HEARTBEAT] Got status %d: %s", resp.StatusCode, string(respBody))
		return "NONE", fmt.Errorf("heartbeat returned status %d", resp.StatusCode)
	}

	var hbResp HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&hbResp); err != nil {
		return "NONE", err
	}

	log.Printf("[HEARTBEAT] Action from backend: %s", hbResp.Action)
	return hbResp.Action, nil
}

// SendRecoveryKey sends the recovery key to the backend
func (bc *BackendClient) SendRecoveryKey(recoveryKey string) error {
	if bc.DeviceID == "" {
		return fmt.Errorf("device not registered")
	}

	req := RecoveryKeyRequest{
		DeviceID:    bc.DeviceID,
		RecoveryKey: recoveryKey,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal recovery key request: %w", err)
	}

	url := fmt.Sprintf("%s/api/recovery-key", bc.BackendURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create recovery key request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("recovery key request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("recovery key submission failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var keyResp RecoveryKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&keyResp); err != nil {
		return fmt.Errorf("failed to decode recovery key response: %w", err)
	}

	log.Printf("[HEARTBEAT] Recovery key submitted successfully: %s", keyResp.Message)
	return nil
}
func (bc *BackendClient) PollBackendWithHeartbeat(interval time.Duration, actionCallback func(string)) {
	log.Printf("[HEARTBEAT] Starting heartbeat poll every %v", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		action, err := bc.SendHeartbeat()
		if err != nil {
			log.Printf("[HEARTBEAT] Poll error: %v", err)
			continue
		}

		if action != "NONE" && action != "ACTIVE" {
			log.Printf("[HEARTBEAT] Backend action: %s", action)
			actionCallback(action)
		}
	}
}

// Helper functions

// getMACAddress retrieves the MAC address of the primary network interface
func getMACAddress() (string, error) {
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

// getOSDetails retrieves Windows OS details
func getOSDetails() string {
	cmd := exec.Command("cmd", "/C", "ver")
	output, err := cmd.Output()
	if err != nil {
		return "Windows (unknown version)"
	}
	return strings.TrimSpace(string(output))
}

// Legacy functions for compatibility

func SendHeartbeat() string {
	deviceID := os.Getenv("COMPUTERNAME")

	body, _ := json.Marshal(HeartbeatRequest{
		DeviceID: deviceID,
	})

	host := GetBackendURL()

	resp, err := http.Post(
		fmt.Sprintf("%s/api/heartbeat", host),
		"application/json",
		bytes.NewBuffer(body),
	)

	if err != nil {
		// Offline → do nothing now (grace logic later)
		return "NONE"
	}
	defer resp.Body.Close()

	var r HeartbeatResponse
	json.NewDecoder(resp.Body).Decode(&r)

	return r.Action
}

// SendRecoveryKeyLegacy legacy function for compatibility
func SendRecoveryKeyLegacy(key string) error {
	deviceID := os.Getenv("COMPUTERNAME")
	payload := map[string]string{
		"device_id":    deviceID,
		"recovery_key": key,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(
		fmt.Sprintf("%s/api/recovery-key", GetBackendURL()),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("backend returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}
