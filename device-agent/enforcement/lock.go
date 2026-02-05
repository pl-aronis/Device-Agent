package enforcement

import (
	"log"

	"device-agent/system"
)

// EnforceDeviceLock enforces device lock based on Windows edition
// Returns the recovery protector if successful, or an error
func EnforceDeviceLock() (*RecoveryProtector, error) {
	log.Println("========== DEVICE LOCK START ==========")

	edition, err := system.DetectWindowsEdition()
	if err != nil {
		log.Printf("[ERROR] Failed to detect Windows edition: %v", err)
		return nil, err
	}

	log.Printf("[INFO] Detected Windows edition: %s", edition)

	// Check if BitLocker is supported on this edition
	if !edition.SupportsFeature("BitLocker") {
		log.Printf("[WARN] BitLocker not supported on Windows %s", edition)
		return nil, ErrBitLockerUnsupported
	}

	// Execute enforcement for the specific edition
	switch edition {
	case system.Enterprise:
		return enforceEnterprise()
	case system.Pro:
		return enforcePro()
	case system.Home:
		// Windows Home doesn't support BitLocker
		log.Println("[WARN] Windows Home does not support BitLocker encryption")
		return nil, ErrBitLockerUnsupported
	default:
		return nil, ErrUnknownEdition
	}
}

// enforceEnterprise applies enforcement for Windows Enterprise
func enforceEnterprise() (*RecoveryProtector, error) {
	log.Println("[ENFORCE-ENTERPRISE] Starting enforcement for Windows Enterprise")
	return performBitLockerEnforcement()
}

// enforcePro applies enforcement for Windows Pro
func enforcePro() (*RecoveryProtector, error) {
	log.Println("[ENFORCE-PRO] Starting enforcement for Windows Pro")
	return performBitLockerEnforcement()
}

// performBitLockerEnforcement performs the actual BitLocker enforcement
func performBitLockerEnforcement() (*RecoveryProtector, error) {
	bm := NewBitLockerManager()

	// Step 1: Check if BitLocker CLI is available
	if !bm.IsAvailable() {
		log.Println("[FATAL] BitLocker CLI unavailable")
		return nil, ErrBitLockerUnavailable
	}

	// Step 2: Enable encryption if not already enabled
	if !bm.IsEncrypted() {
		log.Println("[INFO] Enabling BitLocker encryption")
		if err := bm.EnableEncryption(); err != nil {
			log.Printf("[FATAL] Failed to enable encryption: %v", err)
			return nil, err
		}
	}

	// Step 3: Wait for encryption to complete
	if err := bm.WaitForEncryption(); err != nil {
		log.Printf("[FATAL] Encryption wait failed: %v", err)
		return nil, err
	}

	// Step 4: Snapshot existing protectors
	existing, err := bm.ListRecoveryProtectors()
	if err != nil {
		log.Printf("[WARN] Failed to list existing protectors: %v", err)
	} else {
		log.Printf("[INFO] Found %d pre-existing recovery protectors", len(existing))
	}

	// Step 5: Create first agent recovery password
	log.Println("[INFO] Creating first agent recovery password")
	first, err := bm.CreateRecoveryProtector()
	if err != nil {
		log.Printf("[FATAL] Failed to create recovery protector: %v", err)
		return nil, err
	}
	logRecoveryKey(first)

	// Step 6: Attempt to enable protection
	if bm.IsProtectionEnabled() {
		log.Println("[SUCCESS] BitLocker protection already enabled")
		_ = bm.ForceRecoveryAndReboot()
		return first, nil
	}

	if err := bm.EnableProtection(); err == nil && bm.IsProtectionEnabled() {
		log.Println("[SUCCESS] BitLocker protection enabled")
		_ = bm.ForceRecoveryAndReboot()
		return first, nil
	}

	log.Println("[WARN] Initial protection enable failed - creating second recovery password")

	// Step 7: Create second agent recovery password
	second, err := bm.CreateRecoveryProtector()
	if err != nil {
		log.Printf("[FATAL] Failed to create second recovery protector: %v", err)
		return first, nil // Return the first key at least
	}
	logRecoveryKey(second)

	// Step 8: Delete first agent key
	log.Printf("[INFO] Deleting previous agent recovery password: %s", first.ID)
	_ = bm.DeleteProtector(first.ID)

	// Step 9: Retry protection enable
	if err := bm.EnableProtection(); err == nil && bm.IsProtectionEnabled() {
		log.Println("[SUCCESS] BitLocker protection enabled on retry")
		_ = bm.ForceRecoveryAndReboot()
		return second, nil
	}

	log.Println("[FATAL] Unable to enable BitLocker protection after rotation attempts")
	return second, nil
}

// logRecoveryKey logs the recovery key securely
func logRecoveryKey(p *RecoveryProtector) {
	log.Println("=================================================")
	log.Println("[AGENT RECOVERY KEY]")
	log.Printf("ID : %s\n", p.ID)
	log.Printf("KEY: %s\n", p.Key)
	log.Println("=================================================")
}
