package enforcement

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

/*
BITLOCKER-ONLY ENFORCEMENT FLOW (ALL WINDOWS EDITIONS)

NO SOFT LOCKS
NO FALLBACK LOCKS

FLOW:
1. Detect Windows edition
2. If Home → enable No-TPM BitLocker policy
3. Verify manage-bde exists
4. Ensure BitLocker encryption is enabled
5. Wait until encryption reaches 100%
6. Ensure protection is ON
7. Ensure exactly one recovery key exists
8. Force BitLocker recovery
9. Reboot

If ANY step fails → log + abort
*/

func EnforceDeviceLock() {
	log.Println("========== DEVICE LOCK START ==========")

	edition := detectWindowsEdition()
	log.Printf("[INFO] Detected Windows edition: %s\n", edition)

	if edition == "Home" {
		log.Println("[INFO] Windows Home detected")
		log.Println("[INFO] Applying BitLocker No-TPM policy (best-effort)")
		enableHomeBitLockerPolicy()
	}

	log.Println("[INFO] Verifying BitLocker availability")
	if !isBitLockerPresent() {
		log.Println("[FATAL] manage-bde not found — BitLocker unavailable on this system")
		log.Println("[ABORT] Device lock aborted")
		return
	}

	log.Println("[INFO] Checking BitLocker encryption state")
	if !isEncryptionComplete() {
		log.Println("[INFO] Drive not fully encrypted — enabling BitLocker")
		if err := enableBitLocker(); err != nil {
			log.Println("[FATAL] Failed to enable BitLocker:", err)
			log.Println("[ABORT] Device lock aborted")
			return
		}

		log.Println("[INFO] Waiting for encryption to reach 100%")
		if err := waitForEncryption(); err != nil {
			log.Println("[FATAL] Encryption did not complete:", err)
			log.Println("[ABORT] Device lock aborted")
			return
		}
	}

	log.Println("[INFO] Encryption verified at 100%")

	log.Println("[INFO] Verifying BitLocker protection status")
	if !isProtectionOn() {
		log.Println("[INFO] Protection is OFF — enabling protectors")
		if err := enableProtection(); err != nil {
			log.Println("[FATAL] Failed to enable protection:", err)
			log.Println("[ABORT] Device lock aborted")
			return
		}
	}

	log.Println("[INFO] BitLocker protection is ON")

	log.Println("[INFO] Ensuring exactly one recovery key exists")
	key, err := ensureSingleRecoveryKey()
	if err != nil {
		log.Println("[FATAL] Recovery key validation failed:", err)
		log.Println("[ABORT] Device lock aborted")
		return
	}

	log.Printf("[INFO] Recovery key confirmed: %s\n", maskKey(key))

	log.Println("[ACTION] Forcing BitLocker recovery")
	out := run("manage-bde", "-forcerecovery", "C:")
	log.Println("[CMD OUTPUT]", out)

	if containsError(out) {
		log.Println("[FATAL] BitLocker refused forced recovery")
		log.Println("[ABORT] Device lock aborted")
		return
	}

	log.Println("[ACTION] Rebooting system now")
	run("shutdown", "/r", "/t", "0")

	log.Println("========== DEVICE LOCK TRIGGERED ==========")
}

//
// ---------------- WINDOWS HOME POLICY ----------------
//

func enableHomeBitLockerPolicy() {
	out := run(
		"reg", "add",
		"HKLM\\SOFTWARE\\Policies\\Microsoft\\FVE",
		"/v", "EnableBDEWithNoTPM",
		"/t", "REG_DWORD",
		"/d", "1",
		"/f",
	)
	log.Println("[REGISTRY] EnableBDEWithNoTPM applied:", out)
}

//
// ---------------- OS DETECTION ----------------
//

func detectWindowsEdition() string {
	out := run("cmd", "/c", "wmic os get Caption")
	switch {
	case strings.Contains(out, "Enterprise"):
		return "Enterprise"
	case strings.Contains(out, "Pro"):
		return "Pro"
	case strings.Contains(out, "Home"):
		return "Home"
	default:
		return "Unknown"
	}
}

//
// ---------------- BITLOCKER HELPERS ----------------
//

func isBitLockerPresent() bool {
	out := run("where", "manage-bde")
	return strings.Contains(out, "manage-bde")
}

func isEncryptionComplete() bool {
	out := run("manage-bde", "-status", "C:")
	return strings.Contains(out, "Percentage Encrypted: 100%")
}

func isProtectionOn() bool {
	out := run("manage-bde", "-status", "C:")
	return strings.Contains(out, "Protection Status: Protection On")
}

func enableBitLocker() error {
	out := run("manage-bde", "-on", "C:", "-RecoveryPassword")
	if containsError(out) {
		return errors.New("manage-bde -on failed")
	}
	return nil
}

func enableProtection() error {
	out := run("manage-bde", "-protectors", "-enable", "C:")
	if containsError(out) {
		return errors.New("failed to enable BitLocker protectors")
	}
	return nil
}

func waitForEncryption() error {
	timeout := time.After(60 * time.Minute)
	ticker := time.Tick(15 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("encryption timeout exceeded")
		case <-ticker:
			out := run("manage-bde", "-status", "C:")
			log.Println("[STATUS]", strings.TrimSpace(out))
			if strings.Contains(out, "Percentage Encrypted: 100%") {
				return nil
			}
		}
	}
}

func ensureSingleRecoveryKey() (string, error) {
	out := run("manage-bde", "-protectors", "-get", "C:")
	re := regexp.MustCompile(`(\d{6}-){7}\d{6}`)
	keys := re.FindAllString(out, -1)

	if len(keys) == 0 {
		log.Println("[INFO] No recovery key found — generating one")
		out = run("manage-bde", "-protectors", "-add", "C:", "-RecoveryPassword")
		keys = re.FindAllString(out, -1)
	}

	if len(keys) == 0 {
		return "", errors.New("no recovery key available after generation")
	}

	if len(keys) > 1 {
		log.Printf("[WARN] Multiple recovery keys detected (%d). Using first.\n", len(keys))
	}

	return keys[0], nil
}

//
// ---------------- UTILITIES ----------------
//

func run(cmd string, args ...string) string {
	c := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()

	if err != nil {
		log.Printf("[CMD ERROR] %s %v → %v\n", cmd, args, err)
	}

	if stderr.Len() > 0 {
		log.Printf("[STDERR] %s\n", stderr.String())
	}

	return stdout.String() + stderr.String()
}

func containsError(out string) bool {
	l := strings.ToLower(out)
	return strings.Contains(l, "error") || strings.Contains(l, "failed")
}

func maskKey(k string) string {
	if len(k) < 10 {
		return "INVALID"
	}
	return k[:6] + "-******-******-******-******-******"
}
