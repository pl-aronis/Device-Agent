package enforcement

import (
	"io"
	"log"
	"os/exec"
	"regexp"

	"device-agent/heartbeat"
)

func ForceBitLockerRecovery() {
	log.Println("Initiating Secure Lockdown Sequence...")

	// 1. Generate new Recovery Password
	// manage-bde -protectors -add C: -rp
	cmd := exec.Command("manage-bde", "-protectors", "-add", "C:", "-rp")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Error adding protector:", err)
		// Proceeding might be dangerous if we can't generate a key,
		// but if we are here, IT wants it locked.
		// For safety in this demo, let's try to parse anyway or just log.
	}

	// 2. Extract password from output
	outStr := string(output)
	// Regex for 48-digit numerical password (approximate pattern)
	// Output usually contains: "Password: \n 123456-..."
	re := regexp.MustCompile(`(\d{6}-){7}\d{6}`)
	password := re.FindString(outStr)

	if password != "" {
		log.Printf("Generated Key (sending to backend): %s", password)
		// 3. Send to Backend
		err := heartbeat.SendRecoveryKey(password)
		if err != nil {
			log.Println("CRITICAL: Failed to send recovery key to backend:", err)
			log.Println("Aborting Lock for safety in Verified Mode.")
			// In production, you might lock anyway, but here we save the user.
			return
		} else {
			log.Println("Key successfully archived.")
		}
	} else {
		log.Println("Could not parse new recovery password. Aborting lock for safety.")
		return
	}

	// 1.5 Add Master Password Protector (Backdoor)
	log.Println("Adding Master Password protector...")
	masterCmd := exec.Command("manage-bde", "-protectors", "-add", "C:", "-pw")
	stdin, err := masterCmd.StdinPipe()
	if err == nil {
		go func() {
			defer stdin.Close()
			// Pipe the password twice (for confirmation if prompted, though -pw likely takes it via stdin or prompt)
			// Windows manage-bde -pw usually prompts: "Type the password:" then "Confirm the password:"
			io.WriteString(stdin, "MasterKey@123\nMasterKey@123\n")
		}()
		if err := masterCmd.Run(); err != nil {
			log.Println("Warning: Failed to add Master Password protector (might already exist or policy forbids):", err)
		} else {
			log.Println("Master Password 'MasterKey@123' added.")
		}
	} else {
		log.Println("Failed to attach stdin for Master Password:", err)
	}

	// 4. Force Recovery
	log.Println("Forcing BitLocker recovery...")
	exec.Command("manage-bde", "-forcerecovery", "C:").Run()

	// 5. Reboot
	log.Println("Rebooting...")
	exec.Command("shutdown", "/r", "/t", "0").Run()
}
