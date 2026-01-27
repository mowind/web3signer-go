// Package utils provides common utility functions for internal packages.
//
// This package contains shared functionality that is used across
// multiple internal modules, including validation and HTTP utilities.
package utils

import (
	"strings"
)

// IsValidEthAddress validates an Ethereum address format.
//
// An Ethereum address must:
// - Not be empty
// - Have "0x" prefix
// - Be exactly 42 characters long (0x + 40 hex characters)
// - Contain only hexadecimal digits (0-9, a-f, A-F) after prefix
//
// Note: EIP-55 checksum validation is handled automatically when addresses are
// converted to ethgo.Address type. This function only validates the basic format.
//
// Parameters:
//   - addr: The address string to validate
//
// Returns:
//   - bool: true if address is valid, false otherwise
//
// Example:
//
//	.IsValidEthAddress("0x1234567890123456789012345678901234567890") // true
//	.IsValidEthAddress("0x123456789012345678901234567890123456789")  // false (too short)
//	.IsValidEthAddress("1234567890123456789012345678901234567890")  // false (no 0x prefix)
func IsValidEthAddress(addr string) bool {
	if addr == "" {
		return false
	}

	if !strings.HasPrefix(addr, "0x") {
		return false
	}

	if len(addr) != 42 {
		return false
	}

	for _, c := range addr[2:] {
		if !isHexDigit(c) {
			return false
		}
	}

	return true
}

// isHexDigit checks if a rune is a valid hexadecimal digit (0-9, a-f, A-F).
//
// Parameters:
//   - c: The rune to check
//
// Returns:
//   - bool: true if rune is a valid hex digit, false otherwise
func isHexDigit(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
