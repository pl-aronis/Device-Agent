package enforcement

import (
	"errors"
	"log"
	"time"

	"device-agent-windows/internal/config"
)

func (b *Bitlocker) enableProtectionWithRetry() (string, string, error) {

	log.Println("[STEP] Creating recovery password (Attempt 1)")
	key, id, err := b.addRecoveryPassword()
	if err != nil {
		return "", "", err
	}
	b.logRecoveryKey(key)

	log.Println("[STEP] Enabling protection (Attempt 1)")
	err = b.enableProtection()
	if err == nil {
		return key, id, nil
	}

	log.Println("[WARN] Protection failed. Retrying.")
	b.deleteProtectorByID(id)

	key2, id2, err := b.addRecoveryPassword()
	if err != nil {
		return "", "", err
	}
	b.logRecoveryKey(key2)

	err = b.enableProtection()
	if err != nil {
		b.deleteProtectorByID(id2)
		return "", "", err
	}

	return key2, id2, nil
}

func (b *Bitlocker) enableProtection() error {

	_, err := b.exec.Run(manageBDE, "-protectors", "-enable", "C:")
	if err != nil {
		return err
	}

	timeout := time.After(time.Duration(config.AppConfig.ProtectionTimeoutMinutes) * time.Minute)
	ticker := time.Tick(5 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("protection did not turn ON")
		case <-ticker:
			if b.isProtectionOn() {
				return nil
			}
		}
	}
}
