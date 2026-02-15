package enforcement

import (
	"device-agent-windows/internal/helper"
	"device-agent-windows/internal/model"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

func isBitLockerCLIExecutable() bool {
	_, err := helper.RunCommand(model.ManageBDE, "-status")
	return err == nil
}

func isEncrypted() bool {
	out, err := helper.RunCommand(model.ManageBDE, "-status", "C:")
	return err == nil && !strings.Contains(out, "Fully Decrypted")
}

func enableEncryption() error {
	_, err := helper.RunCommand(model.ManageBDE, "-on", "C:", "-RecoveryPassword")
	return err
}

func createRecoveryProtector() (*model.RecoveryProtector, error) {
	out, err := helper.RunCommand(model.ManageBDE, "-protectors", "-add", "C:", "-RecoveryPassword")
	if err != nil {
		return nil, err
	}

	keyRe := regexp.MustCompile(`(\d{6}-){7}\d{6}`)
	idRe := regexp.MustCompile(`ID:\s*{[^}]+}`)

	key := keyRe.FindString(out)
	idMatch := idRe.FindString(out)

	if key == "" || idMatch == "" {
		return nil, errors.New("failed to extract recovery key or ID")
	}

	id := strings.TrimPrefix(idMatch, "ID: ")

	return &model.RecoveryProtector{
		ID:  id,
		Key: key,
	}, nil
}

func deleteProtector(id string) error {
	_, err := helper.RunCommand(model.ManageBDE, "-protectors", "-delete", "C:", "-id", id)
	if err != nil {
		return fmt.Errorf("failed to delete protector with ID %s: %w", id, err)
	}
	return nil
}

func SetUpLockPreReq() error {

	if !isBitLockerCLIExecutable() {
		return errors.New("BitLocker CLI is not available")
	}

	if !isEncrypted() {
		fmt.Println("[INFO] Enabling BitLocker encryption")
		if err := enableEncryption(); err != nil {
			return fmt.Errorf("Failed to enable BitLocker encryption: %w", err)
		}
	}

	fmt.Println("[INFO] Creating first agent recovery password")
	first, err := createRecoveryProtector()
	if err != nil {
		fmt.Println("[FATAL] Failed to create first recovery protector:", err)
	}

	if err := enableProtection(); err != nil {
		fmt.Printf("[ERROR] First attempt to enable protection failed: %v", err)
	}

	fmt.Println("[INFO] Deleting previous agent recovery password:", first.ID)
	deleteProtector(first.ID)

	first, err = createRecoveryProtector()
	if err != nil {
		fmt.Println("[FATAL] Failed to create second recovery protector:", err)
		return errors.New("Failed to create second recovery protector")
	}

	if err := enableProtection(); err != nil {
		return fmt.Errorf("Failed to enable protection after recovery key update: %w", err)
	}

	return nil
}
