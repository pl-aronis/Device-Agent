package heartbeat

import (
	"bytes"
	"encoding/json"
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

	resp, err := http.Post(
		"https://YOUR_BACKEND/api/heartbeat",
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
