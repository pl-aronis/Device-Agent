package enforcement

import (
	"log"
	"os/exec"
)

func LockDevice() {
	log.Println("[LOCK] Locking device")

	// Lock the screen
	lockScreen()

	// Restrict network access
	restrictNetwork()
}

func lockScreen() {
	cmd := exec.Command("loginctl", "lock-session")
	err := cmd.Run()
	if err != nil {
		log.Println("[LOCK] Failed to lock screen:", err)
	}
}

func restrictNetwork() {
	cmd := exec.Command("nmcli", "networking", "off")
	err := cmd.Run()
	if err != nil {
		log.Println("[LOCK] Failed to restrict network:", err)
	}
}
