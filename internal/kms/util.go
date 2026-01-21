package kms

import "bytes"

func trimLeadingZeros(b []byte) []byte {
	return bytes.TrimLeft(b, "\x00")
}
