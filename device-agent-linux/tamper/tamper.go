package tamper

import (
	"log"
	"os/exec"
)

func CheckIntegrity() {
	if !isServiceRunning() {
		log.Println("[TAMPER] Service not running. Attempting to restart.")
		restartService()
	}

	if isBinaryModified() {
		log.Println("[TAMPER] Binary modified. Alerting backend.")
		alertBackend("Binary modified")
	}
}

func isServiceRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "device-agent-linux.service")
	err := cmd.Run()
	return err == nil
}

func restartService() {
	exec.Command("systemctl", "restart", "device-agent-linux.service").Run()
}

func isBinaryModified() bool {
	// Placeholder for checksum verification logic
	return false
}

func alertBackend(message string) {
	log.Printf("[TAMPER] Alert: %s", message)
	// Placeholder for backend alert logic
}
