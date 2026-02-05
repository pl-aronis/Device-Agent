package credentials

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

const (
	CredentialsFile = "/var/lib/device-agent-linux/credentials.json"
	CredentialsLog  = "/var/lib/device-agent-linux/credentials.log"
)

// Credentials holds all important credentials for the device
type Credentials struct {
	DeviceID     string    `json:"device_id"`
	RecoveryKey  string    `json:"recovery_key"`
	BIOSPassword string    `json:"bios_password"`
	BackendIP    string    `json:"backend_ip"`
	BackendPort  string    `json:"backend_port"`
	MACAddress   string    `json:"mac_address"`
	Hostname     string    `json:"hostname"`
	OSDetails    string    `json:"os_details"`
	RegisteredAt time.Time `json:"registered_at"`
	LastUpdated  time.Time `json:"last_updated"`
}

// LogCredentials logs all credentials to both JSON file and human-readable log
func LogCredentials(creds *Credentials) error {
	log.Println("[CREDENTIALS] ========================================")
	log.Println("[CREDENTIALS] LOGGING DEVICE CREDENTIALS")
	log.Println("[CREDENTIALS] ========================================")

	// Ensure directory exists
	if err := os.MkdirAll("/var/lib/device-agent-linux", 0700); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Update timestamp
	creds.LastUpdated = time.Now()
	if creds.RegisteredAt.IsZero() {
		creds.RegisteredAt = time.Now()
	}

	// Save as JSON
	jsonData, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %v", err)
	}

	if err := os.WriteFile(CredentialsFile, jsonData, 0400); err != nil {
		return fmt.Errorf("failed to write credentials file: %v", err)
	}

	log.Printf("[CREDENTIALS] Credentials saved to: %s", CredentialsFile)

	// Create human-readable log
	logContent := fmt.Sprintf(`
========================================
DEVICE CREDENTIALS LOG
========================================
Generated: %s
Last Updated: %s

DEVICE INFORMATION:
-------------------
Device ID:       %s
MAC Address:     %s
Hostname:        %s
OS Details:      %s

BACKEND CONNECTION:
------------------
Backend IP:      %s
Backend Port:    %s
Backend URL:     http://%s:%s

SECURITY CREDENTIALS:
--------------------
Recovery Key:    %s
BIOS Password:   %s

IMPORTANT NOTES:
---------------
1. SAVE THESE CREDENTIALS SECURELY!
2. Recovery Key is needed to unlock LUKS encrypted partitions
3. BIOS Password is needed to access BIOS/UEFI settings
4. Without these credentials, data recovery may be impossible

FILE LOCATIONS:
--------------
Credentials JSON:  %s
Credentials Log:   %s
Device Info:       /var/lib/device-agent-linux/device.json
BIOS Password:     /var/lib/device-agent-linux/bios.pwd
LUKS Key:          /var/lib/device-agent-linux/luks.key

RECOVERY PROCEDURES:
-------------------
1. To unlock LUKS partition:
   echo '%s' | sudo cryptsetup open /dev/sdX unlocked_disk

2. To get BIOS password:
   sudo cat /var/lib/device-agent-linux/bios.pwd

3. To retrieve recovery key from backend:
   curl "http://%s:%s/admin/set?id=%s&status=ACTIVE"

========================================
END OF CREDENTIALS LOG
========================================
`,
		creds.RegisteredAt.Format(time.RFC3339),
		creds.LastUpdated.Format(time.RFC3339),
		creds.DeviceID,
		creds.MACAddress,
		creds.Hostname,
		creds.OSDetails,
		creds.BackendIP,
		creds.BackendPort,
		creds.BackendIP, creds.BackendPort,
		creds.RecoveryKey,
		creds.BIOSPassword,
		CredentialsFile,
		CredentialsLog,
		creds.RecoveryKey,
		creds.BackendIP, creds.BackendPort, creds.DeviceID,
	)

	if err := os.WriteFile(CredentialsLog, []byte(logContent), 0400); err != nil {
		return fmt.Errorf("failed to write credentials log: %v", err)
	}

	log.Printf("[CREDENTIALS] Human-readable log saved to: %s", CredentialsLog)

	// Log to console
	log.Println("[CREDENTIALS] ========================================")
	log.Println("[CREDENTIALS] DEVICE CREDENTIALS")
	log.Println("[CREDENTIALS] ========================================")
	log.Printf("[CREDENTIALS] Device ID:       %s", creds.DeviceID)
	log.Printf("[CREDENTIALS] MAC Address:     %s", creds.MACAddress)
	log.Printf("[CREDENTIALS] Hostname:        %s", creds.Hostname)
	log.Printf("[CREDENTIALS] Backend:         http://%s:%s", creds.BackendIP, creds.BackendPort)
	log.Println("[CREDENTIALS] ========================================")
	log.Printf("[CREDENTIALS] Recovery Key:    %s", creds.RecoveryKey)
	log.Printf("[CREDENTIALS] BIOS Password:   %s", creds.BIOSPassword)
	log.Println("[CREDENTIALS] ========================================")
	log.Println("[CREDENTIALS] ⚠️  SAVE THESE CREDENTIALS SECURELY!")
	log.Println("[CREDENTIALS] ========================================")
	log.Printf("[CREDENTIALS] Credentials file: %s", CredentialsFile)
	log.Printf("[CREDENTIALS] Readable log:     %s", CredentialsLog)
	log.Println("[CREDENTIALS] ========================================")

	return nil
}

// LoadCredentials loads credentials from file
func LoadCredentials() (*Credentials, error) {
	data, err := os.ReadFile(CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %v", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %v", err)
	}

	return &creds, nil
}

// PrintCredentialsSummary prints a summary of credentials to console
func PrintCredentialsSummary() {
	creds, err := LoadCredentials()
	if err != nil {
		log.Printf("[CREDENTIALS] Failed to load credentials: %v", err)
		return
	}

	log.Println("\n========================================")
	log.Println("DEVICE CREDENTIALS SUMMARY")
	log.Println("========================================")
	log.Printf("Device ID:       %s", creds.DeviceID)
	log.Printf("Recovery Key:    %s", creds.RecoveryKey)
	log.Printf("BIOS Password:   %s", creds.BIOSPassword)
	log.Printf("Backend:         http://%s:%s", creds.BackendIP, creds.BackendPort)
	log.Println("========================================")
	log.Printf("Full details: %s", CredentialsLog)
	log.Println("========================================\n")
}
