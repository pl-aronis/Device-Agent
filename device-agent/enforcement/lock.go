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
UNIFIED LOCK FLOW (ALL WINDOWS EDITIONS)

1. Detect Windows edition
2. If Home → enable BitLocker policy via registry
3. Enable BitLocker if needed
4. Wait for encryption
5. Ensure protection ON
6. Ensure recovery key exists
7. Attempt BitLocker forced recovery
8. If BitLocker fails → soft lock fallback
*/

func EnforceDeviceLock() {
	log.Println("[LOCK] Starting unified device lock flow")

	edition := detectWindowsEdition()
	log.Println("[LOCK] Detected Windows edition:", edition)

	if edition == "Home" {
		log.Println("[LOCK] Windows Home detected — enabling BitLocker policy (No TPM)")
		enableHomeBitLockerPolicy()
	}

	if err := attemptHardLock(); err != nil {
		log.Println("[LOCK] Hard lock failed:", err)
		log.Println("[LOCK] Falling back to soft lock")
		softLock()
	}
}

//
// ---------------- HARD LOCK ATTEMPT ----------------
//

func attemptHardLock() error {
	if !isBitLockerPresent() {
		return errors.New("manage-bde not available")
	}

	if !isEncryptionComplete() {
		log.Println("[LOCK] BitLocker not fully encrypted — enabling/enforcing")
		if err := enableBitLocker(); err != nil {
			return err
		}
		if err := waitForEncryption(); err != nil {
			return err
		}
	}

	if !isProtectionOn() {
		log.Println("[LOCK] Enabling BitLocker protectors")
		run("manage-bde", "-protectors", "-enable", "C:")
	}

	_, err := ensureSingleRecoveryKey()
	if err != nil {
		return err
	}

	log.Println("[LOCK] Forcing BitLocker recovery")
	out := run("manage-bde", "-forcerecovery", "C:")

	if strings.Contains(strings.ToLower(out), "error") {
		return errors.New("forcerecovery command failed")
	}

	log.Println("[LOCK] Rebooting system")
	run("shutdown", "/r", "/t", "0")

	return nil
}

//
// ---------------- WINDOWS HOME POLICY ----------------
//

func enableHomeBitLockerPolicy() {
	run(
		"reg", "add",
		"HKLM\\SOFTWARE\\Policies\\Microsoft\\FVE",
		"/v", "EnableBDEWithNoTPM",
		"/t", "REG_DWORD",
		"/d", "1",
		"/f",
	)
}

//
// ---------------- SOFT LOCK (SAFE FALLBACK) ----------------
//

func softLock() {
	log.Println("[LOCK] Applying soft lock")

	run("rundll32.exe", "user32.dll,LockWorkStation")

	run("powershell", "-NoProfile", "-Command",
		`Get-LocalUser | Where-Object {$_.Enabled -eq $true -and $_.Name -ne "Administrator"} | Disable-LocalUser`,
	)

	run("powershell", "-NoProfile", "-Command",
		`Get-NetAdapter | Where-Object {$_.Status -eq "Up"} | Disable-NetAdapter -Confirm:$false`,
	)
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
	if strings.Contains(strings.ToLower(out), "error") {
		return errors.New("BitLocker enable failed")
	}
	return nil
}

func waitForEncryption() error {
	timeout := time.After(60 * time.Minute)
	ticker := time.Tick(15 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("encryption timeout")
		case <-ticker:
			if isEncryptionComplete() {
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
		out = run("manage-bde", "-protectors", "-add", "C:", "-RecoveryPassword")
		keys = re.FindAllString(out, -1)
	}

	if len(keys) == 0 {
		return "", errors.New("no recovery key available")
	}

	return keys[0], nil
}

//
// ---------------- COMMAND EXEC ----------------
//

func run(cmd string, args ...string) string {
	c := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		log.Printf("[CMD] %s %v failed: %v (%s)", cmd, args, err, stderr.String())
	}
	return stdout.String() + stderr.String()
}
