package service

import (
	"log"
	"time"

	"device-agent/heartbeat"
	"device-agent/enforcement"
)

func Run() {
	for {
		action := heartbeat.SendHeartbeat()

		if action == "LOCK" {
			log.Println("Policy violation → locking device")
			enforcement.ForceBitLockerRecovery()
		}

		time.Sleep(6 * time.Hour)
	}
}
