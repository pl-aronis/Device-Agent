package heartbeat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type Heartbeat struct {
	DeviceID string `json:"device_id"`
}

type Response struct {
	Action string `json:"action"`
}

func SendHeartbeat(deviceId, ip, port string) string {
	if deviceId == "" {
		hostname, _ := os.Hostname()
		deviceId = hostname
	}

	body, _ := json.Marshal(Heartbeat{
		DeviceID: deviceId,
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
