package heartbeat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Heartbeat struct {
	DeviceID string `json:"device_id"`
}

type Response struct {
	Action string `json:"action"`
}

func SendHeartbeat() string {
	deviceID := os.Getenv("COMPUTERNAME")

	body, _ := json.Marshal(Heartbeat{
		DeviceID: deviceID,
	})

	host := os.Getenv("BACKEND_HOST")
	if host == "" {
		host = "192.168.12.82:8080"
	}

	resp, err := http.Post(
		fmt.Sprintf("http://%s/api/heartbeat", host),
		"application/json",
		bytes.NewBuffer(body),
	)

	if err != nil {
		// Offline → do nothing now (grace logic later)
		return "NONE"
	}
	defer resp.Body.Close()

	var r Response
	json.NewDecoder(resp.Body).Decode(&r)

	return r.Action
}

func SendRecoveryKey(key string) error {
	deviceID := os.Getenv("COMPUTERNAME")
	payload := map[string]string{
		"device_id":    deviceID,
		"recovery_key": key,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(
		"http://localhost:8080/api/key",
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
