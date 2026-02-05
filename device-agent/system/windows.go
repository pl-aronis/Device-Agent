package system

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// WindowsEdition represents the Windows edition type
type WindowsEdition string

const (
	Home       WindowsEdition = "Home"
	Pro        WindowsEdition = "Pro"
	Enterprise WindowsEdition = "Enterprise"
	Unknown    WindowsEdition = "Unknown"
)

// DetectWindowsEdition detects the current Windows edition
func DetectWindowsEdition() (WindowsEdition, error) {
	// Use WMI to get the OS caption which includes the edition
	cmd := exec.Command("wmic", "os", "get", "caption", "/value")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return Unknown, fmt.Errorf("failed to detect Windows edition: %w", err)
	}

	caption := string(output)
	log.Printf("[SYSTEM] Windows caption: %s", caption)

	// Check for edition in the output
	if strings.Contains(strings.ToLower(caption), "home") {
		return Home, nil
	}
	if strings.Contains(strings.ToLower(caption), "enterprise") {
		return Enterprise, nil
	}
	if strings.Contains(strings.ToLower(caption), "pro") {
		return Pro, nil
	}

	return Unknown, errors.New("unable to determine Windows edition")
}

// SupportsFeature checks if the Windows edition supports a given feature
func (w WindowsEdition) SupportsFeature(feature string) bool {
	switch feature {
	case "BitLocker":
		// BitLocker is available in Pro and Enterprise
		return w == Pro || w == Enterprise
	case "AppLocker":
		// AppLocker is available in Enterprise
		return w == Enterprise
	case "WDAC":
		// Windows Defender Application Control available in Enterprise
		return w == Enterprise
	default:
		return false
	}
}

// GetString returns the string representation of the Windows edition
func (w WindowsEdition) String() string {
	return string(w)
}
