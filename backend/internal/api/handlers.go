package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"backend/internal/models"
	"backend/internal/storage"
)

// RegisterHandler handles device registration
func RegisterHandler(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req models.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			log.Printf("[API] Register: decode error: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		device, err := store.Register(req.DeviceID, req.MacID, req.Location, req.OSDetails)
		if err != nil {
			log.Printf("[API] Register: %v", err)
			http.Error(w, "registration failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		resp := models.RegisterResponse{
			DeviceID:    device.ID,
			Status:      device.Status,
			RecoveryKey: device.RecoveryKey,
		}
		json.NewEncoder(w).Encode(resp)
	}
}

// HeartbeatHandler handles device heartbeat
func HeartbeatHandler(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read and validate body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("[API] Heartbeat: read error: %v", err)
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		if len(body) == 0 {
			log.Printf("[API] Heartbeat: empty body")
			http.Error(w, "empty request body", http.StatusBadRequest)
			return
		}

		var req models.HeartbeatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			log.Printf("[API] Heartbeat: decode error: %v", err)
			http.Error(w, "invalid heartbeat payload", http.StatusBadRequest)
			return
		}

		if req.DeviceID == "" {
			log.Printf("[API] Heartbeat: empty device ID")
			http.Error(w, "device_id cannot be empty", http.StatusBadRequest)
			return
		}

		device, ok := store.Heartbeat(req.DeviceID)
		if !ok {
			log.Printf("[API] Heartbeat: device not found: %s", req.DeviceID)
			http.Error(w, "unknown device", http.StatusNotFound)
			return
		}

		log.Printf("[API] Heartbeat from device %s - status=%s", device.ID, device.Status)

		w.Header().Set("Content-Type", "application/json")
		resp := models.HeartbeatResponse{Action: device.Status}
		json.NewEncoder(w).Encode(resp)
	}
}

// RecoveryKeyHandler handles recovery key submission
func RecoveryKeyHandler(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req models.RecoveryKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			log.Printf("[API] RecoveryKey: decode error: %v", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.DeviceID == "" {
			http.Error(w, "device_id is required", http.StatusBadRequest)
			return
		}

		if req.RecoveryKey == "" {
			http.Error(w, "recovery_key is required", http.StatusBadRequest)
			return
		}

		_, ok := store.UpdateRecoveryKey(req.DeviceID, req.RecoveryKey)
		if !ok {
			log.Printf("[API] RecoveryKey: device not found: %s", req.DeviceID)
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		resp := models.RecoveryKeyResponse{
			Status:  "success",
			Message: "recovery key stored successfully",
		}
		json.NewEncoder(w).Encode(resp)
	}
}

// AdminSetHandler handles admin status updates
func AdminSetHandler(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		deviceID := r.URL.Query().Get("id")
		status := r.URL.Query().Get("status")

		if status == "" {
			http.Error(w, "missing status param", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if deviceID == "" {
			// Apply to all devices
			devices := store.AllDevices()
			for _, d := range devices {
				store.UpdateStatus(d.ID, status)
			}
			resp := models.AdminSetResponse{
				Status:  status,
				Message: "updated all devices",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		device, oldStatus, ok := store.UpdateStatus(deviceID, status)
		if !ok {
			log.Printf("[API] AdminSet: device not found: %s", deviceID)
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}

		resp := models.AdminSetResponse{
			DeviceID: device.ID,
			Status:   device.Status,
			Message:  "status updated",
		}

		// Return recovery key if transitioning from LOCK to ACTIVE
		if oldStatus == "LOCK" && status == "ACTIVE" {
			resp.RecoveryKey = device.RecoveryKey
			resp.Message = "device unlocked - recovery key displayed"
		}

		json.NewEncoder(w).Encode(resp)
	}
}

// AdminStatusHandler returns all device statuses
func AdminStatusHandler(store storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		devices := store.AllDevices()
		w.Header().Set("Content-Type", "application/json")
		resp := models.AdminStatusResponse{Devices: devices}
		json.NewEncoder(w).Encode(resp)
	}
}

// HealthHandler returns service health status
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}
}
