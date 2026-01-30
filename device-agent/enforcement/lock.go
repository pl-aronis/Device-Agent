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
BITLOCKER ENFORCEMENT — NON-DESTRUCTIVE, AGENT-OWNED ROTATION

RULES:
- DO NOT touch pre-existing recovery passwords
- ONLY manage keys created by this agent
- NEVER delete first
- Create → try lock → if fail → create another → delete agent-old → retry
*/

const manageBDE = `C:\Windows\Sysnative\manage-bde.exe`

type RecoveryProtector struct {
	ID  string
	Key string
}

func EnforceDeviceLock() {
	log.Println("========== DEVICE LOCK START ==========")

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

	// Step 1 — snapshot existing protectors
	existing := listRecoveryProtectorIDs()
	log.Printf("[INFO] Found %d pre-existing recovery protectors\n", len(existing))

	// Step 2 — create first agent key
	log.Println("[INFO] Creating first agent recovery password")
	first, err := createRecoveryProtector()
	if err != nil {
		log.Println("[FATAL]", err)
		return
	}

	logRecoveryKey(first)

	// Step 3 — attempt enable
	if tryEnableProtection() {
		forceRecoveryAndReboot()
		return
	}

	log.Println("[WARN] Enable failed — creating second recovery password")

	// Step 4 — create second agent key
	second, err := createRecoveryProtector()
	if err != nil {
		log.Println("[FATAL]", err)
		return
	}

	logRecoveryKey(second)

	// Step 5 — delete first agent key ONLY
	log.Println("[INFO] Deleting previous agent recovery password:", first.ID)
	deleteProtector(first.ID)

	// Step 6 — retry enable
	if tryEnableProtection() {
		forceRecoveryAndReboot()
		return
	}

	log.Println("[FATAL] Unable to enable BitLocker protection after rotation attempts")
}

//
// ---------------- CORE ACTIONS ----------------
//

func tryEnableProtection() bool {
	log.Println("[ACTION] Attempting to enable BitLocker protection")
	runWithFullLogging(manageBDE, "-protectors", "-enable", "C:")

	time.Sleep(3 * time.Second)

	out, err := run(manageBDE, "-status", "C:")
	if err != nil {
		return false
	}

	n := normalize(out)
	return strings.Contains(n, "protection status: protection on")
}

func forceRecoveryAndReboot() {
	log.Println("[SUCCESS] Protection ON — forcing recovery")
	time.Sleep(5 * time.Minute)
	run(manageBDE, "-forcerecovery", "C:")
	run("shutdown", "/r", "/t", "0")
}

//
// ---------------- RECOVERY PASSWORD MANAGEMENT ----------------
//

func createRecoveryProtector() (*RecoveryProtector, error) {
	out, err := run(manageBDE, "-protectors", "-add", "C:", "-RecoveryPassword")
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

	return &RecoveryProtector{
		ID:  id,
		Key: key,
	}, nil
}

func deleteProtector(id string) {
	run(manageBDE, "-protectors", "-delete", "C:", "-id", id)
}

func listRecoveryProtectorIDs() map[string]bool {
	out, _ := run(manageBDE, "-protectors", "-get", "C:")
	idRe := regexp.MustCompile(`ID:\s*{[^}]+}`)

	ids := make(map[string]bool)
	for _, m := range idRe.FindAllString(out, -1) {
		ids[strings.TrimPrefix(m, "ID: ")] = true
	}
	return ids
}

//
// ---------------- UTIL ----------------
//

func logRecoveryKey(p *RecoveryProtector) {
	log.Println("=================================================")
	log.Println("[AGENT RECOVERY KEY]")
	log.Println("ID :", p.ID)
	log.Println("KEY:", p.Key)
	log.Println("=================================================")
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "\t", " ")
	return strings.Join(strings.Fields(s), " ")
}

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

func runWithFullLogging(cmd string, args ...string) {
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
}
