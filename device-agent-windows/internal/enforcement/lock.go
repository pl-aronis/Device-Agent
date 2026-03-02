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
GENERIC BITLOCKER ENFORCEMENT

SECURITY MODEL:
- Apply No-TPM policy (safe on all editions)
- Ensure encryption
- Rotate recovery password
- Delete old recovery protectors
- Create fresh recovery password
- Enable protection
- Force recovery
- Reboot

TPM protectors are NOT touched.
*/

const manageBDE = `C:\Windows\Sysnative\manage-bde.exe`

func EnforceDeviceLock() {
	log.Println("========== DEVICE LOCK START ==========")

	// Always apply No-TPM policy (safe everywhere)
	log.Println("[INFO] Applying BitLocker No-TPM policy")
	enableNoTPMPolicy()

	if !isBitLockerCLIExecutable() {
		log.Println("[FATAL] BitLocker CLI unavailable")
		return
	}

	if !isEncrypted() {
		log.Println("[INFO] Enabling BitLocker encryption")
		if err := enableEncryption(); err != nil {
			log.Println("[FATAL]", err)
			return
		}
	}

	if err := waitForEncryption(); err != nil {
		log.Println("[FATAL]", err)
		return
	}

	log.Println("[INFO] Rotating recovery password")
	key, err := rotateRecoveryPassword()
	if err != nil {
		log.Println("[FATAL]", err)
		return
	}

	log.Println("[INFO] Enabling BitLocker protection")
	if err := enableProtectionAndVerify(); err != nil {
		log.Println("[FATAL]", err)
		return
	}

	// 🔴 TEMPORARY — REMOVE IN PROD
	log.Println("=================================================")
	log.Println("[RECOVERY KEY — TEMPORARY LOG]")
	log.Println(key)
	log.Println("=================================================")

	log.Println("[INFO] Sleeping for 5 minutes")
	time.Sleep(1 * time.Minute)

	log.Println("[ACTION] Forcing BitLocker recovery")
	if _, err := run(manageBDE, "-forcerecovery", "C:"); err != nil {
		log.Println("[FATAL]", err)
		return
	}

	log.Println("[ACTION] Rebooting")
	run("shutdown", "/r", "/t", "0")
}

//
// ---------------- RECOVERY KEY ROTATION ----------------
//

func rotateRecoveryPassword() (string, error) {
	out, err := run(manageBDE, "-protectors", "-get", "C:")
	if err != nil {
		return "", err
	}

	idRe := regexp.MustCompile(`ID:\s*{[^}]+}`)
	typeRe := regexp.MustCompile(`Numerical Password`)

	lines := strings.Split(out, "\n")
	var idsToDelete []string

	for i := 0; i < len(lines); i++ {
		if typeRe.MatchString(lines[i]) && i+1 < len(lines) {
			if id := idRe.FindString(lines[i+1]); id != "" {
				idsToDelete = append(idsToDelete, strings.TrimPrefix(id, "ID: "))
			}
		}
	}

	for _, id := range idsToDelete {
		log.Println("[INFO] Deleting old recovery protector:", id)
		run(manageBDE, "-protectors", "-delete", "C:", "-id", id)
	}

	log.Println("[INFO] Creating new recovery password")
	out, err = run(manageBDE, "-protectors", "-add", "C:", "-RecoveryPassword")
	if err != nil {
		return "", err
	}

	keyRe := regexp.MustCompile(`(\d{6}-){7}\d{6}`)
	key := keyRe.FindString(out)
	if key == "" {
		return "", errors.New("failed to extract recovery key")
	}

	return key, nil
}

//
// ---------------- PROTECTION ENABLE ----------------
//

func enableProtectionAndVerify() error {
	log.Println("[INFO] Enabling protection")
	_ = runWithFullLogging(manageBDE, "-protectors", "-enable", "C:")

	timeout := time.After(2 * time.Minute)
	ticker := time.Tick(5 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("protection did not turn ON")
		case <-ticker:
			out, err := run(manageBDE, "-status", "C:")
			if err != nil {
				continue
			}

			n := strings.ToLower(out)
			n = strings.ReplaceAll(n, "\t", " ")
			n = strings.Join(strings.Fields(n), " ")

			if strings.Contains(n, "protection status: protection on") {
				log.Println("[INFO] Protection is ON")
				return nil
			}
		}
	}
}

//
// ---------------- CORE HELPERS ----------------
//

func isEncrypted() bool {
	out, err := run(manageBDE, "-status", "C:")
	return err == nil && !strings.Contains(out, "Fully Decrypted")
}

func waitForEncryption() error {
	timeout := time.After(60 * time.Minute)
	ticker := time.Tick(15 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("encryption timeout")
		case <-ticker:
			out, _ := run(manageBDE, "-status", "C:")
			if strings.Contains(out, "Percentage Encrypted: 100") {
				log.Println("[INFO] Encryption complete")
				return nil
			}
		}
	}
}

func enableEncryption() error {
	_, err := run(manageBDE, "-on", "C:", "-RecoveryPassword")
	return err
}

func isBitLockerCLIExecutable() bool {
	_, err := run(manageBDE, "-status")
	return err == nil
}

//
// ---------------- POLICY ----------------
//

func enableNoTPMPolicy() {
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
// ---------------- EXEC HELPERS ----------------
//

func run(cmd string, args ...string) (string, error) {
	c := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	if err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func runWithFullLogging(cmd string, args ...string) error {
	log.Println("[CMD]", cmd, strings.Join(args, " "))
	c := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()

	log.Println("[CMD EXIT]", err)
	if stdout.Len() > 0 {
		log.Println("[STDOUT]", strings.TrimSpace(stdout.String()))
	}
	if stderr.Len() > 0 {
		log.Println("[STDERR]", strings.TrimSpace(stderr.String()))
	}
	return err
}
