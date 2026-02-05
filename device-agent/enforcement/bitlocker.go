package enforcement

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// RecoveryProtector represents a BitLocker recovery protector
type RecoveryProtector struct {
	ID  string
	Key string
}

// BitLockerManager handles BitLocker operations
type BitLockerManager struct {
	cliPath string
}

// NewBitLockerManager creates a new BitLocker manager instance
func NewBitLockerManager() *BitLockerManager {
	return &BitLockerManager{
		cliPath: `C:\Windows\Sysnative\manage-bde.exe`,
	}
}

// IsAvailable checks if BitLocker CLI is available
func (bm *BitLockerManager) IsAvailable() bool {
	_, err := bm.run("-status")
	return err == nil
}

// IsEncrypted checks if the C: drive is encrypted
func (bm *BitLockerManager) IsEncrypted() bool {
	out, err := bm.run("-status", "C:")
	return err == nil && !strings.Contains(out, "Fully Decrypted")
}

// EnableEncryption enables BitLocker encryption on C: drive
func (bm *BitLockerManager) EnableEncryption() error {
	_, err := bm.run("-on", "C:", "-RecoveryPassword")
	return err
}

// WaitForEncryption waits for encryption to complete (max 60 minutes)
func (bm *BitLockerManager) WaitForEncryption() error {
	timeout := time.After(60 * time.Minute)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return errors.New("encryption timeout after 60 minutes")
		case <-ticker.C:
			out, _ := bm.run("-status", "C:")
			if strings.Contains(out, "Percentage Encrypted: 100") {
				log.Println("[BITLOCKER] Encryption completed successfully")
				return nil
			}
		}
	}
}

// CreateRecoveryProtector creates a new recovery password protector
func (bm *BitLockerManager) CreateRecoveryProtector() (*RecoveryProtector, error) {
	out, err := bm.run("-protectors", "-add", "C:", "-RecoveryPassword")
	if err != nil {
		return nil, fmt.Errorf("failed to create recovery protector: %w", err)
	}

	protector, err := bm.parseRecoveryProtector(out)
	if err != nil {
		return nil, err
	}

	log.Printf("[BITLOCKER] Created recovery protector: ID=%s", protector.ID)
	return protector, nil
}

// DeleteProtector deletes a protector by ID
func (bm *BitLockerManager) DeleteProtector(id string) error {
	_, err := bm.run("-protectors", "-delete", "C:", "-id", id)
	if err != nil {
		log.Printf("[BITLOCKER] Failed to delete protector %s: %v", id, err)
		return err
	}
	log.Printf("[BITLOCKER] Deleted protector: %s", id)
	return nil
}

// ListRecoveryProtectors lists all recovery protector IDs
func (bm *BitLockerManager) ListRecoveryProtectors() (map[string]bool, error) {
	out, err := bm.run("-protectors", "-get", "C:")
	if err != nil {
		return nil, fmt.Errorf("failed to list protectors: %w", err)
	}

	idRe := regexp.MustCompile(`ID:\s*{[^}]+}`)
	ids := make(map[string]bool)
	for _, m := range idRe.FindAllString(out, -1) {
		ids[strings.TrimPrefix(m, "ID: ")] = true
	}
	return ids, nil
}

// IsProtectionEnabled checks if BitLocker protection is enabled
func (bm *BitLockerManager) IsProtectionEnabled() bool {
	out, err := bm.run("-status", "C:")
	if err != nil {
		return false
	}
	normalized := normalize(out)
	return strings.Contains(normalized, "protection status: protection on")
}

// EnableProtection attempts to enable BitLocker protection
func (bm *BitLockerManager) EnableProtection() error {
	_, err := bm.run("-protectors", "-enable", "C:")
	if err != nil {
		return err
	}
	time.Sleep(3 * time.Second)
	return nil
}

// ForceRecoveryAndReboot forces recovery reboot
func (bm *BitLockerManager) ForceRecoveryAndReboot() error {
	_, _ = bm.run("-forcerecovery", "C:")
	// Give the system time before reboot
	time.Sleep(1 * time.Minute)
	_, _ = exec.Command("shutdown", "/r", "/t", "0").CombinedOutput()
	return nil
}

// Private helper methods

func (bm *BitLockerManager) run(args ...string) (string, error) {
	cmd := exec.Command(bm.cliPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func (bm *BitLockerManager) parseRecoveryProtector(output string) (*RecoveryProtector, error) {
	keyRe := regexp.MustCompile(`(\d{6}-){7}\d{6}`)
	idRe := regexp.MustCompile(`ID:\s*{[^}]+}`)

	key := keyRe.FindString(output)
	idMatch := idRe.FindString(output)

	if key == "" || idMatch == "" {
		return nil, errors.New("failed to extract recovery key or ID from output")
	}

	return &RecoveryProtector{
		ID:  strings.TrimPrefix(idMatch, "ID: "),
		Key: key,
	}, nil
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "\t", " ")
	return strings.Join(strings.Fields(s), " ")
}
