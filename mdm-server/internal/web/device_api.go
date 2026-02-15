package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// handleAPIDeviceOperations handles /api/devices/{udid}/...
func (h *Handler) handleAPIDeviceOperations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.NotFound(w, r)
		return
	}

	udid := parts[3]

	// Find device by UDID
	device, ok := h.deviceStore.GetDevice(udid)
	if !ok || device == nil {
		http.Error(w, `{"status":"error","message":"Device not found"}`, 404)
		return
	}

	// /api/devices/{udid}/commands - get command history
	if len(parts) >= 5 && parts[4] == "commands" {
		commands, err := h.commandStore.ListByDevice(device.ID, 50)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"status":"error","message":"%s"}`, err), 500)
			return
		}
		json.NewEncoder(w).Encode(commands)
		return
	}

	http.NotFound(w, r)
}
