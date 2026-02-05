package service

import (
	"log"
	"os/exec"
	"sync"
	"time"

	"device-agent/enforcement"
	"device-agent/heartbeat"
)

// ServiceConfig holds the configuration for the service
type ServiceConfig struct {
	HeartbeatInterval time.Duration
	RegistrationRetry time.Duration
}

// DefaultConfig provides sensible defaults
func DefaultConfig() ServiceConfig {
	return ServiceConfig{
		HeartbeatInterval: 3 * time.Second, // Poll backend every 30 seconds
		RegistrationRetry: 5 * time.Second, // Retry registration every 5 seconds if failed
	}
}

// Run is the main entry point for the device agent service
func Run() {
	config := DefaultConfig()
	runWithConfig(config)
}

// runWithConfig runs the service with a given configuration
func runWithConfig(config ServiceConfig) {
	log.Println("========== DEVICE AGENT SERVICE START ==========")

	// Step 1: Register with backend
	backendClient := heartbeat.NewBackendClient()
	if err := registerPhase(backendClient, config); err != nil {
		log.Printf("[FATAL] Registration failed: %v", err)
		// This is fatal - we can't proceed without registration
		return
	}

	// Step 2: Start polling backend with heartbeat
	pollingPhase(backendClient, config)
}

// registerPhase attempts to register the device with the backend
func registerPhase(client *heartbeat.BackendClient, config ServiceConfig) error {
	log.Println("[REGISTER PHASE] Starting device registration")

	for {
		err := client.Register()
		if err == nil {
			log.Printf("[REGISTER PHASE] Registration successful - Device ID: %s", client.DeviceID)
			log.Println("[REGISTER PHASE] Status set to ACTIVE")
			return nil
		}

		log.Printf("[REGISTER PHASE] Registration attempt failed: %v. Retrying in %v...", err, config.RegistrationRetry)
		time.Sleep(config.RegistrationRetry)
	}
}

// pollingPhase continuously polls the backend for actions
func pollingPhase(client *heartbeat.BackendClient, config ServiceConfig) {
	log.Printf("[POLLING PHASE] Starting heartbeat polling every %v", config.HeartbeatInterval)
	log.Println("[POLLING PHASE] Waiting for backend commands...")

	// Use a channel to handle lock actions
	lockChan := make(chan struct{})
	var mu sync.Mutex
	lockRequested := false

	// Start the polling goroutine. When a LOCK action is received the
	// callback will perform the locking synchronously. This blocks the
	// heartbeat loop (so no further heartbeats are sent) until the lock
	// operation completes. After locking finishes the callback signals
	// completion on lockChan so the main goroutine can continue.
	go client.PollBackendWithHeartbeat(config.HeartbeatInterval, func(action string) {
		mu.Lock()
		defer mu.Unlock()

		switch action {
		case "LOCK":
			if !lockRequested {
				lockRequested = true
				log.Println("[POLLING PHASE] Backend command received: LOCK")

				// Perform lock synchronously here to pause further heartbeats
				recoveryProtector, err := enforcement.EnforceDeviceLock()
				if err != nil {
					log.Printf("[ERROR] Device lock failed: %v", err)
					lockChan <- struct{}{}
					return
				}

				if recoveryProtector == nil {
					log.Println("[ERROR] Recovery protector is nil")
					lockChan <- struct{}{}
					return
				}

				// Step 1: Send the recovery key to the backend
				log.Println("[HEARTBEAT-1] Sending recovery key to backend...")
				if err := client.SendRecoveryKey(recoveryProtector.Key); err != nil {
					log.Printf("[ERROR] Failed to send recovery key to backend: %v", err)
					lockChan <- struct{}{}
					return
				}
				log.Println("[HEARTBEAT-1] Recovery key sent successfully")

				// Small delay to ensure backend processes the key
				time.Sleep(1 * time.Second)

				// Step 2: Send confirmation heartbeat that everything is successful
				log.Println("[HEARTBEAT-2] Sending final success heartbeat...")
				if _, err := client.SendHeartbeat(); err != nil {
					log.Printf("[ERROR] Failed to send success heartbeat: %v", err)
					lockChan <- struct{}{}
					return
				}
				log.Println("[HEARTBEAT-2] Success heartbeat sent")

				// Small delay before reboot
				time.Sleep(2 * time.Second)

				// Step 3: Restart the machine
				log.Println("[REBOOT] Initiating system restart in 10 seconds...")
				enforceSystemRestart()

				// Signal completion to the main goroutine
				lockChan <- struct{}{}
			}
		case "ACTIVE":
			// Device should be active, no action needed
			log.Println("[POLLING PHASE] Backend command: ACTIVE (no action)")
		default:
			log.Printf("[POLLING PHASE] Unknown action: %s", action)
		}
	})

	// Wait for lock to complete (signaled by the callback)
	<-lockChan
	log.Println("[ACTION] Device lock completed - waiting for system restart...")

	// Keep the service running during restart sequence
	select {}
}

// enforceSystemRestart initiates a system restart
func enforceSystemRestart() {
	// Use os/exec to run the shutdown command
	// This will restart the machine after a 10-second delay
	cmd := exec.Command("shutdown", "/r", "/t", "10", "/c", "Device locked and security measures applied. System will restart.")

	log.Println("[REBOOT] Executing system restart with 10-second delay...")

	if err := cmd.Start(); err != nil {
		log.Printf("[ERROR] Failed to initiate restart: %v", err)
		// Fallback: try alternative method
		cmd2 := exec.Command("cmd", "/C", "shutdown /r /t 10")
		if err := cmd2.Start(); err != nil {
			log.Printf("[ERROR] Fallback restart also failed: %v", err)
		}
	} else {
		log.Println("[REBOOT] System restart command issued successfully")
	}
}
