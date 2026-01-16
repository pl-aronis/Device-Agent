package enforcement

import (
	"os/exec"
	"log"
)

func ForceBitLockerRecovery() {
	log.Println("Forcing BitLocker recovery")

	cmd := exec.Command(
		"manage-bde",
		"-forcerecovery",
		"C:",
	)

	err := cmd.Run()
	if err != nil {
		log.Println("Failed to force BitLocker recovery:", err)
	}

	// Immediate reboot
	exec.Command("shutdown", "/r", "/t", "0").Run()
}
