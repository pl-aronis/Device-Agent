package main

import (
	"log"

	"device-agent/service"
)

func main() {
	// Initialize logger with proper formatting
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("========== DEVICE AGENT STARTUP ==========")

	// Start the service
	service.Run()
}
