package enforcement

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	LUKSKeyFile      = "/var/lib/device-agent-linux/luks.key"
	BIOSPasswordFile = "/var/lib/device-agent-linux/bios.pwd"
)

// LockDevice performs comprehensive device locking
func LockDevice() {
	log.Println("[LOCK] Locking device")

	// Lock the screen
	lockScreen()

	// Restrict network access
	restrictNetwork()

	// Lock LUKS encrypted partitions
	lockLUKSPartitions()

	// Disable user accounts (except root)
	disableUserAccounts()
}

// lockScreen locks the current user session
func lockScreen() {
	cmd := exec.Command("loginctl", "lock-session")
	err := cmd.Run()
	if err != nil {
		log.Println("[LOCK] Failed to lock screen:", err)
	}
}

// restrictNetwork disables network connectivity
func restrictNetwork() {
	cmd := exec.Command("nmcli", "networking", "off")
	err := cmd.Run()
	if err != nil {
		log.Println("[LOCK] Failed to restrict network:", err)
	}
}

// lockLUKSPartitions closes all LUKS encrypted partitions
func lockLUKSPartitions() {
	log.Println("[LOCK] Closing LUKS encrypted partitions...")

	// Find all LUKS devices
	cmd := exec.Command("lsblk", "-o", "NAME,TYPE", "-n")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[LOCK] Failed to list block devices: %v", err)
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "crypt") {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				deviceName := fields[0]
				// Close the LUKS device
				closeCmd := exec.Command("cryptsetup", "close", deviceName)
				if err := closeCmd.Run(); err != nil {
					log.Printf("[LOCK] Failed to close LUKS device %s: %v", deviceName, err)
				} else {
					log.Printf("[LOCK] Successfully closed LUKS device: %s", deviceName)
				}
			}
		}
	}
}

// disableUserAccounts locks all non-root user accounts
func disableUserAccounts() {
	log.Println("[LOCK] Disabling user accounts...")

	// Get list of users
	cmd := exec.Command("awk", "-F:", "$3 >= 1000 {print $1}", "/etc/passwd")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[LOCK] Failed to get user list: %v", err)
		return
	}

	users := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, user := range users {
		if user != "" && user != "nobody" {
			lockCmd := exec.Command("usermod", "-L", user)
			if err := lockCmd.Run(); err != nil {
				log.Printf("[LOCK] Failed to lock user %s: %v", user, err)
			} else {
				log.Printf("[LOCK] Successfully locked user: %s", user)
			}
		}
	}
}

// SetupLUKSEncryption sets up LUKS encryption on a specified partition
// This should be run during initial device setup
func SetupLUKSEncryption(partition string, recoveryKey string) error {
	log.Printf("[LUKS] Setting up LUKS encryption on %s", partition)

	// Generate a random key file
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate key: %v", err)
	}

	// Save key to file
	if err := os.MkdirAll("/var/lib/device-agent-linux", 0700); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(LUKSKeyFile, key, 0400); err != nil {
		return fmt.Errorf("failed to write key file: %v", err)
	}

	// Format partition with LUKS
	// WARNING: This will destroy all data on the partition
	cmd := exec.Command("cryptsetup", "luksFormat", partition, LUKSKeyFile)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to format LUKS partition: %v", err)
	}

	// Add recovery key as additional passphrase
	addKeyCmd := exec.Command("bash", "-c",
		fmt.Sprintf("echo '%s' | cryptsetup luksAddKey %s --key-file=%s",
			recoveryKey, partition, LUKSKeyFile))
	if err := addKeyCmd.Run(); err != nil {
		log.Printf("[LUKS] Warning: Failed to add recovery key: %v", err)
	}

	log.Printf("[LUKS] Successfully set up LUKS encryption on %s", partition)
	return nil
}

// UnlockLUKSPartition unlocks a LUKS partition using the recovery key
func UnlockLUKSPartition(partition string, recoveryKey string) error {
	log.Printf("[LUKS] Unlocking partition %s with recovery key", partition)

	// Derive device name from partition
	deviceName := strings.TrimPrefix(partition, "/dev/")
	deviceName = strings.ReplaceAll(deviceName, "/", "_")
	mappedName := fmt.Sprintf("unlocked_%s", deviceName)

	// Unlock using recovery key
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf("echo '%s' | cryptsetup open %s %s",
			recoveryKey, partition, mappedName))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unlock LUKS partition: %v", err)
	}

	log.Printf("[LUKS] Successfully unlocked partition: /dev/mapper/%s", mappedName)
	return nil
}

// DetectVendor detects the system manufacturer
func DetectVendor() string {
	// Try to read from DMI
	cmd := exec.Command("cat", "/sys/class/dmi/id/sys_vendor")
	out, err := cmd.Output()
	if err == nil {
		vendor := strings.ToLower(strings.TrimSpace(string(out)))
		log.Printf("[BIOS] Detected vendor from DMI: %s", vendor)
		return vendor
	}

	// Fallback: try dmidecode
	cmd = exec.Command("dmidecode", "-s", "system-manufacturer")
	out, err = cmd.Output()
	if err == nil {
		vendor := strings.ToLower(strings.TrimSpace(string(out)))
		log.Printf("[BIOS] Detected vendor from dmidecode: %s", vendor)
		return vendor
	}

	log.Println("[BIOS] Could not detect vendor, returning 'unknown'")
	return "unknown"
}

// SetBIOSPassword sets a BIOS/UEFI password (vendor-specific)
func SetBIOSPassword(password string) error {
	log.Println("[BIOS] ========================================")
	log.Println("[BIOS] Setting BIOS/UEFI password...")
	log.Println("[BIOS] ========================================")

	// Save password locally for reference
	if err := os.MkdirAll("/var/lib/device-agent-linux", 0700); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(BIOSPasswordFile, []byte(password), 0400); err != nil {
		return fmt.Errorf("failed to save BIOS password: %v", err)
	}

	log.Printf("[BIOS] Password saved to: %s", BIOSPasswordFile)
	log.Printf("[BIOS] BIOS Password: %s", password)
	log.Println("[BIOS] ⚠️  SAVE THIS PASSWORD SECURELY!")
	log.Println("[BIOS] ========================================")

	// Detect vendor
	vendor := DetectVendor()
	log.Printf("[BIOS] System Vendor: %s", vendor)

	// Try vendor-specific tools
	success := false

	// Dell Systems
	if strings.Contains(vendor, "dell") {
		log.Println("[BIOS] Dell system detected, attempting CCTK...")
		if _, err := exec.LookPath("cctk"); err == nil {
			cmd := exec.Command("cctk", "--setuppwd="+password)
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("[BIOS] Dell CCTK failed: %v", err)
				log.Printf("[BIOS] CCTK output: %s", string(output))
			} else {
				log.Println("[BIOS] ✅ Successfully set Dell BIOS password using CCTK")
				log.Printf("[BIOS] CCTK output: %s", string(output))
				success = true
			}
		} else {
			log.Println("[BIOS] Dell CCTK not found. Install from: https://www.dell.com/support/kbdoc/en-us/000178000")
		}
	}

	// HP Systems
	if strings.Contains(vendor, "hp") || strings.Contains(vendor, "hewlett") {
		log.Println("[BIOS] HP system detected, attempting HP tools...")

		// Try HP BIOS Configuration Utility (BCU)
		if _, err := exec.LookPath("hpsetup"); err == nil {
			cmd := exec.Command("hpsetup", "-s", "-a", "SetupPassword="+password)
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("[BIOS] HP Setup failed: %v", err)
				log.Printf("[BIOS] HP Setup output: %s", string(output))
			} else {
				log.Println("[BIOS] ✅ Successfully set HP BIOS password using hpsetup")
				log.Printf("[BIOS] HP Setup output: %s", string(output))
				success = true
			}
		} else {
			log.Println("[BIOS] HP Setup utility not found. Install HP BIOS Configuration Utility (BCU)")
		}

		// Try alternative HP tool
		if !success {
			if _, err := exec.LookPath("conrep"); err == nil {
				log.Println("[BIOS] Trying HP ConRep tool...")
				// ConRep requires a config file, so we'll just log it
				log.Println("[BIOS] HP ConRep found but requires manual configuration")
			}
		}
	}

	// Lenovo Systems
	if strings.Contains(vendor, "lenovo") {
		log.Println("[BIOS] Lenovo system detected, attempting Lenovo tools...")

		// Try Lenovo BIOS Setup utility
		if _, err := exec.LookPath("thinkvantage"); err == nil {
			cmd := exec.Command("thinkvantage", "--set-bios-password", password)
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("[BIOS] Lenovo ThinkVantage failed: %v", err)
				log.Printf("[BIOS] ThinkVantage output: %s", string(output))
			} else {
				log.Println("[BIOS] ✅ Successfully set Lenovo BIOS password using ThinkVantage")
				log.Printf("[BIOS] ThinkVantage output: %s", string(output))
				success = true
			}
		} else {
			log.Println("[BIOS] Lenovo ThinkVantage not found")
		}

		// Try Lenovo System Update
		if !success {
			if _, err := exec.LookPath("lenovo-bios-password"); err == nil {
				cmd := exec.Command("lenovo-bios-password", "--set", password)
				output, err := cmd.CombinedOutput()
				if err != nil {
					log.Printf("[BIOS] Lenovo BIOS password tool failed: %v", err)
					log.Printf("[BIOS] Output: %s", string(output))
				} else {
					log.Println("[BIOS] ✅ Successfully set Lenovo BIOS password")
					log.Printf("[BIOS] Output: %s", string(output))
					success = true
				}
			} else {
				log.Println("[BIOS] Lenovo BIOS password tool not found")
			}
		}
	}

	// If automatic setting failed, provide manual instructions
	if !success {
		log.Println("[BIOS] ========================================")
		log.Println("[BIOS] ⚠️  MANUAL BIOS PASSWORD SETUP REQUIRED")
		log.Println("[BIOS] ========================================")
		log.Printf("[BIOS] Vendor: %s", vendor)
		log.Printf("[BIOS] Password: %s", password)
		log.Println("[BIOS] ")
		log.Println("[BIOS] Automatic BIOS password setting failed or not supported.")
		log.Println("[BIOS] Please manually set the BIOS password:")
		log.Println("[BIOS] ")
		log.Println("[BIOS] Steps:")
		log.Println("[BIOS]   1. Reboot the system")
		log.Println("[BIOS]   2. Press BIOS key during boot:")
		log.Println("[BIOS]      - Dell: F2")
		log.Println("[BIOS]      - HP: F10 or ESC")
		log.Println("[BIOS]      - Lenovo: F1 or F2")
		log.Println("[BIOS]      - Others: DEL, F2, or ESC")
		log.Println("[BIOS]   3. Navigate to Security settings")
		log.Println("[BIOS]   4. Set Supervisor/Administrator Password")
		log.Printf("[BIOS]   5. Enter password: %s", password)
		log.Println("[BIOS]   6. Disable USB boot and PXE boot")
		log.Println("[BIOS]   7. Enable Secure Boot (if available)")
		log.Println("[BIOS]   8. Save and exit (usually F10)")
		log.Println("[BIOS] ========================================")

		// Save manual instructions to file
		instructionsFile := "/var/lib/device-agent-linux/bios-manual-setup.txt"
		instructions := fmt.Sprintf(`BIOS Password Manual Setup Instructions

Vendor: %s
BIOS Password: %s

Steps:
1. Reboot the system
2. Press BIOS key during boot (F2, F10, F1, DEL, or ESC)
3. Navigate to Security settings
4. Set Supervisor/Administrator Password to: %s
5. Disable USB boot and PXE boot
6. Enable Secure Boot if available
7. Save and exit (F10)

Password file location: %s
`, vendor, password, password, BIOSPasswordFile)

		if err := os.WriteFile(instructionsFile, []byte(instructions), 0400); err != nil {
			log.Printf("[BIOS] Failed to save manual instructions: %v", err)
		} else {
			log.Printf("[BIOS] Manual instructions saved to: %s", instructionsFile)
		}
	}

	log.Println("[BIOS] ========================================")
	return nil
}

// GenerateBIOSPassword generates a secure BIOS password
func GenerateBIOSPassword() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GetBIOSPassword retrieves the stored BIOS password
func GetBIOSPassword() (string, error) {
	data, err := os.ReadFile(BIOSPasswordFile)
	if err != nil {
		return "", fmt.Errorf("BIOS password not found: %v", err)
	}
	return string(data), nil
}
