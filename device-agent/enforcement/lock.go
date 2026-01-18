package enforcement

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"device-agent/heartbeat"
)

/*
FLOW:
1. Ensure BitLocker is enabled
2. Wait until encryption = 100%
3. Ensure Protection = ON
4. Ensure exactly one recovery key exists
5. Send recovery key to backend
6. Force recovery
7. Reboot
*/

func ForceBitLockerRecovery() {
	log.Println("[LOCK] Starting automated BitLocker lockdown sequence")

	// STEP 1 — Ensure BitLocker is ON
	if !isBitLockerEnabled() {
		log.Println("[LOCK] BitLocker not enabled — enabling now")
		if err := enableBitLocker(); err != nil {
			log.Println("[FATAL] Failed to enable BitLocker:", err)
			return
		}
	}

	// STEP 2 — Wait for encryption to complete
	log.Println("[LOCK] Waiting for encryption to reach 100%")
	if err := waitForEncryption(); err != nil {
		log.Println("[FATAL] Encryption did not complete:", err)
		return
	}

	// STEP 3 — Ensure protection is ON
	if !isProtectionOn() {
		log.Println("[LOCK] Protection is OFF — enabling protectors")
		if err := enableProtection(); err != nil {
			log.Println("[FATAL] Failed to enable protection:", err)
			return
		}
	}

	// STEP 4 — Ensure exactly ONE recovery key
	log.Println("[LOCK] Ensuring exactly one recovery key exists")
	key, err := ensureSingleRecoveryKey()
	if err != nil {
		log.Println("[FATAL] Recovery key handling failed:", err)
		return
	}

	// STEP 5 — Send key to backend
	log.Println("[LOCK] Sending recovery key to backend")
	if err := heartbeat.SendRecoveryKey(key); err != nil {
		log.Println("[FATAL] Backend did not confirm key receipt — aborting lock")
		return
	}

	// STEP 6 — Force recovery
	log.Println("[LOCK] Forcing BitLocker recovery")
	exec.Command("manage-bde", "-forcerecovery", "C:").Run()

	// STEP 7 — Reboot
	log.Println("[LOCK] Rebooting system now")
	exec.Command("shutdown", "/r", "/t", "0").Run()
}

//
// ----------------- Helper Functions -----------------
//

func isBitLockerEnabled() bool {
	out := run("manage-bde", "-status", "C:")
	return strings.Contains(out, "Conversion Status") &&
		!strings.Contains(out, "Fully Decrypted")
}

func enableBitLocker() error {
	cmd := exec.Command("manage-bde", "-on", "C:", "-RecoveryPassword")
	return cmd.Run()
}

func waitForEncryption() error {
	timeout := time.After(60 * time.Minute)
	ticker := time.Tick(10 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("encryption timeout exceeded")
		case <-ticker:
			out := run("manage-bde", "-status", "C:")
			if strings.Contains(out, "Percentage Encrypted: 100%") {
				log.Println("[LOCK] Encryption complete")
				return nil
			}
		}
	}
}

func isProtectionOn() bool {
	out := run("manage-bde", "-status", "C:")
	return strings.Contains(out, "Protection Status: Protection On")
}

func enableProtection() error {
	cmd := exec.Command("manage-bde", "-protectors", "-enable", "C:")
	return cmd.Run()
}

func ensureSingleRecoveryKey() (string, error) {
	out := run("manage-bde", "-protectors", "-get", "C:")

	re := regexp.MustCompile(`(\d{6}-){7}\d{6}`)
	keys := re.FindAllString(out, -1)

	// If none exist → create one
	if len(keys) == 0 {
		log.Println("[LOCK] No recovery key found — creating one")
		out = run("manage-bde", "-protectors", "-add", "C:", "-RecoveryPassword")
		keys = re.FindAllString(out, -1)
	}

	if len(keys) == 0 {
		return "", errors.New("failed to generate recovery key")
	}

	// If more than one → delete extras
	if len(keys) > 1 {
		log.Println("[LOCK] Multiple recovery keys detected — cleaning up")
		deleteExtraProtectors()
	}

	return keys[0], nil
}

func deleteExtraProtectors() {
	out := run("manage-bde", "-protectors", "-get", "C:")

	idRe := regexp.MustCompile(`ID:\s*{[^}]+}`)
	ids := idRe.FindAllString(out, -1)

	// Keep the first, delete the rest
	for i := 1; i < len(ids); i++ {
		id := strings.TrimPrefix(ids[i], "ID: ")
		exec.Command("manage-bde", "-protectors", "-delete", "C:", "-id", id).Run()
	}
}

func run(cmd string, args ...string) string {
	c := exec.Command(cmd, args...)
	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf
	_ = c.Run()
	return buf.String()
}
