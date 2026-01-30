package api

import (
	"encoding/json"
	"net/http"

	"mdm-server/internal/dep"
)

// SyncDEPHandler triggers a manual sync with Apple Business Manager
func SyncDEPHandler(depClient *dep.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		devices, _, err := depClient.FetchDevices("")
		if err != nil {
			http.Error(w, "Failed to sync DEP: "+err.Error(), 500)
			return
		}

		// TODO: Save devices to store

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "synced",
			"count":   len(devices),
			"devices": devices,
		})
	}
}
