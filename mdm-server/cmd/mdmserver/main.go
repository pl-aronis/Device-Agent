package main

import (
	"log"
	"net/http"
	"os"

	"mdm-server/internal/api"
	"mdm-server/internal/apns"
	"mdm-server/internal/commands"
	"mdm-server/internal/dep"
	"mdm-server/internal/store"
)

func main() {
	log.Println("Starting Custom Apple MDM Server")

	// Initialize stores
	deviceStore := store.NewDeviceStore()
	commandQueue := commands.NewQueue()

	// Initialize APNs Client
	var apnsClient *apns.Client
	if _, err := os.Stat("certs/push.p12"); err == nil {
		client, err := apns.NewClient("certs/push.p12", "password", "com.example.mdm")
		if err != nil {
			log.Printf("Error initializing APNs: %v", err)
		} else {
			apnsClient = client
			log.Println("APNs Client initialized")
		}
	} else {
		log.Println("APNs certificate not found, push notifications validation will be skipped")
	}

	// Initialize DEP Client (Stubbed)
	// In production requires OAuth tokens from ABM
	depClient := dep.NewClient("key", "secret", "access", "access_secret")

	// Initialize handlers
	checkinHandler := api.CheckinHandler(deviceStore, commandQueue)
	adminHandler := api.AdminHandler(deviceStore, commandQueue, apnsClient)
	depHandler := api.SyncDEPHandler(depClient)

	// MDM Endpoints (Must be exposed via HTTPS)
	http.HandleFunc("/mdm/checkin", checkinHandler)
	http.HandleFunc("/mdm/connect", api.ConnectHandler(commandQueue))

	// Admin API Endpoints
	http.HandleFunc("/api/devices", adminHandler.ListDevices)
	http.HandleFunc("/api/devices/", adminHandler.DeviceAction)
	http.HandleFunc("/api/dep/sync", depHandler) // Trigger sync with ABM

	// Helper route to check server status
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Apple MDM Server Running"))
	})

	log.Println("Server listening on :8080 (Requires HTTPS proxy/tunnel for device connectivity)")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
