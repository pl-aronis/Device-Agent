package service

import (
	"log"
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
		HeartbeatInterval: 30 * time.Second, // Poll backend every 30 seconds
		RegistrationRetry: 5 * time.Second,  // Retry registration every 5 seconds if failed
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

	// Step 1: Setup locking prerequisites
	if err := setupPhase(); err != nil {
		log.Printf("[FATAL] Setup phase failed: %v", err)
		// Continue anyway - device might still be able to lock later
	}

	// Step 2: Register with backend
	backendClient := heartbeat.NewBackendClient()
	if err := registerPhase(backendClient, config); err != nil {
		log.Printf("[FATAL] Registration failed: %v", err)
		// This is fatal - we can't proceed without registration
		return
	}

	// Step 3: Start polling backend with heartbeat
	pollingPhase(backendClient, config)
}

// setupPhase initializes locking prerequisites (BitLocker, encryption, etc.)
func setupPhase() error {
	log.Println("[SETUP PHASE] Initializing locking prerequisites")
	err := enforcement.SetupLockingPrerequisites()
	if err != nil {
		log.Printf("[SETUP PHASE] Warning: %v", err)
		return err
	}
	log.Println("[SETUP PHASE] Complete - device is ready for locking")
	return nil
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

	// Start the polling goroutine
	go client.PollBackendWithHeartbeat(config.HeartbeatInterval, func(action string) {
		mu.Lock()
		defer mu.Unlock()

		switch action {
		case "LOCK":
			if !lockRequested {
				lockRequested = true
				log.Println("[POLLING PHASE] Backend command received: LOCK")
				lockChan <- struct{}{}
			}
		case "ACTIVE":
			// Device should be active, no action needed
			log.Println("[POLLING PHASE] Backend command: ACTIVE (no action)")
		default:
			log.Printf("[POLLING PHASE] Unknown action: %s", action)
		}
	})
	// Wait for lock signal
	<-lockChan
	log.Println("[ACTION] Executing device lock")

	if err := enforcement.EnforceDeviceLock(); err != nil {
		log.Printf("[ACTION] Lock enforcement failed: %v", err)
	}
}
