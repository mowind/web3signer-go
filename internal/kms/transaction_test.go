package kms

import (
	"math/big"
	"testing"

	"github.com/umbracle/ethgo"
)

func TestParseTransactionSummary(t *testing.T) {
	tests := []struct {
		name           string
		tx             *ethgo.Transaction
		expectedTo     string
		expectedAmount string
		expectedToken  string
		shouldError    bool
	}{
		{
			name: "Legacy transaction with value",
			tx: &ethgo.Transaction{
				Type:     ethgo.TransactionLegacy,
				Nonce:    0,
				GasPrice: 20000000000,
				Gas:      21000,
				To:       addressPtr(ethgo.HexToAddress("0x2222222222222222222222222222222222222222")),
				Value:    big.NewInt(1000000000000000000),
			},
			expectedTo:     "0x2222222222222222222222222222222222222222",
			expectedAmount: "1000000000000000000",
			expectedToken:  "ETH",
			shouldError:    false,
		},
		{
			name: "EIP-1559 transaction",
			tx: &ethgo.Transaction{
				Type:                 ethgo.TransactionDynamicFee,
				Nonce:                1,
				MaxPriorityFeePerGas: big.NewInt(1500000000),
				MaxFeePerGas:         big.NewInt(2000000000),
				Gas:                  21000,
				To:                   addressPtr(ethgo.HexToAddress("0x3333333333333333333333333333333333333333")),
				Value:                big.NewInt(500000000000000000),
				ChainID:              big.NewInt(1),
			},
			expectedTo:     "0x3333333333333333333333333333333333333333",
			expectedAmount: "500000000000000000",
			expectedToken:  "ETH",
			shouldError:    false,
		},
		{
			name: "Contract creation (to is nil)",
			tx: &ethgo.Transaction{
				Type:     ethgo.TransactionLegacy,
				Nonce:    2,
				GasPrice: 20000000000,
				Gas:      100000,
				To:       nil,
				Value:    big.NewInt(2000000000000000000),
				ChainID:  big.NewInt(1),
			},
			expectedTo:     "",
			expectedAmount: "2000000000000000000",
			expectedToken:  "ETH",
			shouldError:    false,
		},
		{
			name: "Zero value transaction",
			tx: &ethgo.Transaction{
				Type:     ethgo.TransactionLegacy,
				Nonce:    3,
				GasPrice: 20000000000,
				Gas:      21000,
				To:       addressPtr(ethgo.HexToAddress("0x4444444444444444444444444444444444444444")),
				Value:    big.NewInt(0),
			},
			expectedTo:     "0x4444444444444444444444444444444444444444",
			expectedAmount: "0",
			expectedToken:  "ETH",
			shouldError:    false,
		},
		{
			name: "Large value transaction",
			tx: &ethgo.Transaction{
				Type:     ethgo.TransactionLegacy,
				Nonce:    4,
				GasPrice: 20000000000,
				Gas:      21000,
				To:       addressPtr(ethgo.HexToAddress("0x5555555555555555555555555555555555555555")),
				Value:    func() *big.Int { v, _ := new(big.Int).SetString("1000000000000000000000", 10); return v }(),
			},
			expectedTo:     "0x5555555555555555555555555555555555555555",
			expectedAmount: "1000000000000000000000",
			expectedToken:  "ETH",
			shouldError:    false,
		},
		{
			name: "EIP-2930 AccessList transaction",
			tx: &ethgo.Transaction{
				Type:     ethgo.TransactionAccessList,
				Nonce:    5,
				GasPrice: 20000000000,
				Gas:      21000,
				To:       addressPtr(ethgo.HexToAddress("0x6666666666666666666666666666666666666666")),
				Value:    big.NewInt(1000000000000000000),
				ChainID:  big.NewInt(1),
			},
			expectedTo:     "0x6666666666666666666666666666666666666666",
			expectedAmount: "1000000000000000000",
			expectedToken:  "ETH",
			shouldError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.tx.MarshalRLPTo(nil)
			if err != nil {
				t.Fatalf("Failed to encode transaction: %v", err)
			}

			summary, err := ParseTransactionSummary(encoded)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if summary.Type != string(SummaryTypeTransfer) {
				t.Errorf("Expected type TRANSFER, got %s", summary.Type)
			}

			if summary.To != tt.expectedTo {
				t.Errorf("Expected to %s, got %s", tt.expectedTo, summary.To)
			}

			if summary.Amount != tt.expectedAmount {
				t.Errorf("Expected amount %s, got %s", tt.expectedAmount, summary.Amount)
			}

			if summary.Token != tt.expectedToken {
				t.Errorf("Expected token %s, got %s", tt.expectedToken, summary.Token)
			}

			if summary.From != "" {
				t.Errorf("Expected from to be empty (not in RLP), got %s", summary.From)
			}
		})
	}
}

func TestParseTransactionSummary_InvalidData(t *testing.T) {
	tests := []struct {
		name        string
		txData      []byte
		shouldError bool
	}{
		{
			name:        "empty data",
			txData:      []byte{},
			shouldError: true,
		},
		{
			name:        "invalid RLP data",
			txData:      []byte{0x12, 0x34, 0x56},
			shouldError: true,
		},
		{
			name:        "nil data",
			txData:      nil,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTransactionSummary(tt.txData)

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestParseTransactionSummary_TypeSupport(t *testing.T) {
	tests := []struct {
		name        string
		txType      ethgo.TransactionType
		shouldError bool
	}{
		{
			name:        "Legacy type",
			txType:      ethgo.TransactionLegacy,
			shouldError: false,
		},
		{
			name:        "EIP-2930 AccessList type",
			txType:      ethgo.TransactionAccessList,
			shouldError: false,
		},
		{
			name:        "EIP-1559 DynamicFee type",
			txType:      ethgo.TransactionDynamicFee,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &ethgo.Transaction{
				Type:  tt.txType,
				Nonce: 0,
				Gas:   21000,
				To:    addressPtr(ethgo.HexToAddress("0x1111111111111111111111111111111111111111")),
				Value: big.NewInt(1000000000000000000),
			}

			switch tt.txType {
			case ethgo.TransactionLegacy:
				tx.GasPrice = 20000000000
			case ethgo.TransactionAccessList:
				tx.GasPrice = 20000000000
				tx.ChainID = big.NewInt(1)
			case ethgo.TransactionDynamicFee:
				tx.MaxPriorityFeePerGas = big.NewInt(1500000000)
				tx.MaxFeePerGas = big.NewInt(2000000000)
				tx.ChainID = big.NewInt(1)
			}

			encoded, err := tx.MarshalRLPTo(nil)
			if err != nil {
				t.Fatalf("Failed to encode transaction: %v", err)
			}

			summary, err := ParseTransactionSummary(encoded)

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.shouldError && summary == nil {
				t.Error("Expected summary but got nil")
			}
		})
	}
}

func addressPtr(addr ethgo.Address) *ethgo.Address {
	return &addr
}
