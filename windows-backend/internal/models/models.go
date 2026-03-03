package models

import "time"

type Device struct {
	AgentID     string    `json:"agent_id"`
	MacID       string    `json:"mac_id"`
	OS          string    `json:"os"`
	Arch        string    `json:"arch"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	ShouldLock  bool      `json:"should_lock"`
	IsLocked    bool      `json:"is_locked"`
	RecoveryKey string    `json:"recovery_key"`
	RecoveryID  string    `json:"recovery_id"`
	LastSeen    time.Time `json:"last_seen"`
}
