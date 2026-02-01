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
	deviceID := os.Getenv("HOSTNAME")

	body, _ := json.Marshal(Heartbeat{
		DeviceID: deviceID,
	})

	host := os.Getenv("BACKEND_HOST")
	if host == "" {
		host = "192.168.1.11:8080"
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
