package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

var (
	currentAction = "NONE" // NONE, WARNING, LOCK
	mu            sync.Mutex
)

type HeartbeatReq struct {
	DeviceID string `json:"device_id"`
}

type HeartbeatResp struct {
	Action string `json:"action"`
}

type KeyReq struct {
	DeviceID    string `json:"device_id"`
	RecoveryKey string `json:"recovery_key"`
}

func main() {
	// 1. /api/heartbeat
	http.HandleFunc("/api/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		action := currentAction
		mu.Unlock()

		log.Printf("Heartbeat received. Returning action: %s", action)

		json.NewEncoder(w).Encode(HeartbeatResp{Action: action})
	})

	// 2. /api/key
	http.HandleFunc("/api/key", func(w http.ResponseWriter, r *http.Request) {
		var req KeyReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		log.Printf("🔥🔥🔥 RECEIVED RECOVERY KEY for %s: %s 🔥🔥🔥", req.DeviceID, req.RecoveryKey)
		w.WriteHeader(200)
	})

	// 3. /admin/set?action=XXX
	http.HandleFunc("/admin/set", func(w http.ResponseWriter, r *http.Request) {
		newAction := r.URL.Query().Get("action")
		if newAction != "NONE" && newAction != "WARNING" && newAction != "LOCK" {
			fmt.Fprintf(w, "Invalid action. Use NONE, WARNING, or LOCK")
			return
		}

		mu.Lock()
		currentAction = newAction
		mu.Unlock()

		log.Printf("Admin changed action to: %s", newAction)
		fmt.Fprintf(w, "Action updated to %s", newAction)
	})

	// 4. /admin/status
	http.HandleFunc("/admin/status", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintf(w, "Current Action: %s", currentAction)
	})

	log.Println("Mock Backend listening on :8080")
	log.Println("Usage: curl http://localhost:8080/admin/set?action=WARNING")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
