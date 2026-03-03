package enforcement

import (
	"errors"
	"strings"
	"time"

	"device-agent-windows/internal/config"
)

func (b *Bitlocker) enableEncryption() error {
	_, err := b.exec.Run(manageBDE, "-on", "C:", "-RecoveryPassword")
	return err
}

func (b *Bitlocker) waitForEncryption() error {

	timeout := time.After(time.Duration(config.AppConfig.EncryptionTimeoutMinutes) * time.Minute)
	ticker := time.Tick(15 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("encryption timeout")
		case <-ticker:
			out, _ := b.exec.Run(manageBDE, "-status", "C:")
			if strings.Contains(out, "Percentage Encrypted: 100") {
				return nil
			}
		}
	}
}
