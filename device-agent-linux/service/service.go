package service

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"device-agent-linux/enforcement"
	"device-agent-linux/heartbeat"
	"device-agent-linux/tamper"
)

const (
	MaxOfflineDuration = 6 * time.Hour
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
	ioutil.WriteFile(LockCacheFile, []byte("LOCK"), 0644)
}

func isLockCached() bool {
	if _, err := os.Stat(LockCacheFile); err == nil {
		return true
	}
	return false
}

func Run() {
	lastSuccessfulHeartbeat := time.Now()

	for {
		action := heartbeat.SendHeartbeat()

		if action == "LOCK" {
			log.Println("Policy violation → locking device")
			cacheLockCommand()
			enforcement.LockDevice()
		} else if action == "WARNING" {
			log.Println("Policy warning → displaying alert")
			// TODO
			// enforcement.ShowWarning()
		} else if action == "NONE" {
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
