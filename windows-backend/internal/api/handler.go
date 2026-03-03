package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"windows-backend/internal/models"
	"windows-backend/internal/service"
)

type Handler struct {
	service *service.DeviceService
}

func NewHandler(s *service.DeviceService) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var d models.Device
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(d.MacID) == "" {
		http.Error(w, "mac_id is required", http.StatusBadRequest)
		return
	}

	registered := h.service.Register(d)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"agent_id": registered.AgentID,
	})
}

func (h *Handler) ReAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MacID string `json:"mac_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.MacID) == "" {
		http.Error(w, "mac_id is required", http.StatusBadRequest)
		return
	}

	d, found := h.service.ReAuthenticate(req.MacID)
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"agent_id":    d.AgentID,
		"recovery_id": d.RecoveryID,
	})
}

func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MacID string  `json:"mac_id"`
		Lat   float64 `json:"lat"`
		Lon   float64 `json:"lon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.MacID) == "" {
		http.Error(w, "mac_id is required", http.StatusBadRequest)
		return
	}

	d := h.service.Heartbeat(req.MacID, req.Lat, req.Lon)
	if strings.TrimSpace(d.MacID) == "" {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"should_lock": d.ShouldLock,
	})
}

func (h *Handler) LockSuccess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MacID string `json:"mac_id"`
		Key   string `json:"key"`
		ID    string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.MacID) == "" {
		http.Error(w, "mac_id is required", http.StatusBadRequest)
		return
	}

	h.service.MarkLocked(req.MacID, req.Key, req.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) LockFailure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MacID string `json:"mac_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.MacID) == "" {
		http.Error(w, "mac_id is required", http.StatusBadRequest)
		return
	}

	h.service.MarkLockFailed(req.MacID)
	w.WriteHeader(http.StatusNoContent)
}

type adminDeviceResponse struct {
	ID                  string      `json:"id"`
	Status              string      `json:"status"`
	Location            string      `json:"location"`
	MacID               string      `json:"mac_id"`
	OSDetails           string      `json:"os_details"`
	LastSeen            interface{} `json:"last_seen"`
	RecoveryKey         string      `json:"recovery_key"`
	RecoveryProtectorID string      `json:"recovery_protector_id"`
	IsLocked            bool        `json:"is_locked"`
	ShouldLock          bool        `json:"should_lock"`
}

func (h *Handler) AdminStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	devices := h.service.ListDevices()
	resp := make([]adminDeviceResponse, 0, len(devices))

	for _, d := range devices {
		status := "ACTIVE"
		if d.IsLocked || d.ShouldLock {
			status = "LOCK"
		}

		var lastSeen interface{}
		if !d.LastSeen.IsZero() {
			lastSeen = d.LastSeen
		}

		resp = append(resp, adminDeviceResponse{
			ID:                  d.AgentID,
			Status:              status,
			Location:            fmt.Sprintf("%.5f, %.5f", d.Latitude, d.Longitude),
			MacID:               d.MacID,
			OSDetails:           strings.TrimSpace(fmt.Sprintf("%s/%s", d.OS, d.Arch)),
			LastSeen:            lastSeen,
			RecoveryKey:         d.RecoveryKey,
			RecoveryProtectorID: d.RecoveryID,
			IsLocked:            d.IsLocked,
			ShouldLock:          d.ShouldLock,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) AdminSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type request struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	var req request
	if r.Method == http.MethodGet {
		req.ID = r.URL.Query().Get("id")
		req.Status = r.URL.Query().Get("status")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
	}

	req.ID = strings.TrimSpace(req.ID)
	req.Status = strings.TrimSpace(req.Status)
	if req.ID == "" || req.Status == "" {
		http.Error(w, "id and status are required", http.StatusBadRequest)
		return
	}

	updated, err := h.service.SetAdminStatus(req.ID, req.Status)
	if err != nil {
		if err.Error() == "device not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status := "ACTIVE"
	if updated.IsLocked || updated.ShouldLock {
		status = "LOCK"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":                    updated.AgentID,
		"status":                status,
		"recovery_key":          updated.RecoveryKey,
		"recovery_protector_id": updated.RecoveryID,
		"should_lock":           updated.ShouldLock,
		"is_locked":             updated.IsLocked,
	})
}
