package enforcement

import "strings"

var (
	ErrNotAdmin       = errorString("must run as administrator")
	ErrCLIUnavailable = errorString("bitlocker CLI not found")
)

type errorString string

func (e errorString) Error() string { return string(e) }

func (b *Bitlocker) isAdmin() bool {
	_, err := b.exec.Run("net", "session")
	return err == nil
}

func (b *Bitlocker) isBitLockerCLIExecutable() bool {
	_, err := b.exec.Run("where", manageBDE)
	return err == nil
}

func (b *Bitlocker) isEncrypted() bool {
	out, _ := b.exec.Run(manageBDE, "-status", "C:")
	return !strings.Contains(out, "Fully Decrypted")
}

func (b *Bitlocker) isProtectionOn() bool {
	out, _ := b.exec.Run(manageBDE, "-status", "C:")
	n := strings.ToLower(out)
	n = strings.ReplaceAll(n, "\t", " ")
	n = strings.Join(strings.Fields(n), " ")
	return strings.Contains(n, "protection status: protection on")
}
