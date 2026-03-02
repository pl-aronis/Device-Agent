package enforcement

import (
	"device-agent-windows/internal/helper"
	"device-agent-windows/internal/model"
	"errors"
	"log"
	"strings"
	"time"
)

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "\t", " ")
	return strings.Join(strings.Fields(s), " ")
}

func tryEnableProtection() bool {
	log.Println("[ACTION] Attempting to enable BitLocker protection")
	helper.RunCommand(model.ManageBDE, "-protectors", "-enable", "C:")

	time.Sleep(3 * time.Second)

	output, err := helper.RunCommand(model.ManageBDE, "-status", "C:")
	if err != nil {
		return false
	}

	normalizedOutput := normalize(output)
	return strings.Contains(normalizedOutput, "protection status: protection on")
}

func forceRecoveryAndReboot() error {
	log.Println("[SUCCESS] Protection ON — forcing recovery")
	_, err := helper.RunCommand(model.ManageBDE, "-forcerecovery", "C:")
	if err != nil {
		log.Println("[ERROR] Failed to force recovery:", err)
		return errors.New("failed to force BitLocker recovery")
	}

	_, err = helper.RunCommand("shutdown", "/r", "/t", "0")
	if err != nil {
		log.Println("[ERROR] Failed to reboot:", err)
		return errors.New("failed to reboot the system")
	}
	return nil
}

