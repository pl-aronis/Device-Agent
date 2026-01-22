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
BITLOCKER-ONLY ENFORCEMENT FLOW (NON-DESTRUCTIVE)

CHANGES FROM PREVIOUS VERSION:
- DO NOT delete existing protectors
- ADD our own recovery protector if none exists
- RETRY enabling protection until it flips ON
- HANDLE BitLocker async behavior correctly

FLOW:
1. Detect Windows edition
2. If Home → enable No-TPM policy (best-effort)
3. Verify manage-bde.exe is usable
4. Ensure encryption
5. Ensure at least one recovery protector exists
6. Enable protection (retry window)
7. Verify protection ON
8. Print recovery key
9. Sleep 5 minutes
10. Force recovery
11. Reboot
*/

const manageBDE = `C:\Windows\Sysnative\manage-bde.exe`

func EnforceDeviceLock() {
	log.Println("========== DEVICE LOCK START ==========")

	edition := detectWindowsEdition()
	log.Printf("[INFO] Detected Windows edition: %s\n", edition)

	if edition == "Home" {
		log.Println("[INFO] Windows Home detected")
		log.Println("[INFO] Applying BitLocker No-TPM policy (best-effort)")
		enableHomeBitLockerPolicy()
	}

	log.Println("[INFO] Verifying BitLocker CLI availability")
	if !isBitLockerCLIExecutable() {
		log.Println("[FATAL] BitLocker CLI unavailable")
		log.Println("[ABORT] Device lock aborted")
		return
	}

	if !isEncrypted() {
		log.Println("[INFO] Enabling BitLocker encryption")
		if err := enableEncryption(); err != nil {
			log.Println("[FATAL] Failed to enable encryption:", err)
			return
		}
	}

	log.Println("[INFO] Waiting for encryption to complete")
	if err := waitForEncryption(); err != nil {
		log.Println("[FATAL]", err)
		return
	}

	log.Println("[INFO] Ensuring recovery protector exists")
	key, err := ensureRecoveryProtector()
	if err != nil {
		log.Println("[FATAL] Failed to ensure recovery protector:", err)
		return
	}

	log.Println("[INFO] Enabling BitLocker protection")
	if err := enableProtectionWithRetry(); err != nil {
		log.Println("[FATAL]", err)
		return
	}

	log.Println("=================================================")
	log.Println("[RECOVERY KEY — SAVE THIS NOW]")
	log.Println(key)
	log.Println("=================================================")

	log.Println("[INFO] Sleeping for 5 minutes before enforcing lock")
	time.Sleep(5 * time.Minute)

	log.Println("[ACTION] Forcing BitLocker recovery")
	if _, err := run(manageBDE, "-forcerecovery", "C:"); err != nil {
		log.Println("[FATAL] Failed to force recovery:", err)
		return
	}

	log.Println("[ACTION] Rebooting system")
	run("shutdown", "/r", "/t", "0")

	log.Println("========== DEVICE LOCK TRIGGERED ==========")
}

//
// ---------------- WINDOWS HOME POLICY ----------------
//

func enableHomeBitLockerPolicy() {
	run(
		"reg",
		"add",
		`HKLM\SOFTWARE\Policies\Microsoft\FVE`,
		"/v", "EnableBDEWithNoTPM",
		"/t", "REG_DWORD",
		"/d", "1",
		"/f",
	)
}

//
// ---------------- OS DETECTION ----------------
//

func detectWindowsEdition() string {
	out, err := run(
		"reg",
		"query",
		`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"/v", "EditionID",
	)
	if err != nil {
		return "Unknown"
	}

	switch {
	case strings.Contains(out, "Enterprise"):
		return "Enterprise"
	case strings.Contains(out, "Professional"):
		return "Pro"
	case strings.Contains(out, "Core"):
		return "Home"
	default:
		return "Unknown"
	}
}

//
// ---------------- BITLOCKER CORE ----------------
//

func isBitLockerCLIExecutable() bool {
	_, err := run(manageBDE, "-status")
	return err == nil
}

func isEncrypted() bool {
	out, err := run(manageBDE, "-status", "C:")
	return err == nil && !strings.Contains(out, "Fully Decrypted")
}

func enableEncryption() error {
	_, err := run(manageBDE, "-on", "C:", "-RecoveryPassword")
	return err
}

func waitForEncryption() error {
	timeout := time.After(60 * time.Minute)
	ticker := time.Tick(15 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("encryption timeout exceeded")
		case <-ticker:
			out, err := run(manageBDE, "-status", "C:")
			if err == nil && strings.Contains(out, "Percentage Encrypted: 100") {
				log.Println("[INFO] Encryption complete")
				return nil
			}
		}
	}
}

//
// ---------------- PROTECTOR HANDLING ----------------
//

func ensureRecoveryProtector() (string, error) {
	out, err := run(manageBDE, "-protectors", "-get", "C:")
	if err != nil {
		return "", err
	}

	keyRe := regexp.MustCompile(`(\d{6}-){7}\d{6}`)
	keys := keyRe.FindAllString(out, -1)

	if len(keys) > 0 {
		log.Println("[INFO] Existing recovery protector found")
		return keys[0], nil
	}

	log.Println("[INFO] No recovery protector found — creating one")
	out, err = run(manageBDE, "-protectors", "-add", "C:", "-RecoveryPassword")
	if err != nil {
		return "", err
	}

	key := keyRe.FindString(out)
	if key == "" {
		return "", errors.New("failed to extract recovery key")
	}

	return key, nil
}

func enableProtectionWithRetry() error {
	log.Println("[INFO] Attempting to enable BitLocker protection (single attempt)")
	_ = runWithFullLogging(manageBDE, "-protectors", "-enable", "C:")

	timeout := time.After(2 * time.Minute)
	ticker := time.Tick(5 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("protection did not turn ON (status parsing mismatch)")
		case <-ticker:
			out, err := run(manageBDE, "-status", "C:")
			if err != nil {
				continue
			}

			// 🔧 NORMALIZE OUTPUT (THIS IS THE FIX)
			n := strings.ToLower(out)
			n = strings.ReplaceAll(n, "\t", " ")
			n = strings.Join(strings.Fields(n), " ")

			log.Println("[DEBUG] Normalized BitLocker status:")
			log.Println(n)

			if strings.Contains(n, "protection status: protection on") {
				log.Println("[INFO] Protection is ON (verified)")
				return nil
			}

			log.Println("[INFO] Waiting for protection to turn ON")
		}
	}
}

//
// ---------------- COMMAND EXEC ----------------
//

func run(cmd string, args ...string) (string, error) {
	c := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()

	if err != nil {
		if stderr.Len() > 0 {
			log.Printf("[STDERR] %s\n", stderr.String())
		}
		return "", err
	}

	return stdout.String(), nil
}

func runWithFullLogging(cmd string, args ...string) error {
	log.Println("[CMD] Executing:", cmd, strings.Join(args, " "))

	c := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()

	log.Println("[CMD] Exit error:", err)

	if stdout.Len() > 0 {
		log.Println("[CMD] STDOUT:")
		log.Println(strings.TrimSpace(stdout.String()))
	} else {
		log.Println("[CMD] STDOUT: <empty>")
	}

	if stderr.Len() > 0 {
		log.Println("[CMD] STDERR:")
		log.Println(strings.TrimSpace(stderr.String()))
	} else {
		log.Println("[CMD] STDERR: <empty>")
	}

	return err
}
