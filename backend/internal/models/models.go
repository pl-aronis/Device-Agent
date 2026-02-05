package models

import "time"

// Device represents a managed device
type Device struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	RecoveryKey string    `json:"recovery_key"`
	MacID       string    `json:"mac_id"`
	Location    string    `json:"location"`
	OSDetails   string    `json:"os_details"`
	LastSeen    time.Time `json:"last_seen"`
}

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

// AdminSetRequest represents an admin status update request
type AdminSetRequest struct {
	DeviceID string `json:"device_id,omitempty"`
	Status   string `json:"status"`
}

// AdminSetResponse represents an admin status update response
type AdminSetResponse struct {
	DeviceID    string `json:"device_id,omitempty"`
	Status      string `json:"status"`
	RecoveryKey string `json:"recovery_key,omitempty"`
	Message     string `json:"message,omitempty"`
}

// AdminStatusResponse represents the response for getting all device statuses
type AdminStatusResponse struct {
	Devices []Device `json:"devices"`
}
