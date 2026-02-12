package enforcement

import (
	"device-agent-windows/internal/helper"
	"device-agent-windows/internal/model"
	"errors"
	"fmt"
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
	log.Println("[SUCCESS] Protection ON - forcing recovery")
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

func enableProtection() error {
	if tryEnableProtection() {
		return nil
	}
	return errors.New("failed to enable BitLocker protection")
}

func EnforceDeviceLock() error {
	if _, err := EnforceDeviceLockWithRecovery(); err != nil {
		return err
	}
	return ForceRecoveryAndReboot()
}

func EnforceDeviceLockWithRecovery() (*model.RecoveryProtector, error) {
	if !isBitLockerCLIExecutable() {
		return nil, errors.New("BitLocker CLI is not available")
	}

	if !isEncrypted() {
		log.Println("[INFO] Enabling BitLocker encryption")
		if err := enableEncryption(); err != nil {
			return nil, fmt.Errorf("failed to enable BitLocker encryption: %w", err)
		}
	}

	recovery, err := createRecoveryProtector()
	if err != nil {
		return nil, fmt.Errorf("failed to create recovery protector: %w", err)
	}

	if err := enableProtection(); err != nil {
		return nil, err
	}
	return recovery, nil
}

func ForceRecoveryAndReboot() error {
	if err := forceRecoveryAndReboot(); err != nil {
		return fmt.Errorf("failed to force recovery and reboot: %w", err)
	}
	return nil
}

func ReleaseDeviceLock(managedProtectorID string) error {
	_, err := helper.RunCommand(model.ManageBDE, "-protectors", "-disable", "C:")
	if err != nil {
		return fmt.Errorf("failed to disable BitLocker protection: %w", err)
	}

	if managedProtectorID == "" {
		return nil
	}

	if err := deleteProtector(managedProtectorID); err != nil {
		return err
	}
	return nil
}
