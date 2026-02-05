package heartbeat

import (
	"bytes"
	"device-agent-linux/registration"
	"encoding/json"
	"fmt"
	"net/http"
)

type Heartbeat struct {
	DeviceID string `json:"device_id"`
}

type Response struct {
	Action string `json:"action"`
}

func SendHeartbeat(ip string, port string) string {
	// Use registered device ID
	deviceID := registration.GetDeviceID()

	body, _ := json.Marshal(Heartbeat{
		DeviceID: deviceID,
	})

	resp, err := http.Post(
		fmt.Sprintf("http://%s:%s/api/heartbeat", ip, port),
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
