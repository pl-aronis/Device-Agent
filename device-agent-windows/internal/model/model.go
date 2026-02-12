package model

import (
	"time"
)

type GeoResponse struct {
	Lat float64
	Lon float64
}

type RegisterClientRequest struct {
	DeviceID     string      `json:"device_id"`
	AgentId      string      `json:"agent_id"`
	RegisteredAt time.Time   `json:"registered_at"`
	Location     GeoResponse `json:"location"`
	OSDetails    string      `json:"os_details"`
	RecoveryKey  string      `json:"recovery_key,omitempty"`
}

type RegisterClientResponse struct {
	DeviceID string `json:"device_id"`
	Status   string `json:"status"`
}

type HeartbeatRequest struct {
	DeviceID string      `json:"device_id"`
	AgentId  string      `json:"agent_id"`
	Location GeoResponse `json:"location"`
}

type HeartbeatResponse struct {
	Status string `json:"status"`
	Action string `json:"action"`
}

type RecoveryKeyUpdateRequest struct {
	DeviceID    string `json:"device_id"`
	ProtectorID string `json:"protector_id"`
	RecoveryKey string `json:"recovery_key"`
}

type RecoveryProtector struct {
	ID  string
	Key string
}

const (
	RegisterEndpoint    = "/api/register"
	HeartbeatEndpoint   = "/api/heartbeat"
	RecoveryKeyEndpoint = "/api/recovery-key"
	ManageBDE           = `C:\Windows\Sysnative\manage-bde.exe`
)
