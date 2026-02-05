package system

import (
	"errors"
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
	// First try WMI (legacy, may be missing on newer Windows)
	cmd := exec.Command("wmic", "os", "get", "caption", "/value")
	if output, err := cmd.CombinedOutput(); err == nil {
		caption := string(output)
		log.Printf("[SYSTEM] Windows caption (wmic): %s", caption)
		if edition, ok := parseEditionFromString(caption); ok {
			return edition, nil
		}
	}

	// Fallback: read from registry via reg.exe (available on all Windows)
	if edition, ok := detectEditionFromRegistry(); ok {
		return edition, nil
	}

	return Unknown, errors.New("unable to determine Windows edition")
}

func detectEditionFromRegistry() (WindowsEdition, bool) {
	// Prefer EditionID (e.g., Professional, Enterprise, Core)
	cmd := exec.Command("reg", "query", `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "/v", "EditionID")
	if output, err := cmd.CombinedOutput(); err == nil {
		text := string(output)
		log.Printf("[SYSTEM] Windows EditionID (reg): %s", text)
		if edition, ok := parseEditionFromString(text); ok {
			return edition, true
		}
	}

	// Fallback to ProductName (e.g., Windows 11 Pro)
	cmd = exec.Command("reg", "query", `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "/v", "ProductName")
	if output, err := cmd.CombinedOutput(); err == nil {
		text := string(output)
		log.Printf("[SYSTEM] Windows ProductName (reg): %s", text)
		if edition, ok := parseEditionFromString(text); ok {
			return edition, true
		}
	}

	return Unknown, false
}

func parseEditionFromString(s string) (WindowsEdition, bool) {
	lower := strings.ToLower(s)

	// Map common edition identifiers
	switch {
	case strings.Contains(lower, "enterprise"):
		return Enterprise, true
	case strings.Contains(lower, "professional"),
		strings.Contains(lower, "pro"):
		return Pro, true
	case strings.Contains(lower, "home"),
		strings.Contains(lower, "core"), // EditionID for Home is often "Core"
		strings.Contains(lower, "homesinglelanguage"),
		strings.Contains(lower, "home single language"):
		return Home, true
	default:
		return Unknown, false
	}
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
