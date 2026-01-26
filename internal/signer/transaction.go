package signer

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/umbracle/ethgo"
	"github.com/valyala/fastjson"
)

// JSONRPCTransaction is a wrapper around ethgo.Transaction
// designed to parse JSON-RPC parameters for eth_signTransaction and eth_sendTransaction
//
// Key differences from ethgo.Transaction.UnmarshalJSON:
// - Does not require hash, from, v, r, s fields (these are generated during signing)
// - Accepts optional fields gracefully
// - Handles string-formatted numeric fields (0x prefix)
type JSONRPCTransaction struct {
	ethgo.Transaction
}

var defaultPool fastjson.ParserPool

// UnmarshalJSON implements json.Unmarshaler for JSON-RPC transaction parameters
func (jt *JSONRPCTransaction) UnmarshalJSON(data []byte) error {
	p := defaultPool.Get()
	defer defaultPool.Put(p)

	v, err := p.Parse(string(data))
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return jt.unmarshalJSON(v)
}

func (jt *JSONRPCTransaction) unmarshalJSON(v *fastjson.Value) error {
	var err error

	// Parse required field: gas
	if jt.Gas, err = decodeUint(v, "gas"); err != nil {
		return fmt.Errorf("failed to decode gas: %w", err)
	}

	// Parse input/data field (optional)
	if jt.Input, err = decodeBytes(jt.Input[:0], v, "data"); err != nil {
		return fmt.Errorf("failed to decode data: %w", err)
	}

	// Parse optional fields
	if jt.Value, err = decodeBigIntOptional(v, "value"); err != nil {
		return fmt.Errorf("failed to decode value: %w", err)
	}

	if jt.Nonce, err = decodeUintOptional(v, "nonce"); err != nil {
		return fmt.Errorf("failed to decode nonce: %w", err)
	}

	// Parse to field (optional, can be null for contract creation)
	if isKeySet(v, "to") {
		if v.Get("to").String() != "null" {
			var to ethgo.Address
			if err := decodeAddr(&to, v, "to"); err != nil {
				return fmt.Errorf("failed to decode to: %w", err)
			}
			jt.To = &to
		}
	}

	// Parse from field (optional, used for address validation)
	if isKeySet(v, "from") {
		if err := decodeAddr(&jt.From, v, "from"); err != nil {
			return fmt.Errorf("failed to decode from: %w", err)
		}
	}

	// Determine transaction type based on fields
	// Check for EIP-1559 (Type 2) fields first
	//nolint:gocritic // if-else chain is appropriate here as we check different fields in priority order
	if isKeySet(v, "maxFeePerGas") || isKeySet(v, "maxPriorityFeePerGas") {
		jt.Type = ethgo.TransactionDynamicFee
		if jt.MaxPriorityFeePerGas, err = decodeBigIntOptional(v, "maxPriorityFeePerGas"); err != nil {
			return fmt.Errorf("failed to decode maxPriorityFeePerGas: %w", err)
		}
		if jt.MaxFeePerGas, err = decodeBigIntOptional(v, "maxFeePerGas"); err != nil {
			return fmt.Errorf("failed to decode maxFeePerGas: %w", err)
		}
		if jt.ChainID, err = decodeBigIntOptional(v, "chainId"); err != nil {
			return fmt.Errorf("failed to decode chainId: %w", err)
		}
	} else if isKeySet(v, "accessList") {
		// Check for EIP-2930 (Type 1) - has accessList
		jt.Type = ethgo.TransactionAccessList
		if jt.GasPrice, err = decodeUintOptional(v, "gasPrice"); err != nil {
			return fmt.Errorf("failed to decode gasPrice: %w", err)
		}
		if jt.ChainID, err = decodeBigIntOptional(v, "chainId"); err != nil {
			return fmt.Errorf("failed to decode chainId: %w", err)
		}
	} else {
		// Legacy transaction (Type 0)
		jt.Type = ethgo.TransactionLegacy
		if jt.GasPrice, err = decodeUintOptional(v, "gasPrice"); err != nil {
			return fmt.Errorf("failed to decode gasPrice: %w", err)
		}
	}

	// Parse accessList if present (for EIP-2930 and EIP-1559)
	if isKeySet(v, "accessList") {
		if err := unmarshalAccessList(&jt.AccessList, v.Get("accessList")); err != nil {
			return fmt.Errorf("failed to decode accessList: %w", err)
		}
	}

	return nil
}

// unmarshalAccessList decodes an access list from JSON
func unmarshalAccessList(al *ethgo.AccessList, v *fastjson.Value) error {
	elems, err := v.Array()
	if err != nil {
		return err
	}
	for _, elem := range elems {
		entry := ethgo.AccessEntry{}
		if err = decodeAddr(&entry.Address, elem, "address"); err != nil {
			return err
		}
		storage, err := elem.Get("storageKeys").Array()
		if err != nil {
			return err
		}

		entry.Storage = make([]ethgo.Hash, len(storage))
		for indx, stg := range storage {
			b, err := stg.StringBytes()
			if err != nil {
				return err
			}
			if err := entry.Storage[indx].UnmarshalText(b); err != nil {
				return err
			}
		}
		*al = append(*al, entry)
	}
	return nil
}

// isKeySet checks if a key exists and is not null
func isKeySet(v *fastjson.Value, key string) bool {
	value := v.Get(key)
	return value != nil && value.Type() != fastjson.TypeNull
}

// decodeBigIntOptional decodes a big.Int field if present
func decodeBigIntOptional(v *fastjson.Value, key string) (*big.Int, error) {
	if !isKeySet(v, key) {
		return nil, nil
	}
	return decodeBigInt(nil, v, key)
}

// decodeBigInt decodes a big.Int field
// Requires hex format with 0x prefix (per Ethereum JSON-RPC spec)
func decodeBigInt(b *big.Int, v *fastjson.Value, key string) (*big.Int, error) {
	vv := v.Get(key)
	if vv == nil {
		return nil, fmt.Errorf("field '%s' not found", key)
	}
	str := vv.String()
	str = strings.Trim(str, "\"")

	if !strings.HasPrefix(str, "0x") {
		return nil, fmt.Errorf("field '%s' does not have 0x prefix: '%s'", key, str)
	}

	if b == nil {
		b = new(big.Int)
	}

	hexStr := str[2:]
	if hexStr == "" {
		hexStr = "0"
	}

	_, ok := b.SetString(hexStr, 16)
	if !ok {
		return nil, fmt.Errorf("field '%s' failed to decode big int: '%s'", key, str)
	}

	return b, nil
}

// decodeUintOptional decodes a uint64 field if present
func decodeUintOptional(v *fastjson.Value, key string) (uint64, error) {
	if !isKeySet(v, key) {
		return 0, nil
	}
	return decodeUint(v, key)
}

// decodeUint decodes a uint64 field
// Requires hex format with 0x prefix (per Ethereum JSON-RPC spec)
func decodeUint(v *fastjson.Value, key string) (uint64, error) {
	vv := v.Get(key)
	if vv == nil {
		return 0, fmt.Errorf("field '%s' not found", key)
	}
	str := vv.String()
	str = strings.Trim(str, "\"")

	if !strings.HasPrefix(str, "0x") {
		return 0, fmt.Errorf("field '%s' does not have 0x prefix: '%s'", key, str)
	}

	hexStr := str[2:]
	if hexStr == "" {
		hexStr = "0"
	}

	num, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("field '%s' failed to decode uint: %s", key, str)
	}

	return num, nil
}

// decodeBytes decodes a bytes field (hex string)
func decodeBytes(dst []byte, v *fastjson.Value, key string) ([]byte, error) {
	if !isKeySet(v, key) {
		return nil, nil
	}

	vv := v.Get(key)
	str := vv.String()
	str = strings.Trim(str, "\"")

	if !strings.HasPrefix(str, "0x") {
		return nil, fmt.Errorf("field '%s' does not have 0x prefix: '%s'", key, str)
	}
	str = str[2:]
	if len(str)%2 != 0 {
		str = "0" + str
	}
	buf, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	dst = append(dst, buf...)
	return dst, nil
}

// isValidEthAddress validates an Ethereum address format
// Returns true if the address is valid, false otherwise
// Validation criteria:
// - Not empty
// - Starts with "0x" prefix
// - Exactly 42 characters long (including "0x")
// - All characters after "0x" are valid hexadecimal digits (0-9, a-f, A-F)
func isValidEthAddress(addr string) bool {
	// Check if string is empty
	if addr == "" {
		return false
	}

	// Check if starts with "0x" prefix
	if !strings.HasPrefix(addr, "0x") {
		return false
	}

	// Check if length is exactly 42
	if len(addr) != 42 {
		return false
	}

	// Check if all characters after "0x" are valid hex digits
	for _, c := range addr[2:] {
		if !isHexDigit(c) {
			return false
		}
	}

	return true
}

// isHexDigit checks if a character is a valid hexadecimal digit (0-9, a-f, A-F)
func isHexDigit(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// decodeAddr decodes an Address field
func decodeAddr(a *ethgo.Address, v *fastjson.Value, key string) error {
	b := v.GetStringBytes(key)
	if len(b) == 0 {
		return fmt.Errorf("field '%s' not found", key)
	}

	addrStr := string(b)

	// Validate address format before unmarshalling
	if !isValidEthAddress(addrStr) {
		return fmt.Errorf("field '%s' has invalid address format: '%s'", key, addrStr)
	}

	if err := a.UnmarshalText(b); err != nil {
		return fmt.Errorf("field '%s' failed to decode address: %w", key, err)
	}
	return nil
}
