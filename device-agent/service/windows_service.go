package service

import (
	"log"

	"device-agent/enforcement"
)

func Run() {
	for {
		// action := heartbeat.SendHeartbeat()

		//if action == "LOCK" {
		log.Println("Policy violation → locking device")
		enforcement.EnforceDeviceLock()
		//} else if action == "WARNING" {
		//	log.Println("Policy warning → displaying alert")
		//	enforcement.ShowWarning()
		//}

		//time.Sleep(10 * time.Second) // Reduced for testing
	}
}
