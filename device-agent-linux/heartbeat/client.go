package heartbeat

import (
	"bytes"
	"device-agent-linux/registration"
	"encoding/json"
	"fmt"
	"log"
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
		log.Println("Heartbeat failed: ", err)
		return ""
	}
	defer resp.Body.Close()

	var r Response
	json.NewDecoder(resp.Body).Decode(&r)

	return r.Action
}
