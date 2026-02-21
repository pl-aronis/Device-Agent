package service

import (
	"log"
	"os"
	"os/exec"
	"time"

	"device-agent-linux/enforcement"
	"device-agent-linux/heartbeat"
	"device-agent-linux/tamper"
)

const (
	MaxOfflineDuration = 1 * time.Minute // update after testing
	LockCacheFile      = "/var/lib/device-agent-linux/lock.cache"
)

func configureFirewall(backendIP string) {
	log.Println("Configuring firewall rules")

	// Block all outbound traffic
	exec.Command("iptables", "-P", "OUTPUT", "DROP").Run()

	// Allow traffic to the backend
	exec.Command("iptables", "-A", "OUTPUT", "-d", backendIP, "-j", "ACCEPT").Run()
}

func cacheLockCommand() {
	os.WriteFile(LockCacheFile, []byte("LOCK"), 0644)
}

func isLockCached() bool {
	if _, err := os.Stat(LockCacheFile); err == nil {
		return true
	}
	return false
}

func Run(ip string, port string) {
	lastSuccessfulHeartbeat := time.Now()

	for {
		action := heartbeat.SendHeartbeat(ip, port)

		switch action {
		case "LOCK":
			log.Println("Policy violation → locking device")
			cacheLockCommand()
			configureFirewall(ip)
			enforcement.LockDevice()
		case "WARNING":
			log.Println("Policy warning → displaying alert")
			// TODO
			// enforcement.ShowWarning()
		case "NONE":
			lastSuccessfulHeartbeat = time.Now()
		}

		tamper.CheckIntegrity()

		if isLockCached() {
			log.Println("[OFFLINE LOCK] Cached lock command found → locking device")
			enforcement.LockDevice()
		}

		if time.Since(lastSuccessfulHeartbeat) > MaxOfflineDuration {
			log.Println("[FAIL-CLOSED] Backend unreachable for too long → locking device")
			cacheLockCommand()
			enforcement.LockDevice()
		}

		time.Sleep(10 * time.Second) // Polling interval
	}
}
