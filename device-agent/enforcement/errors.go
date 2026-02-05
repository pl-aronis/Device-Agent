package enforcement

import "errors"

var (
	// ErrBitLockerUnsupported indicates BitLocker is not supported on this Windows edition
	ErrBitLockerUnsupported = errors.New("bitlocker is not supported on this Windows edition")

	// ErrBitLockerUnavailable indicates BitLocker CLI tools are not available
	ErrBitLockerUnavailable = errors.New("bitlocker cli tools are unavailable")

	// ErrUnknownEdition indicates the Windows edition could not be determined
	ErrUnknownEdition = errors.New("unknown Windows edition")
)
