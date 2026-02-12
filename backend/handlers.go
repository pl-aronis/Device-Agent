package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type RegisterReq struct {
	DeviceID  string `json:"device_id,omitempty"`
	MacID     string `json:"mac_id"`
	Location  string `json:"location"`
	OSDetails string `json:"os_details"`
	BIOSPass  string `json:"bios_pass,omitempty"`
}

type RegisterResp struct {
	DeviceID    string `json:"device_id"`
	Status      string `json:"status"`
	RecoveryKey string `json:"recovery_key"`
}

type HeartbeatReq struct {
	DeviceID string `json:"device_id"`
}

type HeartbeatResp struct {
	Action string `json:"action"`
}

type RecoveryKeyUpdateReq struct {
	DeviceID    string `json:"device_id"`
	ProtectorID string `json:"protector_id"`
	RecoveryKey string `json:"recovery_key"`
}

func registerHandler(s *Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		d, err := s.Register(req.DeviceID, req.MacID, req.Location, req.OSDetails, req.BIOSPass)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(RegisterResp{DeviceID: d.ID, Status: d.Status, RecoveryKey: d.RecoveryKey})
	}
}

func recoveryKeyHandler(s *Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RecoveryKeyUpdateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.DeviceID == "" || req.ProtectorID == "" || req.RecoveryKey == "" {
			http.Error(w, "device_id, protector_id and recovery_key are required", http.StatusBadRequest)
			return
		}

		d, ok := s.UpdateRecoveryKey(req.DeviceID, req.ProtectorID, req.RecoveryKey)
		if !ok {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"device_id":    d.ID,
			"status":       "ok",
			"protector_id": d.RecoveryProtectorID,
		})
	}
}

func heartbeatHandler(s *Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Read body to check if it's empty
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("[HEARTBEAT] Error reading body: %v", err)
			http.Error(w, "failed to read body: "+err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("[HEARTBEAT] Raw body: %q (length: %d)", string(body), len(body))

		if len(body) == 0 {
			log.Printf("[HEARTBEAT] ERROR: Empty request body received")
			http.Error(w, "empty request body", http.StatusBadRequest)
			return
		}

		var req HeartbeatReq
		if err := json.Unmarshal(body, &req); err != nil {
			log.Printf("[HEARTBEAT] Decode error: %v", err)
			http.Error(w, "invalid heartbeat payload: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.DeviceID == "" {
			log.Printf("[HEARTBEAT] Empty device ID in request")
			http.Error(w, "device_id cannot be empty", http.StatusBadRequest)
			return
		}
		d, ok := s.Heartbeat(req.DeviceID)
		if !ok {
			log.Printf("[HEARTBEAT] Device not found: %s", req.DeviceID)
			http.Error(w, "unknown device", http.StatusNotFound)
			return
		}
		log.Printf("Heartbeat from %s - status=%s", d.ID, d.Status)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HeartbeatResp{Action: d.Status})
	}
}

// adminSetHandler: /admin/set?id=...&status=ACTIVE|LOCK
type AdminSetResp struct {
	DeviceID            string `json:"device_id,omitempty"`
	Status              string `json:"status"`
	RecoveryKey         string `json:"recovery_key,omitempty"`
	RecoveryProtectorID string `json:"recovery_protector_id,omitempty"`
	Message             string `json:"message,omitempty"`
}

func adminSetHandler(s *Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		status := r.URL.Query().Get("status")
		if status == "" {
			http.Error(w, "missing status param", http.StatusBadRequest)
			return
		}
		if id == "" {
			// apply to all devices
			devs := s.AllDevices()
			for _, d := range devs {
				_, _, _ = s.UpdateStatus(d.ID, status)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(AdminSetResp{Status: status, Message: "updated all devices"})
			return
		}
		d, oldStatus, ok := s.UpdateStatus(id, status)
		if !ok {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := AdminSetResp{
			DeviceID:            d.ID,
			Status:              d.Status,
			RecoveryProtectorID: d.RecoveryProtectorID,
		}
		// Return recovery key if transitioning from LOCK to ACTIVE
		if oldStatus == "LOCK" && status == "ACTIVE" {
			resp.RecoveryKey = d.RecoveryKey
			resp.Message = "device unlocked - recovery key displayed"
		} else {
			resp.Message = "status updated"
		}
		json.NewEncoder(w).Encode(resp)
	}
}

func adminStatusHandler(s *Storage) http.HandlerFunc {
	log.Println("adminStatusHandler called")
	return func(w http.ResponseWriter, r *http.Request) {
		out := s.AllDevices()
		json.NewEncoder(w).Encode(out)
	}
}
