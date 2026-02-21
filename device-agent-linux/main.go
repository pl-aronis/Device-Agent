package main

import (
	"device-agent-linux/credentials"
	"device-agent-linux/enforcement"
	"device-agent-linux/registration"
	"device-agent-linux/service"
	"log"
	"os"
)

func main() {
	log.Println("========================================")
	log.Println("Starting Device Agent for Linux")
	log.Println("========================================")

	// Get backend configuration
	ip := os.Getenv("BACKEND_IP")
	if ip == "" {
		ip = "192.168.1.11"
	}
	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("[STARTUP] Backend configured: http://%s:%s", ip, port)

	// Optional: Set up BIOS password if not already set
	var biosPassword string
	if os.Getenv("SETUP_BIOS_PASSWORD") == "true" {
		log.Println("[STARTUP] BIOS password setup enabled")
		biosPassword = enforcement.GenerateBIOSPassword()
		biosPassword = "admin" // remove after testing
		if err := enforcement.SetBIOSPassword(biosPassword); err != nil {
			log.Printf("[STARTUP] Failed to set BIOS password: %v", err)
		}
	} else {
		log.Println("[STARTUP] BIOS password setup skipped (set SETUP_BIOS_PASSWORD=true to enable)")
	}

	// Register device with backend at startup
	log.Println("[STARTUP] Registering device with backend...")
	deviceInfo, err := registration.RegisterDevice(ip, port, biosPassword)
	if err != nil {
		log.Printf("[STARTUP] ❌ Failed to register device: %v", err)
		log.Println("[STARTUP] Continuing with fallback device ID...")
		log.Println("[STARTUP] WARNING: Device may not function properly without registration")
	} else {
		log.Println("[STARTUP] ✅ Device registered successfully")
		log.Printf("[STARTUP] Device ID: %s", deviceInfo.DeviceID)
		log.Printf("[STARTUP] Status: %s", deviceInfo.Status)

		// Log all credentials
		creds := &credentials.Credentials{
			DeviceID:     deviceInfo.DeviceID,
			RecoveryKey:  deviceInfo.RecoveryKey,
			BIOSPassword: biosPassword,
			BackendIP:    ip,
			BackendPort:  port,
			MACAddress:   deviceInfo.MacID,
			Hostname:     deviceInfo.Location,
			OSDetails:    deviceInfo.OSDetails,
		}

		if err := credentials.LogCredentials(creds); err != nil {
			log.Printf("[STARTUP] Failed to log credentials: %v", err)
		}
	}

	log.Println("========================================")
	log.Println("[STARTUP] Starting main service loop...")
	log.Println("========================================")

	// Start the main service loop
	service.Run(ip, port)
}
