package signer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// ParseSignParams from JSON-RPC parameters parses signature parameters
//
// Parameters format: ["0xAddress", "0xData"]
func ParseSignParams(params json.RawMessage) (address string, data []byte, err error) {
	var paramsArray []interface{}
	if err := json.Unmarshal(params, &paramsArray); err != nil {
		return "", nil, fmt.Errorf("failed to parse sign params: %v", err)
	}

	if len(paramsArray) < 2 {
		return "", nil, fmt.Errorf("insufficient parameters for eth_sign")
	}

	// 第一个参数是地址
	address, ok := paramsArray[0].(string)
	if !ok {
		return "", nil, fmt.Errorf("invalid address parameter")
	}

	// 第二个参数是要签名的数据
	dataStr, ok := paramsArray[1].(string)
	if !ok {
		return "", nil, fmt.Errorf("invalid data parameter")
	}

	data, err = parseHex(dataStr)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse data: %v", err)
	}

	if len(data) != 32 {
		return "", nil, fmt.Errorf("invalid data length: expected 32 bytes, got %d", len(data))
	}

	return address, data, nil
}

// parseHex parses a hex string to bytes
func parseHex(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}

	// 移除 0x 前缀
	if len(s) >= 2 && s[0:2] == "0x" {
		s = s[2:]
	}

	// 如果长度为奇数，在前面补 0
	if len(s)%2 != 0 {
		s = "0" + s
	}

	return hex.DecodeString(s)
}
