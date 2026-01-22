package signer

import (
	"encoding/json"
	"fmt"
)

// ParseJSONRPCTransaction parses JSON-RPC transaction parameters
//
// Supports two formats:
//
//	Array format: [{"from": "...", "to": "...", ...}]
//	Object format: {"from": "...", "to": "...", ...}
//
// This function is designed for eth_signTransaction and eth_sendTransaction methods.
func ParseJSONRPCTransaction(params json.RawMessage) (JSONRPCTransaction, error) {
	var tx JSONRPCTransaction

	// Handle array format [{"key": "value"}]
	var paramsArray []json.RawMessage
	if err := json.Unmarshal(params, &paramsArray); err == nil && len(paramsArray) > 0 {
		// Array format, take first element
		if err := json.Unmarshal(paramsArray[0], &tx); err != nil {
			return tx, fmt.Errorf("failed to parse transaction params: %w", err)
		}
	} else {
		// Direct object format
		if err := json.Unmarshal(params, &tx); err != nil {
			return tx, fmt.Errorf("failed to parse transaction params: %w", err)
		}
	}

	return tx, nil
}
