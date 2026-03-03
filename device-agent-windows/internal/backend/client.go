package backend

import (
	"bytes"
	"device-agent-windows/internal/device"
	"encoding/json"
	"net/http"
)

type Client struct {
	baseURL string
}

func NewClient(base string) *Client {
	return &Client{baseURL: base}
}

func (c *Client) Register(info device.DeviceInfo) (*RegisterResponse, error) {
	body, _ := json.Marshal(info)
	resp, err := http.Post(c.baseURL+"/register", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var r RegisterResponse
	json.NewDecoder(resp.Body).Decode(&r)
	return &r, nil
}

func (c *Client) ReAuthenticate(mac string) (*ReAuthResponse, error) {
	body, _ := json.Marshal(map[string]string{"mac_id": mac})
	resp, err := http.Post(c.baseURL+"/re-authenticate", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var r ReAuthResponse
	json.NewDecoder(resp.Body).Decode(&r)
	return &r, nil
}

func (c *Client) Heartbeat(mac string, loc device.Location) (*HeartbeatResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"mac_id": mac,
		"lat":    loc.Latitude,
		"lon":    loc.Longitude,
	})

	resp, err := http.Post(c.baseURL+"/heartbeat", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var r HeartbeatResponse
	json.NewDecoder(resp.Body).Decode(&r)
	return &r, nil
}

func (c *Client) SendLockSuccess(mac, key, id string) {
	json.Marshal(map[string]string{
		"mac_id": mac,
		"key":    key,
		"id":     id,
	})
}

func (c *Client) SendLockFailure(mac, reason string) {
	json.Marshal(map[string]string{
		"mac_id": mac,
		"error":  reason,
	})
}
