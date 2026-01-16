package signer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/umbracle/ethgo"
)

func TestTransactionBuilder_BuildLegacyTransaction(t *testing.T) {
	builder := NewTransactionBuilder()

	params := TransactionParams{
		From:     "0x1234567890123456789012345678901234567890",
		To:       "0x0987654321098765432109876543210987654321",
		Gas:      "21000",
		GasPrice: "20000000000",         // 20 Gwei
		Value:    "1000000000000000000", // 1 ETH
		Nonce:    "5",
		Data:     "",
	}

	tx, err := builder.BuildTransaction(params)
	if err != nil {
		t.Fatalf("Failed to build transaction: %v", err)
	}

	// 验证基本字段
	if tx.Nonce != 5 {
		t.Errorf("Expected nonce 5, got %d", tx.Nonce)
	}

	if tx.Gas != 21000 {
		t.Errorf("Expected gas 21000, got %d", tx.Gas)
	}

	if tx.GasPrice != 20000000000 {
		t.Errorf("Expected gasPrice 20000000000, got %d", tx.GasPrice)
	}

	if tx.Value.Cmp(big.NewInt(1000000000000000000)) != 0 {
		t.Errorf("Expected value 1 ETH, got %s", tx.Value.String())
	}

	expectedTo := ethgo.HexToAddress("0x0987654321098765432109876543210987654321")
	if tx.To == nil || *tx.To != expectedTo {
		t.Errorf("Expected to address %s, got %v", expectedTo.String(), tx.To)
	}
}

func TestTransactionBuilder_BuildEIP1559Transaction(t *testing.T) {
	builder := NewTransactionBuilder()

	params := TransactionParams{
		From:                 "0x1234567890123456789012345678901234567890",
		To:                   "0x0987654321098765432109876543210987654321",
		Gas:                  "21000",
		MaxFeePerGas:         "30000000000",         // 30 Gwei
		MaxPriorityFeePerGas: "2000000000",          // 2 Gwei
		Value:                "1000000000000000000", // 1 ETH
		Nonce:                "5",
		ChainID:              "1",
		Data:                 "",
	}

	tx, err := builder.BuildTransaction(params)
	if err != nil {
		t.Fatalf("Failed to build EIP-1559 transaction: %v", err)
	}

	// 验证 EIP-1559 字段
	if tx.MaxFeePerGas.Cmp(big.NewInt(30000000000)) != 0 {
		t.Errorf("Expected maxFeePerGas 30000000000, got %s", tx.MaxFeePerGas.String())
	}

	if tx.MaxPriorityFeePerGas.Cmp(big.NewInt(2000000000)) != 0 {
		t.Errorf("Expected maxPriorityFeePerGas 2000000000, got %s", tx.MaxPriorityFeePerGas.String())
	}

	if tx.ChainID.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Expected chainID 1, got %s", tx.ChainID.String())
	}

	if tx.GasPrice != 0 {
		t.Errorf("Expected gasPrice 0 for EIP-1559, got %d", tx.GasPrice)
	}
}

func TestTransactionBuilder_BuildEIP2930Transaction(t *testing.T) {
	builder := NewTransactionBuilder()

	params := TransactionParams{
		From:     "0x1234567890123456789012345678901234567890",
		To:       "0x0987654321098765432109876543210987654321",
		Gas:      "21000",
		GasPrice: "20000000000",
		Value:    "1000000000000000000",
		Nonce:    "5",
		ChainID:  "1",
		AccessList: []AccessListItem{
			{
				Address:     "0x1111111111111111111111111111111111111111",
				StorageKeys: []string{"0x0000000000000000000000000000000000000000000000000000000000000001"},
			},
		},
	}

	tx, err := builder.BuildTransaction(params)
	if err != nil {
		t.Fatalf("Failed to build EIP-2930 transaction: %v", err)
	}

	// 验证访问列表
	if len(tx.AccessList) != 1 {
		t.Fatalf("Expected 1 access list entry, got %d", len(tx.AccessList))
	}

	expectedAddr := ethgo.HexToAddress("0x1111111111111111111111111111111111111111")
	if tx.AccessList[0].Address != expectedAddr {
		t.Errorf("Expected access list address %s, got %s", expectedAddr.String(), tx.AccessList[0].Address.String())
	}

	if len(tx.AccessList[0].Storage) != 1 {
		t.Errorf("Expected 1 storage key, got %d", len(tx.AccessList[0].Storage))
	}

	expectedStorageKey := ethgo.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")
	if tx.AccessList[0].Storage[0] != expectedStorageKey {
		t.Errorf("Expected storage key %s, got %s", expectedStorageKey.String(), tx.AccessList[0].Storage[0].String())
	}
}

func TestTransactionBuilder_WithData(t *testing.T) {
	builder := NewTransactionBuilder()

	params := TransactionParams{
		From:     "0x1234567890123456789012345678901234567890",
		To:       "0x0987654321098765432109876543210987654321",
		Gas:      "50000",
		GasPrice: "20000000000",
		Value:    "0",
		Nonce:    "1",
		Data:     "0x6060604052341561000f576",
	}

	tx, err := builder.BuildTransaction(params)
	if err != nil {
		t.Fatalf("Failed to build transaction with data: %v", err)
	}

	if len(tx.Input) == 0 {
		t.Error("Expected non-empty input data")
	}
}

func TestTransactionBuilder_ContractCreation(t *testing.T) {
	builder := NewTransactionBuilder()

	params := TransactionParams{
		From:     "0x1234567890123456789012345678901234567890",
		To:       "", // 空地址表示合约创建
		Gas:      "300000",
		GasPrice: "20000000000",
		Value:    "0",
		Nonce:    "1",
		Data:     "0x6060604052341561000f576",
	}

	tx, err := builder.BuildTransaction(params)
	if err != nil {
		t.Fatalf("Failed to build contract creation transaction: %v", err)
	}

	// 对于合约创建，To 应该是零地址
	if tx.To == nil {
		t.Error("Expected To address for contract creation (should be zero address)")
	} else if *tx.To != ethgo.ZeroAddress {
		t.Errorf("Expected zero address for contract creation, got %s", tx.To.String())
	}
}

func TestParseTransactionParams(t *testing.T) {
	// 测试数组格式参数
	jsonParams := `[{
		"from": "0x1234567890123456789012345678901234567890",
		"to": "0x0987654321098765432109876543210987654321",
		"gas": "21000",
		"gasPrice": "20000000000",
		"value": "1000000000000000000"
	}]`

	var rawParams json.RawMessage
	if err := json.Unmarshal([]byte(jsonParams), &rawParams); err != nil {
		t.Fatalf("Failed to unmarshal test params: %v", err)
	}

	params, err := ParseTransactionParams(rawParams)
	if err != nil {
		t.Fatalf("Failed to parse transaction params: %v", err)
	}

	if params.From != "0x1234567890123456789012345678901234567890" {
		t.Errorf("Expected from address %s, got %s", "0x1234567890123456789012345678901234567890", params.From)
	}

	if params.To != "0x0987654321098765432109876543210987654321" {
		t.Errorf("Expected to address %s, got %s", "0x0987654321098765432109876543210987654321", params.To)
	}

	if params.Gas != "21000" {
		t.Errorf("Expected gas %s, got %s", "21000", params.Gas)
	}
}

func TestParseSignParams(t *testing.T) {
	jsonParams := `["0x1234567890123456789012345678901234567890", "0xdeadbeef"]`

	var rawParams json.RawMessage
	if err := json.Unmarshal([]byte(jsonParams), &rawParams); err != nil {
		t.Fatalf("Failed to unmarshal test params: %v", err)
	}

	address, data, err := ParseSignParams(rawParams)
	if err != nil {
		t.Fatalf("Failed to parse sign params: %v", err)
	}

	if address != "0x1234567890123456789012345678901234567890" {
		t.Errorf("Expected address %s, got %s", "0x1234567890123456789012345678901234567890", address)
	}

	expectedData, _ := hex.DecodeString("deadbeef")
	if !bytes.Equal(data, expectedData) {
		t.Errorf("Expected data %x, got %x", expectedData, data)
	}
}

func TestParseSignParams_InvalidParams(t *testing.T) {
	testCases := []struct {
		name       string
		jsonParams string
		expectErr  bool
	}{
		{
			name:       "empty array",
			jsonParams: `[]`,
			expectErr:  true,
		},
		{
			name:       "single parameter",
			jsonParams: `["0x1234567890123456789012345678901234567890"]`,
			expectErr:  true,
		},
		{
			name:       "invalid address type",
			jsonParams: `[123, "0xdeadbeef"]`,
			expectErr:  true,
		},
		{
			name:       "invalid data type",
			jsonParams: `["0x1234567890123456789012345678901234567890", 123]`,
			expectErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var rawParams json.RawMessage
			if err := json.Unmarshal([]byte(tc.jsonParams), &rawParams); err != nil {
				t.Fatalf("Failed to unmarshal test params: %v", err)
			}

			_, _, err := ParseSignParams(rawParams)
			if tc.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
