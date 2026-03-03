package enforcement

import (
	"log"
	"time"

	"device-agent-windows/internal/config"
)

const manageBDE = "manage-bde"

type Bitlocker struct {
	exec Executor
}

func NewBitlocker(exec Executor) *Bitlocker {
	return &Bitlocker{exec: exec}
}

func (b *Bitlocker) Enforce() (string, string, error) {

	log.Println("========== DEVICE LOCK START ==========")

	if !b.isAdmin() {
		return "", "", ErrNotAdmin
	}

	log.Println("[STEP] Applying No-TPM policy")
	b.enableNoTPMPolicy()

	if !b.isBitLockerCLIExecutable() {
		return "", "", ErrCLIUnavailable
	}

	log.Println("[STEP] Checking encryption status")
	if !b.isEncrypted() {
		if err := b.enableEncryption(); err != nil {
			return "", "", err
		}
	}

	if err := b.waitForEncryption(); err != nil {
		return "", "", err
	}

	key, id, err := b.enableProtectionWithRetry()
	if err != nil {
		return "", "", err
	}

	time.Sleep(time.Duration(config.AppConfig.ForceRecoverySleepSec) * time.Second)

	b.exec.Run(manageBDE, "-forcerecovery", "C:")

	return key, id, nil
}
